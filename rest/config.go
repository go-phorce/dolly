package rest

import (
	"net/http"
	"os"
	"strings"
)

// RoleMapper abstracts how a role is extracted from an HTTP request
// Your role mapper can be called concurrently by multiple go-routines so should
// be careful if it manages any state.
type RoleMapper func(r *http.Request) string

// Authz represents an Authorization provider interface,
// You can call Allow or AllowAny to specify which roles are allowed
// access to which path segments.
// once configured you can create a http.Handler that enforces that
// configuration for you by calling NewHandler
type Authz interface {
	// SetRoleMapper configures the function that provides the mapping from an HTTP request to a role name
	SetRoleMapper(m RoleMapper)
	// NewHandler returns a http.Handler that enforces the current authorization configuration
	// The handler has its own copy of the configuration changes to the Provider after calling
	// NewHandler won't affect previously created Handlers.
	// The returned handler will extract the role and verify that the role has access to the
	// URI being request, and either return an error, or pass the request on to the supplied
	// delegate handler
	NewHandler(delegate http.Handler) (http.Handler, error)
}

// TLSInfoConfig contains configuration info for the TLS
type TLSInfoConfig interface {
	// GetCertFile returns location of the cert
	GetCertFile() string
	// GetKeyFile returns location of the key
	GetKeyFile() string
	// GetCABundleFile returns location of the CA bundle file.
	// If CA bundle is provided, then intermediate CA issuers will be included TLS response.
	GetCABundleFile() string
	// TrustedCAFile specifies location of the Trusted CA file
	GetTrustedCAFile() string
	// ClientCertAuth controls client auth
	GetClientCertAuth() *bool
}

// HTTPServerConfig contains the configuration of the HTTPS API Service
type HTTPServerConfig interface {
	// ServiceName specifies name of the service: HTTP|HTTPS|WebAPI
	GetServiceName() string
	// Disabled specifies if the service is disabled
	GetDisabled() *bool
	// VIPName is the FQ name of the VIP to the cluster [this is used when building the cert requests]
	GetVIPName() string
	// BindAddr is the address that the HTTPS service should be exposed on
	GetBindAddr() string
	// PackageLogger if set, specifies name of the package logger
	GetPackageLogger() string
	// AllowProfiling if set, will allow for per request CPU/Memory profiling triggered by the URI QueryString
	GetAllowProfiling() *bool
	// ProfilerDir specifies the directories where per-request profile information is written, if not set will write to a TMP dir
	GetProfilerDir() string
	// Services is a list of services to enable for this HTTP Service
	GetServices() []string
	// HeartbeatSecs specifies heartbeat GetHeartbeatSecserval in seconds [30 secs is a minimum]
	GetHeartbeatSecs() int
}

// GetPort returns the port from HTTP bind address,
// or standard HTTPS 443 port, if it's not specified in the config
func GetPort(bindAddr string) string {
	i := strings.LastIndex(bindAddr, ":")
	if i >= 0 {
		return bindAddr[i+1:]
	}
	return "443"
}

// GetHostName returns Hostname from HTTP bind address,
// or OS Hostname, if it's not specified in the config
func GetHostName(bindAddr string) string {
	hn := bindAddr
	i := strings.LastIndex(bindAddr, ":")
	if i >= 0 {
		hn = bindAddr[:i]
	}
	if hn == "" {
		hn, _ = os.Hostname()
	}
	return hn
}
