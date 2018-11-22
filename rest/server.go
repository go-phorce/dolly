package rest

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-phorce/dolly/metrics"
	"github.com/go-phorce/dolly/netutil"
	"github.com/go-phorce/dolly/rest/container"
	"github.com/go-phorce/dolly/rest/ready"
	"github.com/go-phorce/dolly/tasks"
	"github.com/go-phorce/dolly/xhttp"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/go-phorce/dolly/xhttp/identity"
	"github.com/go-phorce/dolly/xhttp/marshal"
	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "rest")

// MaxRequestSize specifies max size of regular HTTP Post requests in bytes, 64 Mb
const MaxRequestSize = 64 * 1024 * 1024

const (
	// EvtSourceStatus specifies source for service Status
	EvtSourceStatus = "status"
	// EvtServiceStarted specifies Service Started event
	EvtServiceStarted = "service started"
	// EvtServiceStopped specifies Service Stopped event
	EvtServiceStopped = "service stopped"
)

// ClusterMember provides information about cluster member
type ClusterMember struct {
	// ID is the member ID for this member.
	ID string `protobuf:"bytes,1,opt,name=ID,proto3" json:"id,omitempty"`
	// Name is the human-readable name of the member. If the member is not started, the name will be an empty string.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// PeerURLs is the list of URLs the member exposes to the cluster for communication.
	PeerURLs []string `protobuf:"bytes,3,rep,name=peers" json:"peers,omitempty"`
}

// ClusterInfo is an interface to provide basic info about the cluster
type ClusterInfo interface {
	// NodeID returns the ID of the node in the cluster
	NodeID() string

	NodeName() string

	// LeaderID returns the ID of the leader
	LeaderID() string

	// ClusterMembers returns the list of members in the cluster
	ClusterMembers() ([]*ClusterMember, error)

	NodeHostName(nodeID string) (string, error)
}

// Server is an interface to provide server status
type Server interface {
	ClusterInfo
	Name() string
	Version() string
	RoleName() string
	HostName() string
	LocalIP() string
	Port() string
	Protocol() string
	StartedAt() time.Time
	Uptime() time.Duration
	Service(name string) Service
	HTTPConfig() HTTPServerConfig

	// IsReady indicates that all subservices are ready to serve
	IsReady() bool

	// Call Event to record a new Auditable event
	// Audit event
	// source indicates the area that the event was triggered by
	// eventType indicates the specific event that occured
	// identity specifies the identity of the user that triggered this event, typically this is <role>/<cn>
	// contextID specifies the request ContextID that the event was triggered in [this can be used for cross service correlation of logs]
	// raftIndex indicates the index# of the raft log in RAFT that the event occured in [if applicable]
	// message contains any additional information about this event that is eventType specific
	Audit(source string,
		eventType string,
		identity string,
		contextID string,
		raftIndex uint64,
		message string)

	AddService(s Service)
	StartHTTP() error
	StopHTTP()

	Scheduler() tasks.Scheduler

	// Invoke runs the given function after instantiating its dependencies.
	//
	// Any arguments that the function has are treated as its dependencies. The
	// dependencies are instantiated in an unspecified order along with any
	// dependencies that they might have.
	//
	// The function may return an error to indicate failure. The error will be
	// returned to the caller as-is.
	Invoke(function interface{}) error
}

// server is responsible for exposing the collection of the services
// as a single HTTP server
type server struct {
	Server
	container      container.Container
	auditor        Auditor
	authz          Authz
	cluster        ClusterInfo
	httpConfig     HTTPServerConfig
	tlsConfig      *tls.Config
	httpServer     *http.Server
	rolename       string
	hostname       string
	port           string
	ipaddr         string
	version        string
	serving        bool
	startedAt      time.Time
	withClientAuth bool
	scheduler      tasks.Scheduler
	services       map[string]Service
	lock           sync.RWMutex
}

// ensure implements interface
var _ Server = &server{}

