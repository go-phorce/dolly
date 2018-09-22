package rest

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-phorce/dolly/testify/auditor"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tlsConfig struct {
	// CertFile specifies location of the cert
	CertFile string
	// KeyFile specifies location of the key
	KeyFile string
	// TrustedCAFile specifies location of the CA file
	TrustedCAFile string
	// ClientCertAuth controls client auth
	ClientCertAuth *bool
}

// GetCertFile specifies location of the cert
func (c *tlsConfig) GetCertFile() string {
	if c == nil {
		return ""
	}
	return c.CertFile
}

// GetKeyFile specifies location of the key
func (c *tlsConfig) GetKeyFile() string {
	if c == nil {
		return ""
	}
	return c.KeyFile
}

// GetTrustedCAFile specifies location of the CA file
func (c *tlsConfig) GetTrustedCAFile() string {
	if c == nil {
		return ""
	}
	return c.TrustedCAFile
}

// GetClientCertAuth controls client auth
func (c *tlsConfig) GetClientCertAuth() *bool {
	if c == nil {
		return nil
	}
	return c.ClientCertAuth
}

type serverConfig struct {

	// ServiceName specifies name of the service: HTTP|HTTPS|WebAPI
	ServiceName string

	// Disabled specifies if the service is disabled
	Disabled *bool

	// VIPName is the FQ name of the VIP to the cluster [this is used when building the cert requests]
	VIPName string

	// BindAddr is the address that the HTTPS service should be exposed on
	BindAddr string

	// ServerTLS provides TLS config for server
	ServerTLS tlsConfig

	// PackageLogger if set, specifies name of the package logger
	PackageLogger string

	// AllowProfiling if set, will allow for per request CPU/Memory profiling triggered by the URI QueryString
	AllowProfiling *bool

	// ProfilerDir specifies the directories where per-request profile information is written, if not set will write to a TMP dir
	ProfilerDir string

	// Services is a list of services to enable for this HTTP Service
	Services []string

	// HeartbeatSecs specifies heartbeat interval in seconds [30 secs is a minimum]
	HeartbeatSecs int
}

// GetServiceName specifies name of the service: HTTP|HTTPS|WebAPI
func (c *serverConfig) GetServiceName() string {
	return c.ServiceName
}

// GetDisabled specifies if the service is disabled
func (c *serverConfig) GetDisabled() *bool {
	return c.Disabled
}

// GetVIPName is the FQ name of the VIP to the cluster [this is used when building the cert requests]
func (c *serverConfig) GetVIPName() string {
	return c.VIPName
}

// GetBindAddr is the address that the HTTPS service should be exposed on
func (c *serverConfig) GetBindAddr() string {
	return c.BindAddr
}

// GetServerTLSCfg provides TLS config for server
func (c *serverConfig) GetServerTLSCfg() TLSInfoConfig {
	return &c.ServerTLS
}

// GetPackageLogger if set, specifies name of the package logger
func (c *serverConfig) GetPackageLogger() string {
	return c.PackageLogger
}

// GetAllowProfiling if set, will allow for per request CPU/Memory profiling triggered by the URI QueryString
func (c *serverConfig) GetAllowProfiling() *bool {
	return c.AllowProfiling
}

// GetProfilerDir specifies the directories where per-request profile information is written, if not set will write to a TMP dir
func (c *serverConfig) GetProfilerDir() string {
	return c.ProfilerDir
}

// GetServices is a list of services to enable for this HTTP Service
func (c *serverConfig) GetServices() []string {
	return c.Services
}

// GetHeartbeatSecs specifies heartbeat interval in seconds [30 secs is a minimum]
func (c *serverConfig) GetHeartbeatSecs() int {
	return c.HeartbeatSecs
}

func ExampleServer() {
	sigs := make(chan os.Signal, 2)

	cfg := &serverConfig{
		BindAddr: ":8080",
	}

	audit := auditor.NewInMemory()
	server, err := New("test", audit, nil, cfg, nil, nil, "v1.0.123")
	if err != nil {
		logger.Panicf("unable to create the server: %v", errors.ErrorStack(err))
	}

	err = server.StartHTTP()
	if err != nil {
		logger.Panicf("unable to start the server: %v", errors.ErrorStack(err))
	}

	go func() {
		// Send STOP signal after 3 seconds,
		time.Sleep(3 * time.Second)
		sigs <- syscall.SIGTERM
	}()

	// register for signals, and wait to be shutdown
	signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGUSR2, syscall.SIGABRT)
	// Block until a signal is received.
	sig := <-sigs

	server.StopHTTP()

	// SIGUSR2 is triggered by the upstart pre-stop script, we don't want
	// to actually exit the process in that case until upstart sends SIGTERM
	if sig == syscall.SIGUSR2 {
		select {
		case <-time.After(time.Second * 5):
			logger.Info("api=startService, status='service shutdown from SIGUSR2 complete, waiting for SIGTERM to exit'")
		case sig = <-sigs:
			logger.Infof("api=startService, status=exiting, reason=received_signal, sig=%v", sig)
		}
	}
}

func Test_NewServer(t *testing.T) {
	cfg := &serverConfig{
		BindAddr: ":8080",
	}

	audit := auditor.NewInMemory()
	server, err := New("test", audit, nil, cfg, nil, nil, "v1.0.123")
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
	assert.NotNil(t, server.StartedAt())
	assert.NotNil(t, server.LocalCtx())
	assert.Nil(t, server.Service("abc"))
	assert.True(t, server.IsReady())
	assert.NotNil(t, server.Scheduler())

	//	assert.NotNil(t, server.AddService())
	err = server.StartHTTP()
	require.NoError(t, err)
	e := audit.Find(EvtSourceStatus, EvtServiceStarted)
	require.NotNil(t, e)
	assert.Contains(t, e.Message, "ClientAuth=false")

	server.StopHTTP()
	e = audit.Find(EvtSourceStatus, EvtServiceStopped)
	require.NotNil(t, e)
}
