package rest_test

import (
	"testing"
	"time"

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
	assert.NotNil(t, server.LocalCtx)
	assert.NotNil(t, server.Service)
	assert.NotNil(t, server.IsReady)
	assert.NotNil(t, server.Audit)
	assert.NotNil(t, server.AddService)
	assert.NotNil(t, server.StartHTTP)
	assert.NotNil(t, server.StopHTTP)
	assert.NotNil(t, server.Scheduler)

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
	assert.NotNil(t, server.LocalCtx())
	assert.Nil(t, server.Service("abc"))
	assert.False(t, server.IsReady())
	assert.NotNil(t, server.Scheduler())

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