// New creates a new instance of the server
func New(
	rolename string,
	version string,
	container container.Container,
) (Server, error) {
	var err error
	ipaddr, err := netutil.GetLocalIP()
	if err != nil {
		ipaddr = "127.0.0.1"
		logger.Errorf("api=rest.New, reason=unable_determine_ipaddr, use=%q, err=[%v]", ipaddr, errors.ErrorStack(err))
	}

	if container == nil {
		logger.Panic("container parameter is required")
	}

	s := &server{
		services:  map[string]Service{},
		scheduler: tasks.NewScheduler(),
		rolename:  rolename,
		startedAt: time.Now().UTC(),
		version:   version,
		ipaddr:    ipaddr,
		container: container,
	}

	err = container.Invoke(func(httpConfig HTTPServerConfig) {
		s.httpConfig = httpConfig
		baddr := httpConfig.GetBindAddr()
		s.hostname = GetHostName(baddr)
		s.port = GetPort(baddr)
	})
	if err != nil {
		return nil, errors.Errorf("HTTPServerConfig not provided, rolename=%s, err=%q",
			rolename, err.Error())
	}

	err = container.Invoke(func(authz Authz) {
		s.authz = authz
	})
	if err != nil {
		logger.Warningf("api=rest.New, reason='failed to initialize Authz', service=%s, err=%q",
			s.httpConfig.GetServiceName(), err.Error())
	}

	err = container.Invoke(func(cluster ClusterInfo) {
		s.cluster = cluster
	})
	if err != nil {
		logger.Warningf("api=rest.New, reason='ClusterInfo not provided', service=%s, err=%q",
			s.httpConfig.GetServiceName(), err.Error())
	}

	err = container.Invoke(func(auditor Auditor) {
		s.auditor = auditor
	})
	if err != nil {
		logger.Warningf("api=rest.New, reason='Auditor not provided', service=%s, err=%q",
			s.httpConfig.GetServiceName(), err.Error())
	}

	err = container.Invoke(func(tlsConfig *tls.Config) {
		s.tlsConfig = tlsConfig
		if tlsConfig != nil {
			s.withClientAuth = tlsConfig.ClientAuth == tls.RequireAndVerifyClientCert
		}
	})
	if err != nil {
		logger.Warningf("api=rest.New, reason='tls.Config not provided', service=%s, err=%q",
			s.httpConfig.GetServiceName(), err.Error())
	}

	return s, nil
}

// AddService provides a service registration for the server
func (server *server) AddService(s Service) {
	server.lock.Lock()
	defer server.lock.Unlock()
	server.services[s.Name()] = s
}

// Scheduler returns task scheduler for the server
func (server *server) Scheduler() tasks.Scheduler {
	return server.scheduler
}

// Service returns a registered server
func (server *server) Service(name string) Service {
	server.lock.Lock()
	defer server.lock.Unlock()
	return server.services[name]
}

// RoleName returns the name of the server role
func (server *server) RoleName() string {
	return server.rolename
}

// HostName returns the host name of the server
func (server *server) HostName() string {
	return server.hostname
}

// NodeName returns the node name in the cluster
func (server *server) NodeName() string {
	if server.cluster != nil {
		return server.cluster.NodeName()
	}
	return server.HostName()
}

// Port returns the port name of the server
func (server *server) Port() string {
	return server.port
}

// Protocol returns the protocol
func (server *server) Protocol() string {
	if server.tlsConfig != nil {
		return "https"
	}
	return "http"
}

// LocalIP returns the IP address of the server
func (server *server) LocalIP() string {
	return server.ipaddr
}

// StartedAt returns the time when the server started
func (server *server) StartedAt() time.Time {
	return server.startedAt
}

// Uptime returns the duration the server was up
func (server *server) Uptime() time.Duration {
	return time.Now().UTC().Sub(server.startedAt)
}

// Version returns the version of the server
func (server *server) Version() string {
	return server.version
}

// Name returns the server name
func (server *server) Name() string {
	return server.httpConfig.GetServiceName()
}

func (server *server) HTTPConfig() HTTPServerConfig {
	return server.httpConfig
}

func (server *server) NodeID() string {
	if server.cluster == nil {
		return ""
	}
	return server.cluster.NodeID()
}

func (server *server) LeaderID() string {
	if server.cluster == nil {
		return ""
	}
	return server.cluster.LeaderID()
}

