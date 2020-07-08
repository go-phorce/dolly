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

	metricsutil "github.com/go-phorce/dolly/metrics/util"
	"github.com/go-phorce/dolly/netutil"
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

// ServerEvent specifies server event type
type ServerEvent int

const (
	// ServerStartedEvent is fired on server start
	ServerStartedEvent ServerEvent = iota
	// ServerStoppedEvent is fired after server stopped
	ServerStoppedEvent
)

// ServerEventFunc is a callback to handle server events
type ServerEventFunc func(evt ServerEvent)

// Server is an interface to provide server status
type Server interface {
	http.Handler
	Name() string
	Version() string
	HostName() string
	LocalIP() string
	Port() string
	Protocol() string
	StartedAt() time.Time
	Uptime() time.Duration
	Service(name string) Service
	HTTPConfig() HTTPServerConfig
	TLSConfig() *tls.Config

	// IsReady indicates that all subservices are ready to serve
	IsReady() bool

	// Audit records an auditable event.
	// 	source indicates the area that the event was triggered by
	// 	eventType indicates the specific event that occured
	// 	identity specifies the identity of the user that triggered this event, typically this is <role>/<cn>
	// 	contextID specifies the request ContextID that the event was triggered in [this can be used for cross service correlation of logs]
	// 	raftIndex indicates the index# of the raft log in RAFT that the event occured in [if applicable]
	// 	message contains any additional information about this event that is eventType specific
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

	OnEvent(evt ServerEvent, handler ServerEventFunc)
}

// MuxFactory creates http handlers.
type MuxFactory interface {
	NewMux() http.Handler
}

// HTTPServer is responsible for exposing the collection of the services
// as a single HTTP server
type HTTPServer struct {
	Server
	auditor     Auditor
	authz       Authz
	httpConfig  HTTPServerConfig
	tlsConfig   *tls.Config
	httpServer  *http.Server
	cors        *CORSOptions
	muxFactory  MuxFactory
	hostname    string
	port        string
	ipaddr      string
	version     string
	serving     bool
	startedAt   time.Time
	clientAuth  string
	scheduler   tasks.Scheduler
	services    map[string]Service
	evtHandlers map[ServerEvent][]ServerEventFunc
	lock        sync.RWMutex
}

// New creates a new instance of the server
func New(
	version string,
	ipaddr string,
	httpConfig HTTPServerConfig,
	tlsConfig *tls.Config,
) (*HTTPServer, error) {
	var err error

	if ipaddr == "" {
		ipaddr, err = netutil.GetLocalIP()
		if err != nil {
			ipaddr = "127.0.0.1"
			logger.Errorf("api=rest.New, reason=unable_determine_ipaddr, use=%q, err=[%v]", ipaddr, errors.ErrorStack(err))
		}
	}

	s := &HTTPServer{
		services:    map[string]Service{},
		scheduler:   tasks.NewScheduler(),
		startedAt:   time.Now().UTC(),
		version:     version,
		ipaddr:      ipaddr,
		evtHandlers: make(map[ServerEvent][]ServerEventFunc),
		clientAuth:  tlsClientAuthToStrMap[tls.NoClientCert],
		httpConfig:  httpConfig,
		hostname:    GetHostName(httpConfig.GetBindAddr()),
		port:        GetPort(httpConfig.GetBindAddr()),
		tlsConfig:   tlsConfig,
	}
	s.muxFactory = s
	if tlsConfig != nil {
		s.clientAuth = tlsClientAuthToStrMap[tlsConfig.ClientAuth]
	}

	return s, nil
}

// WithAuditor enables to use auditor
func (server *HTTPServer) WithAuditor(auditor Auditor) *HTTPServer {
	server.auditor = auditor
	return server
}

// WithAuthz enables to use Authz
func (server *HTTPServer) WithAuthz(authz Authz) *HTTPServer {
	server.authz = authz
	return server
}

// WithScheduler enables to use custom scheduler
func (server *HTTPServer) WithScheduler(scheduler tasks.Scheduler) *HTTPServer {
	server.scheduler = scheduler
	return server
}

