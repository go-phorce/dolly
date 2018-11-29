package rest_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/go-phorce/dolly/xhttp/header"

	"github.com/go-phorce/dolly/rest"
	"github.com/go-phorce/dolly/rest/container"
	"github.com/go-phorce/dolly/testify/auditor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewServer(t *testing.T) {
	cfg := &serverConfig{
		BindAddr: ":8081",
	}

	ioc := container.New()

	t.Run("IoC without HTTPConfig should fail", func(t *testing.T) {
		server, err := rest.New("test", "v1.0.123", ioc)
		assert.Error(t, err)
		assert.Nil(t, server)
	})

	ioc.Provide(func() rest.HTTPServerConfig {
		return cfg
	})

	audit := auditor.NewInMemory()
	ioc.Provide(func() rest.Auditor {
		return audit
	})

	server, err := rest.New("test", "v1.0.123", ioc)
	require.NoError(t, err)
	require.NotNil(t, server)

	require.NotNil(t, server.(rest.Server), "ensure interface")
	err = server.Invoke(func(c rest.HTTPServerConfig) {
		require.NotNil(t, c)
	})
	require.NoError(t, err)

	assert.NotNil(t, server.NodeName)
	assert.NotNil(t, server.LeaderID)
	assert.NotNil(t, server.NodeID)
	assert.NotNil(t, server.PeerURLs)
	assert.NotNil(t, server.Version)
	assert.NotNil(t, server.RoleName)
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

	assert.NotEmpty(t, server.NodeName())
	assert.Empty(t, server.LeaderID())
	assert.Empty(t, server.NodeID())
	assert.NotEmpty(t, server.Version())
	assert.NotEmpty(t, server.RoleName())
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

	peersURLs, err := server.PeerURLs(server.NodeID())
	assert.Error(t, err)
	assert.Equal(t, "cluster not supported", err.Error())
	assert.Empty(t, peersURLs)

	//	assert.NotNil(t, server.AddService())
	err = server.StartHTTP()
	require.NoError(t, err)
	e := audit.Find(rest.EvtSourceStatus, rest.EvtServiceStarted)
	require.NotNil(t, e)
	assert.Contains(t, e.Message, "ClientAuth=false")

	for i := 0; i < 10 && !server.IsReady(); i++ {
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, server.IsReady())

	server.StopHTTP()
	e = audit.Find(rest.EvtSourceStatus, rest.EvtServiceStopped)
	require.NotNil(t, e)
}

func Test_GetServerURL(t *testing.T) {
	cfg := &serverConfig{
		BindAddr: "hostname:8081",
	}

	ioc := container.New()
	ioc.Provide(func() rest.HTTPServerConfig {
		return cfg
	})

	server, err := rest.New("test", "v1.0.123", ioc)
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

func (c *cluster) PeerURLs(nodeID string) ([]*url.URL, error) {
	return rest.GetNodePeerURLs(c, nodeID)
}

func Test_ClusterInfo(t *testing.T) {
	cfg := &serverConfig{
		BindAddr: "hostname:8081",
	}

	ioc := container.New()
	ioc.Provide(func() rest.HTTPServerConfig {
		return cfg
	})

	ioc.Provide(func() rest.ClusterInfo {
		return &cluster{
			this:   0,
			leader: 0,
			members: []*rest.ClusterMember{
				{ID: "0000", Name: "node0", PeerURLs: []string{"https://host0:8080", "https://127.0.0.1:8080"}},
				{ID: "1111", Name: "node1", PeerURLs: []string{"https://host1:8081"}},
				{ID: "2222", Name: "node2", PeerURLs: []string{"https://host2:8082"}},
			},
		}
	})

	server, err := rest.New("test", "v1.0.123", ioc)
	require.NoError(t, err)
	require.NotNil(t, server)

	l, err := rest.GetNodePeerURLs(server, "0000")
	require.NoError(t, err)
	assert.Equal(t, 2, len(l))

	l, err = rest.GetNodePeerURLs(server, "3333")
	require.Error(t, err)
}
