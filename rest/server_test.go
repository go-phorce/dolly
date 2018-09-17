package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authzConfig struct {
	// Allow will allow the specified roles access to this path and its children, in format: ${path}:${role},${role}
	Allow []string
	// AllowAny will allow any authenticated request access to this path and its children
	AllowAny []string
	// AllowAnyRole will allow any authenticated request that include a non empty role
	AllowAnyRole []string
}

// GetAllow will allow the specified roles access to this path and its children, in format: ${path}:${role},${role}
func (c *authzConfig) GetAllow() []string {
	return c.Allow
}

// GetAllowAny will allow any authenticated request access to this path and its children
func (c *authzConfig) GetAllowAny() []string {
	return c.AllowAny
}

// GetAllowAnyRole will allow any authenticated request that include a non empty role
func (c *authzConfig) GetAllowAnyRole() []string {
	return c.AllowAnyRole
}

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
	return c.CertFile
}

// GetKeyFile specifies location of the key
func (c *tlsConfig) GetKeyFile() string {
	return c.KeyFile
}

// GetTrustedCAFile specifies location of the CA file
func (c *tlsConfig) GetTrustedCAFile() string {
	return c.TrustedCAFile
}

// GetClientCertAuth controls client auth
func (c *tlsConfig) GetClientCertAuth() *bool {
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

	// Authz contains configuration for the API authorization layer
	Authz authzConfig

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

// GetAuthzCfg contains configuration for the API authorization layer
func (c *serverConfig) GetAuthzCfg() AuthzConfig {
	return &c.Authz
}

// GetHeartbeatSecs specifies heartbeat interval in seconds [30 secs is a minimum]
func (c *serverConfig) GetHeartbeatSecs() int {
	return c.HeartbeatSecs
}

func Test_NewServer(t *testing.T) {

	server, err := New("test", nil, &serverConfig{}, nil, "v1.0-123")
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
}