func (server *server) ClusterMembers() ([]*ClusterMember, error) {
	if server.cluster == nil {
		return nil, errors.NotSupportedf("cluster")
	}
	return server.cluster.ClusterMembers()
}

// NodeHostName returns the host name of specific node
func (server *server) NodeHostName(nodeID string) (string, error) {
	if server.cluster == nil {
		return "", errors.NotSupportedf("cluster")
	}
	return server.cluster.NodeHostName(nodeID)
}

// IsReady returns true when the server is ready to serve
func (server *server) IsReady() bool {
	if !server.serving {
		return false
	}
	for _, ss := range server.services {
		if !ss.IsReady() {
			return false
		}
	}
	return true
}

func (server *server) Invoke(function interface{}) error {
	err := server.container.Invoke(function)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Audit create an audit event
func (server *server) Audit(source string,
	eventType string,
	identity string,
	contextID string,
	raftIndex uint64,
	message string) {
	if server.auditor != nil {
		server.auditor.Audit(source, eventType, identity, contextID, raftIndex, message)
	} else {
		// {contextID}:{identity}:{raftIndex}:{source}:{type}:{message}
		logger.Infof("audit:%s:%s:%s:%s:%d:%s\n",
			source, eventType, identity, contextID, raftIndex, message)
	}
}

// StartHTTP will verify all the TLS related files are present and start the actual HTTPS listener for the server
func (server *server) StartHTTP() error {
	bindAddr := server.httpConfig.GetBindAddr()
	var err error

	// Main server
	if _, err = net.ResolveTCPAddr("tcp", bindAddr); err != nil {
		return errors.Annotatef(err, "api=StartHTTP, reason=ResolveTCPAddr, service=%s, addr=%q",
			server.Name(), bindAddr)
	}

	server.httpServer = &http.Server{
		IdleTimeout: time.Hour * 2,
		ErrorLog:    xlog.Stderr,
	}

	var httpsListener net.Listener

	if server.tlsConfig != nil {
		// Start listening on main server over TLS
		httpsListener, err = tls.Listen("tcp", bindAddr, server.tlsConfig)
		if err != nil {
			return errors.Annotatef(err, "api=StartHTTP, reason=unable_listen, service=%s, address=%q",
				server.Name(), bindAddr)
		}

		server.httpServer.TLSConfig = server.tlsConfig
	} else {
		server.httpServer.Addr = bindAddr
	}

	readyHandler := ready.NewServiceStatusVerifier(server, server.NewMux())

	if server.httpConfig.GetAllowProfiling() {
		if readyHandler, err = xhttp.NewRequestProfiler(readyHandler, server.httpConfig.GetProfilerDir(), nil, xhttp.LogProfile()); err != nil {
			return errors.Trace(err)
		}
	}

	server.httpServer.Handler = readyHandler

	serve := func() error {
		server.serving = true
		if httpsListener != nil {
			return server.httpServer.Serve(httpsListener)
		}
		return server.httpServer.ListenAndServe()
	}

	go func() {
		logger.Infof("api=StartHTTP, service=%s, port=%v, status=starting, protocol=%s",
			server.Name(), bindAddr, server.Protocol())

		if err := serve(); err != nil {
			server.serving = false
			//panic, only if not Serve error while stopping the server,
			// which is a valid error
			if netutil.IsAddrInUse(err) || err != http.ErrServerClosed {
				logger.Panicf("api=StartHTTP, service=%s, err=[%v]", server.Name(), errors.Trace(err))
			}
			logger.Warningf("api=StartHTTP, service=%s, status=stopped, reason=[%s]", server.Name(), err.Error())
		}
	}()

	if server.httpConfig.GetHeartbeatSecs() > 0 {
		task := tasks.NewTaskAtIntervals(uint64(server.httpConfig.GetHeartbeatSecs()), tasks.Seconds).
			Do("hearbeat", hearbeatTask, server)
		server.Scheduler().Add(task)
		task.Run()

		task = tasks.NewTaskAtIntervals(60, tasks.Seconds).Do("uptime", uptimeTask, server)
		server.Scheduler().Add(task)
		task.Run()
	}

	server.scheduler.Start()
	server.Audit(
		EvtSourceStatus,
		EvtServiceStarted,
		server.NodeName(),
		server.NodeID(),
		0,
		fmt.Sprintf("node=%q, address=%q, ClientAuth=%t",
			server.NodeName(), strings.TrimPrefix(bindAddr, ":"), server.withClientAuth),
	)

	return nil
}

func hearbeatTask(server *server) {
	metrics.PublishHeartbeat(server.httpConfig.GetServiceName())
}

func uptimeTask(server *server) {
	metrics.PublishUptime(server.httpConfig.GetServiceName(), server.Uptime())
}

// StopHTTP will perform a graceful shutdown of the serivce by
//		1) signally to the Load Balancer to remove this instance from the pool
//				by changing to response to /availability
//		2) cause new responses to have their Connection closed when finished
//				to force clients to re-connect [hopefully to a different instance]
//		3) wait the minShutdownTime to ensure the LB has noticed the status change
//		4) wait for existing requests to finish processing
//		5) step 4 is capped by a overrall timeout where we'll give up waiting
//			 for the requests to complete and will exit.
//
// it is expected that you don't try and use the server instance again
// after this. [i.e. if you want to start it again, create another server instance]
func (server *server) StopHTTP() {
	// stop scheduled tasks
	server.scheduler.Stop()

	// close services
	for _, f := range server.services {
		logger.Tracef("api=StopHTTP, service=%q", f.Name())
		f.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := server.httpServer.Shutdown(ctx)
	if err != nil {
		logger.Errorf("api=StopHTTP, reason=Shutdown, err=[%v]", errors.ErrorStack(err))
	}

	ut := server.Uptime() / time.Second * time.Second
	server.Audit(
		EvtSourceStatus,
		EvtServiceStopped,
		server.NodeName(),
		server.NodeID(),
		0,
		fmt.Sprintf("node=%s, uptime=%s", server.NodeName(), ut),
	)
}

// NewMux creates a new http handler for the http server, typically you only
// need to call this directly for tests.
func (server *server) NewMux() http.Handler {
	router := NewRouter(server.notFoundHandler)

	for _, f := range server.services {
		f.Register(router)
	}
	logger.Debugf("api=NewMux, service=%s, service_count=%d",
		server.Name(), len(server.services))

	var err error
	httpHandler := router.Handler()

	logger.Infof("api=NewMux, service=%s, withClientAuth=%t", server.Name(), server.withClientAuth)

	if server.authz != nil {
		// authz wrapper
		server.authz.SetRoleMapper(func(r *http.Request) string {
			return identity.ForRequest(r).Identity().Role()
		})

		httpHandler, err = server.authz.NewHandler(httpHandler)
		if err != nil {
			panic(errors.ErrorStack(err))
		}
		// TODO: only allow configured certs
		// httpHandler = authz.NewClientCertVerifier(httpHandler)
	}

	// logging wrapper
	httpHandler = xhttp.NewRequestLogger(httpHandler, server.rolename, serverExtraLogger, time.Millisecond, server.httpConfig.GetPackageLogger())

	// metrics wrapper
	httpHandler = xhttp.NewRequestMetrics(httpHandler)

	// role/contextID wrapper
	httpHandler = identity.NewContextHandler(httpHandler)
	return httpHandler
}

func (server *server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	marshal.WriteJSON(w, r, httperror.New(http.StatusNotFound, httperror.NotFound, "URL: %s", r.RequestURI))
}

func serverExtraLogger(resp *xhttp.ResponseCapture, req *http.Request) []string {
	return []string{identity.ForRequest(req).CorrelationID()}
}

// GetServerURL returns complete server URL for given relative end-point
func GetServerURL(s Server, r *http.Request, relativeEndpoint string) *url.URL {
	proto := s.Protocol()

	// Allow upstream proxies  to specify the forwarded protocol. Allow this value
	// to override our own guess.
	if specifiedProto := r.Header.Get(header.XForwardedProto); specifiedProto != "" {
		proto = specifiedProto
	}

	host := r.Host
	if host == "" {
		host = s.HostName() + ":" + s.Port()
	}

	return &url.URL{
		Scheme: proto,
		Host:   host,
		Path:   relativeEndpoint,
	}
}
