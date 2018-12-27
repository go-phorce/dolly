package rest_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/metrics"
	"github.com/go-phorce/dolly/rest"
	"github.com/go-phorce/dolly/rest/tlsconfig"
	"github.com/go-phorce/dolly/testify/auditor"
	"github.com/go-phorce/dolly/xhttp/authz"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xhttp/identity"
	"github.com/go-phorce/dolly/xhttp/marshal"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projectPath = "../"

var tlsConnectionForAdmin = &tls.ConnectionState{
	PeerCertificates: []*x509.Certificate{
		{
			Subject: pkix.Name{
				CommonName:   "admin",
				Organization: []string{"go-phorce"},
			},
		},
	},
	VerifiedChains: [][]*x509.Certificate{
		{
			{
				Subject: pkix.Name{
					CommonName:   "[TEST] Root CA",
					Organization: []string{"go-phorce"},
				},
			},
		},
	},
}

var tlsConnectionForAdminUntrusted = &tls.ConnectionState{
	PeerCertificates: []*x509.Certificate{
		{
			Subject: pkix.Name{
				CommonName:   "admin",
				Organization: []string{"go-phorce"},
			},
		},
	},
	VerifiedChains: [][]*x509.Certificate{
		{
			{
				Subject: pkix.Name{
					CommonName:   "[TEST] Untrusted Root CA",
					Organization: []string{"go-phorce"},
				},
			},
		},
	},
}

var tlsConnectionForClient = &tls.ConnectionState{
	PeerCertificates: []*x509.Certificate{
		{
			Subject: pkix.Name{
				CommonName:   "client",
				Organization: []string{"go-phorce"},
			},
		},
	},
	VerifiedChains: [][]*x509.Certificate{
		{
			{
				Subject: pkix.Name{
					CommonName:   "[TEST] Root CA",
					Organization: []string{"go-phorce"},
				},
			},
		},
	},
}

var tlsConnectionForClientFromOtherOrg = &tls.ConnectionState{
	PeerCertificates: []*x509.Certificate{
		{
			Subject: pkix.Name{
				CommonName:   "client",
				Organization: []string{"someorg"},
			},
		},
	},
	VerifiedChains: [][]*x509.Certificate{
		{
			{
				Subject: pkix.Name{
					CommonName:   "[TEST] Root CA",
					Organization: []string{"go-phorce"},
				},
			},
		},
	},
}