// WithCORS enables CORS options
func (server *HTTPServer) WithCORS(cors *CORSOptions) *HTTPServer {
	server.cors = cors
	return server
}

var tlsClientAuthToStrMap = map[tls.ClientAuthType]string{
	tls.NoClientCert:               "NoClientCert",
	tls.RequestClientCert:          "RequestClientCert",
	tls.RequireAnyClientCert:       "RequireAnyClientCert",
	tls.VerifyClientCertIfGiven:    "VerifyClientCertIfGiven",
	tls.RequireAndVerifyClientCert: "RequireAndVerifyClientCert",
}

// AddService provides a service registration for the server
func (server *HTTPServer) AddService(s Service) {
	server.lock.Lock()
	defer server.lock.Unlock()
	server.services[s.Name()] = s
}

// OnEvent accepts a callback to handle server events
func (server *HTTPServer) OnEvent(evt ServerEvent, handler ServerEventFunc) {
	server.lock.Lock()
	defer server.lock.Unlock()

	server.evtHandlers[evt] = append(server.evtHandlers[evt], handler)
}

// Scheduler returns task scheduler for the server
func (server *HTTPServer) Scheduler() tasks.Scheduler {
	return server.scheduler
}

// Service returns a registered server
func (server *HTTPServer) Service(name string) Service {
	server.lock.Lock()
	defer server.lock.Unlock()
	return server.services[name]
}

// HostName returns the host name of the server
func (server *HTTPServer) HostName() string {
	return server.hostname
}

// Port returns the port name of the server
func (server *HTTPServer) Port() string {
	return server.port
}

// Protocol returns the protocol
func (server *HTTPServer) Protocol() string {
	if server.tlsConfig != nil {
		return "https"
	}
	return "http"
}

// LocalIP returns the IP address of the server
func (server *HTTPServer) LocalIP() string {
	return server.ipaddr
}

// StartedAt returns the time when the server started
func (server *HTTPServer) StartedAt() time.Time {
	return server.startedAt
}

// Uptime returns the duration the server was up
func (server *HTTPServer) Uptime() time.Duration {
	return time.Now().UTC().Sub(server.startedAt)
}

// Version returns the version of the server
func (server *HTTPServer) Version() string {
	return server.version
}

// Name returns the server name
func (server *HTTPServer) Name() string {
	return server.httpConfig.GetServiceName()
}

// HTTPConfig returns HTTPServerConfig
func (server *HTTPServer) HTTPConfig() HTTPServerConfig {
	return server.httpConfig
}

// TLSConfig returns TLSConfig
func (server *HTTPServer) TLSConfig() *tls.Config {
	return server.tlsConfig
}

