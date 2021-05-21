package rest

import (
	"os"
	"strings"
)

// TLSInfoConfig contains configuration info for the TLS
type TLSInfoConfig interface {
	// GetCertFile returns location of the cert
	GetCertFile() string
	// GetKeyFile returns location of the key
	GetKeyFile() string
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
	GetDisabled() bool
	// VIPName is the FQ name of the VIP to the cluster [this is used when building the cert requests]
	GetVIPName() string
	// BindAddr is the address that the HTTPS service should be exposed on
	GetBindAddr() string
	// PackageLogger if set, specifies name of the package logger
	GetPackageLogger() string
	// AllowProfiling if set, will allow for per request CPU/Memory profiling triggered by the URI QueryString
	GetAllowProfiling() bool
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