func Test_NewServer(t *testing.T) {
	cfg := &serverConfig{
		BindAddr: ":8081",
	}

	audit := auditor.NewInMemory()

	server, err := rest.New("v1.0.123", "", cfg, nil, audit, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	require.NotNil(t, server.(rest.Server), "ensure interface")
	require.NoError(t, err)

	assert.NotNil(t, server.AddNode)
	assert.NotNil(t, server.RemoveNode)
	assert.NotNil(t, server.NodeName)
	assert.NotNil(t, server.LeaderID)
	assert.NotNil(t, server.NodeID)
	assert.NotNil(t, server.Version)
	assert.NotNil(t, server.HostName)
	assert.NotNil(t, server.LocalIP)
	assert.NotNil(t, server.Port)
	assert.NotNil(t, server.Protocol)
	assert.NotNil(t, server.StartedAt)
	assert.NotNil(t, server.Uptime)
	assert.NotNil(t, server.Service)
	assert.NotNil(t, server.IsReady)
	assert.NotNil(t, server.Audit)
	assert.NotNil(t, server.AddService)
	assert.NotNil(t, server.StartHTTP)
	assert.NotNil(t, server.StopHTTP)
	assert.NotNil(t, server.Scheduler)
	assert.NotNil(t, server.HTTPConfig)
	assert.NotNil(t, server.OnEvent)
	assert.NotEmpty(t, server.NodeName())
	assert.Empty(t, server.LeaderID())
	assert.Empty(t, server.NodeID())
	assert.NotEmpty(t, server.Version())
	assert.NotEmpty(t, server.HostName())
	assert.NotEmpty(t, server.LocalIP())
	assert.NotEmpty(t, server.Port())
	assert.Equal(t, "http", server.Protocol())
	assert.NotNil(t, server.StartedAt())
	assert.Nil(t, server.Service("abc"))
	assert.False(t, server.IsReady())
	assert.NotNil(t, server.Scheduler())
	assert.NotNil(t, server.HTTPConfig())
	assert.Equal(t, cfg, server.HTTPConfig())

	_, _, err = server.AddNode(nil, []string{"https://localhost:9443"})
	assert.Error(t, err)
	assert.Equal(t, "cluster not supported", err.Error())

	_, err = server.RemoveNode(nil, "https://localhost:9443")
	assert.Error(t, err)
	assert.Equal(t, "cluster not supported", err.Error())

	peersURLs, err := rest.GetNodeListenPeerURLs(server, server.NodeID())
	assert.Error(t, err)
	assert.Equal(t, "cluster not supported", err.Error())
	assert.Empty(t, peersURLs)

	assert.Equal(t, fmt.Sprintf("http://%s:8081", server.HostName()), rest.GetServerBaseURL(server).String())

	//	assert.NotNil(t, server.AddService())
	err = server.StartHTTP()
	require.NoError(t, err)
	e := audit.Find(rest.EvtSourceStatus, rest.EvtServiceStarted)
	require.NotNil(t, e)
	assert.Contains(t, e.Message, "ClientAuth=NoClientCert")

	for i := 0; i < 10 && !server.IsReady(); i++ {
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, server.IsReady())

	server.StopHTTP()
	e = audit.Find(rest.EvtSourceStatus, rest.EvtServiceStopped)
	require.NotNil(t, e)
}

func Test_ResolveTCPAddr(t *testing.T) {
	cfg := &serverConfig{
		ServiceName: "invalid",
		BindAddr:    "0-0-0-0",
	}

	server, err := rest.New("wrong", "", cfg, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	err = server.StartHTTP()
	require.Error(t, err)

	assert.Equal(t, `api=StartHTTP, reason=ResolveTCPAddr, service=invalid, bind="0-0-0-0": address 0-0-0-0: missing port in address`, err.Error())
}

func Test_GetServerURL(t *testing.T) {
	cfg := &serverConfig{
		BindAddr: "hostname:8081",
	}

	server, err := rest.New("wrong", "", cfg, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	t.Run("without XForwardedProto", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/get/GET", nil)
		require.NoError(t, err)

		u := rest.GetServerURL(server, r, "/another/location")
		require.NotNil(t, u)

		assert.Equal(t, fmt.Sprintf("%s://%s/another/location", server.Protocol(), cfg.BindAddr), u.String())
	})

	t.Run("with XForwardedProto", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/get/GET", nil)
		require.NoError(t, err)
		r.Header.Set(header.XForwardedProto, "https")

		u := rest.GetServerURL(server, r, "/another/location")
		require.NotNil(t, u)

		assert.Equal(t, "https://hostname:8081/another/location", u.String())
	})

	t.Run("with XForwardedProto", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/get/GET", nil)
		require.NoError(t, err)
		r.Header.Set(header.XForwardedProto, "https")
		r.Host = "localhost"

		u := rest.GetServerURL(server, r, "/another/location")
		require.NotNil(t, u)

		assert.Equal(t, "https://localhost/another/location", u.String())
	})
}

type cluster struct {
	this    int
	leader  int
	members []*rest.ClusterMember
}

// AddNode returns created node and a list of peers after adding the node to the cluster.
func (c *cluster) AddNode(_ context.Context, peerAddrs []string) (*rest.ClusterMember, []*rest.ClusterMember, error) {
	member := &rest.ClusterMember{
		ID:             guid.MustCreate(),
		Name:           fmt.Sprintf("node%d", 1+len(c.members)),
		ListenPeerURLs: peerAddrs,
	}
	c.members = append(c.members, member)
	return member, c.members, nil
}