// IsReady returns true when the server is ready to serve
func (server *HTTPServer) IsReady() bool {
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

// Audit create an audit event
func (server *HTTPServer) Audit(source string,
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

// WithMuxFactory requires the server to use `muxFactory` to create server handler.
func (server *HTTPServer) WithMuxFactory(muxFactory MuxFactory) {
	server.muxFactory = muxFactory
}

// StartHTTP will verify all the TLS related files are present and start the actual HTTPS listener for the server
func (server *HTTPServer) StartHTTP() error {
	bindAddr := server.httpConfig.GetBindAddr()
	var err error

	// Main server
	if _, err = net.ResolveTCPAddr("tcp", bindAddr); err != nil {
		return errors.Annotatef(err, "api=StartHTTP, reason=ResolveTCPAddr, service=%s, bind=%q",
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

	httpHandler := server.muxFactory.NewMux()

	if server.httpConfig.GetAllowProfiling() {
		if httpHandler, err = xhttp.NewRequestProfiler(httpHandler, server.httpConfig.GetProfilerDir(), nil, xhttp.LogProfile()); err != nil {
			return errors.Trace(err)
		}
	}

	server.httpServer.Handler = httpHandler

	serve := func() error {
		server.serving = true
		if httpsListener != nil {
			return server.httpServer.Serve(httpsListener)
		}
		return server.httpServer.ListenAndServe()
	}

	go func() {
		for _, handler := range server.evtHandlers[ServerStartedEvent] {
			handler(ServerStartedEvent)
		}

		logger.Infof("api=StartHTTP, service=%s, port=%v, status=starting, protocol=%s",
			server.Name(), bindAddr, server.Protocol())

		// this is a blocking call to serve
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

	if server.scheduler != nil {
		if server.httpConfig.GetHeartbeatSecs() > 0 {
			task := tasks.NewTaskAtIntervals(uint64(server.httpConfig.GetHeartbeatSecs()), tasks.Seconds).
				Do("hearbeat", hearbeatMetricsTask, server)
			server.Scheduler().Add(task)
			task.Run()

			task = tasks.NewTaskAtIntervals(60, tasks.Seconds).Do("uptime", uptimeMetricsTask, server)
			server.Scheduler().Add(task)
			task.Run()
		}

		server.scheduler.Start()
	}

	server.Audit(
		EvtSourceStatus,
		EvtServiceStarted,
		server.HostName(),
		server.LocalIP(),
		0,
		fmt.Sprintf("address=%q, ClientAuth=%s",
			strings.TrimPrefix(bindAddr, ":"), server.clientAuth),
	)

	return nil
}

func hearbeatMetricsTask(server *HTTPServer) {
	metricsutil.PublishHeartbeat(server.httpConfig.GetServiceName())
}

func uptimeMetricsTask(server *HTTPServer) {
	metricsutil.PublishUptime(server.httpConfig.GetServiceName(), server.Uptime())
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
func (server *HTTPServer) StopHTTP() {
	if server.scheduler != nil {
		// stop scheduled tasks
		server.scheduler.Stop()
	}

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

	for _, handler := range server.evtHandlers[ServerStoppedEvent] {
		handler(ServerStoppedEvent)
	}

	ut := server.Uptime() / time.Second * time.Second
	server.Audit(
		EvtSourceStatus,
		EvtServiceStopped,
		server.HostName(),
		server.LocalIP(),
		0,
		fmt.Sprintf("uptime=%s", ut),
	)
}

// NewMux creates a new http handler for the http server, typically you only
// need to call this directly for tests.
func (server *HTTPServer) NewMux() http.Handler {
	var router Router
	if server.cors != nil {
		router = NewRouterWithCORS(notFoundHandler, server.cors)
	} else {
		router = NewRouter(notFoundHandler)
	}

	for _, f := range server.services {
		f.Register(router)
	}
	logger.Debugf("api=NewMux, service=%s, service_count=%d",
		server.Name(), len(server.services))

	var err error
	httpHandler := router.Handler()

	logger.Infof("api=NewMux, service=%s, ClientAuth=%s", server.Name(), server.clientAuth)

	if server.authz != nil {
		httpHandler, err = server.authz.NewHandler(httpHandler)
		if err != nil {
			panic(errors.ErrorStack(err))
		}
	}

	// logging wrapper
	httpHandler = xhttp.NewRequestLogger(httpHandler, server.Name(), serverExtraLogger, time.Millisecond, server.httpConfig.GetPackageLogger())

	// metrics wrapper
	httpHandler = xhttp.NewRequestMetrics(httpHandler)

	// service ready
	httpHandler = ready.NewServiceStatusVerifier(server, httpHandler)

	// role/contextID wrapper
	httpHandler = identity.NewContextHandler(httpHandler)
	return httpHandler
}

// ServeHTTP should write reply headers and data to the ResponseWriter
// and then return. Returning signals that the request is finished; it
// is not valid to use the ResponseWriter or read from the
// Request.Body after or concurrently with the completion of the
// ServeHTTP call.
func (server *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.httpServer.Handler.ServeHTTP(w, r)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	marshal.WriteJSON(w, r, httperror.WithNotFound(r.URL.Path))
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

	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	if host == "" {
		host = s.HostName() + ":" + s.Port()
	}

	return &url.URL{
		Scheme: proto,
		Host:   host,
		Path:   relativeEndpoint,
	}
}

// GetServerBaseURL returns server base URL
func GetServerBaseURL(s Server) *url.URL {
	return &url.URL{
		Scheme: s.Protocol(),
		Host:   s.HostName() + ":" + s.Port(),
	}
}
