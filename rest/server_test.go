package rest_test

import (
	"fmt"
	"net/http"
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

	assert.NotNil(t, server.NodeName)
	assert.NotNil(t, server.LeaderID)
	assert.NotNil(t, server.NodeID)
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