// RemoveNode returns a list of peers after removing the node from the cluster.
func (c *cluster) RemoveNode(_ context.Context, nodeID string) ([]*rest.ClusterMember, error) {
	members := c.members
	for i, n := range c.members {
		if n.ID == nodeID {
			last := len(members) - 1
			members[i] = members[last]
			members[last] = nil
			c.members = members[:last]

			return c.members[:], nil
		}
	}

	return nil, errors.NotFoundf("node %s", nodeID)
}

func (c *cluster) NodeID() string {
	return c.members[c.this].ID
}

func (c *cluster) NodeName() string {
	return c.members[c.this].Name
}

func (c *cluster) LeaderID() string {
	return c.members[c.leader].ID
}

func (c *cluster) ClusterMembers() ([]*rest.ClusterMember, error) {
	return c.members[:], nil
}

func (c *cluster) ListenPeerURLs(nodeID string) ([]*url.URL, error) {
	return rest.GetNodeListenPeerURLs(c, nodeID)
}

func Test_ClusterInfo(t *testing.T) {
	cfg := &serverConfig{
		BindAddr: "hostname:8081",
	}

	clstr := &cluster{
		this:   0,
		leader: 0,
		members: []*rest.ClusterMember{
			{ID: "0000", Name: "node0", ListenPeerURLs: []string{"https://host0:8080", "https://127.0.0.1:8080"}},
			{ID: "1111", Name: "node1", ListenPeerURLs: []string{"https://host1:8081"}},
			{ID: "2222", Name: "node2", ListenPeerURLs: []string{"https://host2:8082"}},
		},
	}

	server, err := rest.New("v1.0.123", "", cfg, nil, nil, nil, clstr, clstr)
	require.NoError(t, err)
	require.NotNil(t, server)

	assert.Equal(t, "0000", server.LeaderID())

	l, err := rest.GetNodeListenPeerURLs(server, "0000")
	require.NoError(t, err)
	assert.Equal(t, 2, len(l))

	l, err = rest.GetNodeListenPeerURLs(server, "3333")
	require.Error(t, err)

	m, p, err := server.AddNode(nil, []string{"https://host2:8083"})
	require.NoError(t, err)
	assert.Equal(t, []string{"https://host2:8083"}, m.ListenPeerURLs)
	assert.Equal(t, 4, len(p))
	_, err = rest.GetNodeListenPeerURLs(server, m.ID)
	require.NoError(t, err)

	p, err = server.RemoveNode(nil, "2222")
	require.NoError(t, err)
	assert.Equal(t, 3, len(p))
	_, err = rest.GetNodeListenPeerURLs(server, "2222")
	require.Error(t, err)
}

type response struct {
	Method string
	Path   string
}

func Test_Authz(t *testing.T) {
	im := metrics.NewInmemSink(time.Minute, time.Minute)
	_, err := metrics.NewGlobal(metrics.DefaultConfig("authztest"), im)
	require.NoError(t, err)

	defer func() {
		md := im.Data()
		if len(md) > 0 {
			for k := range md[0].Gauges {
				t.Log("Gauge:", k)
			}
			for k := range md[0].Counters {
				t.Log("Counter:", k)
			}
			for k := range md[0].Samples {
				t.Log("Sample:", k)
			}
		}
	}()

	assertSample := func(key string) {
		md := im.Data()
		require.NotEqual(t, 0, len(md))

		_, exists := md[0].Samples[key]
		assert.True(t, exists, "sample metric not found: %s", key)
	}
	assertCounter := func(key string, expectedCount int) {
		md := im.Data()
		require.NotEqual(t, 0, len(md))

		s, exists := md[0].Counters[key]
		if assert.True(t, exists, "counter metric not found: %s", key) {
			assert.Equal(t, expectedCount, s.Count, "unexpected count for metric %s", key)
		}
	}

	tlsCfg, err := tlsconfig.NewServerTLSFromFiles(
		projectPath+"etc/dev/certs/test_dolly_server.pem",
		projectPath+"etc/dev/certs/test_dolly_server-key.pem",
		projectPath+"etc/dev/certs/rootca/test_dolly_root_CA.pem",
		tls.RequireAndVerifyClientCert,
	)
	require.NoError(t, err)

	cfg := &serverConfig{
		BindAddr: ":8081",
		Services: []string{"authztest"},
	}
	authz, err := authz.New(&authz.Config{
		Allow:                  []string{"/v1/allow:admin"},
		AllowAny:               []string{"/v1/allowany"},
		AllowAnyRole:           []string{"/v1/allowanyrole"},
		ValidOrganizations:     []string{"go-phorce"},
		ValidIssuerCommonNames: []string{"[TEST] Root CA"},
		LogAllowed:             true,
		LogDenied:              true,
	})
	require.NoError(t, err)

	clusterInfo := &cluster{
		this:   0,
		leader: 0,
		members: []*rest.ClusterMember{
			{ID: "0", Name: "localhost", ListenPeerURLs: []string{"https://localhost:8081"}},
		},
	}

	server, err := rest.New("v1.0.123", "127.0.0.1", cfg, tlsCfg, nil, authz, clusterInfo, clusterInfo)
	require.NoError(t, err)
	require.NotNil(t, server)

	service := newService(t, server, "authztest", false)
	server.AddService(service)

	err = server.StartHTTP()
	require.NoError(t, err)
	defer server.StopHTTP()
	time.Sleep(time.Second)

	t.Run("service not ready", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/v1/allowany", nil)
		server.ServeHTTP(w, r)

		assert.NotEmpty(t, w.Header().Get(header.XHostname))
		assert.NotEmpty(t, w.Header().Get(header.XCorrelationID))
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Equal(t, `{"code":"not_ready","message":"the service is not ready yet"}`, string(w.Body.Bytes()))
	})

	service.setReady()

	t.Run("connection is not over TLS", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/v1/allowany", nil)
		server.ServeHTTP(w, r)

		assert.NotEmpty(t, w.Header().Get(header.XHostname))
		assert.NotEmpty(t, w.Header().Get(header.XCorrelationID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("guest_to_allow_401", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/v1/allow", nil)
		r.TLS = tlsConnectionForClient
		server.ServeHTTP(w, r)
		assert.NotEmpty(t, w.Header().Get(header.XHostname))
		assert.NotEmpty(t, w.Header().Get(header.XCorrelationID))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, `{"code":"unauthorized","message":"the \"guest\" role is not allowed"}`, string(w.Body.Bytes()))

		assertCounter(fmt.Sprintf("authztest.http.request.status.failed;method=GET;role=guest;status=401;uri=/v1/allow"), 1)
	})

	identity.SetGlobalIdentityMapper(identityMapperFromCNMust)

	t.Run("must_have_TLS", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/v1/allow", nil)
		server.ServeHTTP(w, r)
		assert.NotEmpty(t, w.Header().Get(header.XHostname))
		assert.Empty(t, w.Header().Get(header.XCorrelationID))
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		assertCounter(fmt.Sprintf("authztest.http.request.status.failed;method=GET;role=guest;status=401;uri=/v1/allow"), 1)
	})

	identity.SetGlobalIdentityMapper(identityMapperFromCN)

	t.Run("admin_to_allow_200", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/v1/allow", nil)
		r.TLS = tlsConnectionForAdmin
		server.ServeHTTP(w, r)
		assert.NotEmpty(t, w.Header().Get(header.XHostname))
		assert.NotEmpty(t, w.Header().Get(header.XCorrelationID))
		assert.Equal(t, http.StatusOK, w.Code)

		assertCounter(fmt.Sprintf("authztest.http.request.status.successful;method=GET;role=admin;status=200;uri=/v1/allow"), 1)
		assertSample(fmt.Sprintf("authztest.http.request.perf;method=GET;role=admin;status=200;uri=/v1/allow"))
	})

	t.Run("untrusted_root_admin_to_allow_401", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/v1/allow", nil)
		r.TLS = tlsConnectionForAdminUntrusted
		server.ServeHTTP(w, r)
		assert.NotEmpty(t, w.Header().Get(header.XHostname))
		assert.NotEmpty(t, w.Header().Get(header.XCorrelationID))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, `{"code":"unauthorized","message":"the \"[TEST] Untrusted Root CA\" root CA is not allowed"}`, string(w.Body.Bytes()))

		assertCounter(fmt.Sprintf("authztest.http.request.status.failed;method=GET;role=guest;status=401;uri=/v1/allow"), 1)
	})

	t.Run("client_to_allow_401", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/v1/allow", nil)
		r.TLS = tlsConnectionForClient
		server.ServeHTTP(w, r)
		assert.NotEmpty(t, w.Header().Get(header.XHostname))
		assert.NotEmpty(t, w.Header().Get(header.XCorrelationID))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, `{"code":"unauthorized","message":"the \"client\" role is not allowed"}`, string(w.Body.Bytes()))

		assertCounter(fmt.Sprintf("authztest.http.request.status.failed;method=GET;role=guest;status=401;uri=/v1/allow"), 1)
	})

	t.Run("other_org_client_to_allow_401", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/v1/allow", nil)
		r.TLS = tlsConnectionForClientFromOtherOrg
		server.ServeHTTP(w, r)
		assert.NotEmpty(t, w.Header().Get(header.XHostname))
		assert.NotEmpty(t, w.Header().Get(header.XCorrelationID))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, `{"code":"unauthorized","message":"the \"someorg\" organization is not allowed"}`, string(w.Body.Bytes()))

		assertCounter(fmt.Sprintf("authztest.http.request.status.failed;method=GET;role=guest;status=401;uri=/v1/allow"), 1)
	})

	t.Run("client_to_allowany_200", func(t *testing.T) {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest(http.MethodGet, "/v1/allowany", nil)
		r.TLS = tlsConnectionForClient
		server.ServeHTTP(w, r)
		assert.NotEmpty(t, w.Header().Get(header.XHostname))
		assert.NotEmpty(t, w.Header().Get(header.XCorrelationID))
		assert.Equal(t, http.StatusOK, w.Code)

		assertCounter(fmt.Sprintf("authztest.http.request.status.successful;method=GET;role=client;status=200;uri=/v1/allowany"), 1)
		assertSample(fmt.Sprintf("authztest.http.request.perf;method=GET;role=client;status=200;uri=/v1/allowany"))
	})
}

func identityMapperFromCN(r *http.Request) (identity.Identity, error) {
	var role string
	var name string
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		name = identity.ClientIPFromRequest(r)
		role = "guest"
	} else {
		name = r.TLS.PeerCertificates[0].Subject.CommonName
		role = r.TLS.PeerCertificates[0].Subject.CommonName
	}
	return identity.NewIdentity(role, name), nil
}

func identityMapperFromCNMust(r *http.Request) (identity.Identity, error) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return nil, errors.New("missing client certificate")
	}
	return identity.NewIdentity(r.TLS.PeerCertificates[0].Subject.CommonName, r.TLS.PeerCertificates[0].Subject.CommonName), nil
}

type serviceX struct {
	t      *testing.T
	server rest.Server
	name   string
	ready  bool
}

// newService returns ane instances of the Status service
func newService(t *testing.T, server rest.Server, name string, ready bool) *serviceX {
	svc := &serviceX{
		t:      t,
		server: server,
		name:   name,
		ready:  ready,
	}
	return svc
}

func (s *serviceX) setReady() {
	s.ready = true
}

// Name returns the service name
func (s *serviceX) Name() string {
	return s.name
}

// IsReady indicates that the service is ready to serve its end-points
func (s *serviceX) IsReady() bool {
	return s.ready
}

// Close the subservices and it's resources
func (s *serviceX) Close() {
}

// Register adds the server Status API endpoints to the overall URL router
func (s *serviceX) Register(r rest.Router) {
	r.GET("/v1/allow", s.handle())
	r.GET("/v1/allowany", s.handle())
	r.GET("/v1/allowanyrole", s.handle())
}

func (s *serviceX) handle() rest.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ rest.Params) {
		s.t.Logf("serviceX: %s %s", r.Method, r.URL.Path)
		res := &response{
			Method: r.Method,
			Path:   r.URL.Path,
		}

		marshal.WriteJSON(w, r, res)
	}
}
