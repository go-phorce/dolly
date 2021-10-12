package rest_test

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/rest/tlsconfig"
	"github.com/go-phorce/dolly/testify/testca"
	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
)

type testSuite struct {
	suite.Suite
	tmpDir         string
	serverRootFile string
	serverCertFile string
	serverKeyFile  string
	clientRootFile string
	clientCertFile string
	clientKeyFile  string
	rootsFile      string
}

func Test_RestSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupTest() {
	xlog.SetGlobalLogLevel(xlog.DEBUG)

	s.tmpDir = filepath.Join(os.TempDir(), "tests", "rest", guid.MustCreate())
	err := os.MkdirAll(s.tmpDir, os.ModePerm)
	s.Require().NoError(err)

	// Chain for Server
	var (
		ca1 = testca.NewEntity(
			testca.Authority,
			testca.Subject(pkix.Name{
				CommonName: "[TEST] Root CA One",
			}),
			testca.KeyUsage(x509.KeyUsageCertSign|x509.KeyUsageCRLSign|x509.KeyUsageDigitalSignature),
		)
		inter1 = ca1.Issue(
			testca.Authority,
			testca.Subject(pkix.Name{
				CommonName: "[TEST] Issuing CA One Level 1",
			}),
			testca.KeyUsage(x509.KeyUsageCertSign|x509.KeyUsageCRLSign|x509.KeyUsageDigitalSignature),
		)
		srv = inter1.Issue(
			testca.Subject(pkix.Name{
				CommonName: "localhost",
			}),
			testca.ExtKeyUsage(x509.ExtKeyUsageServerAuth),
			testca.DNSName("localhost", "127.0.0.1"),
		)
	)

	// Chain for Client
	var (
		ca2 = testca.NewEntity(
			testca.Authority,
			testca.Subject(pkix.Name{
				CommonName: "[TEST] Root CA Two",
			}),
			testca.KeyUsage(x509.KeyUsageCertSign|x509.KeyUsageCRLSign|x509.KeyUsageDigitalSignature),
		)
		inter2 = ca2.Issue(
			testca.Authority,
			testca.Subject(pkix.Name{
				CommonName: "[TEST] Issuing CA Two Level 1",
			}),
			testca.KeyUsage(x509.KeyUsageCertSign|x509.KeyUsageCRLSign|x509.KeyUsageDigitalSignature),
		)
		cli = inter2.Issue(
			testca.Subject(pkix.Name{
				CommonName: "localhost",
			}),
			testca.ExtKeyUsage(x509.ExtKeyUsageClientAuth),
		)
	)

	s.serverCertFile = filepath.Join(s.tmpDir, "test-server.pem")
	s.serverKeyFile = filepath.Join(s.tmpDir, "test-server-key.pem")
	s.serverRootFile = filepath.Join(s.tmpDir, "test-server-rootca.pem")
	s.clientCertFile = filepath.Join(s.tmpDir, "test-client.pem")
	s.clientKeyFile = filepath.Join(s.tmpDir, "test-client-key.pem")
	s.clientRootFile = filepath.Join(s.tmpDir, "test-client-rootca.pem")
	s.rootsFile = filepath.Join(s.tmpDir, "test-roots.pem")

	//
	// save keys
	//
	fkey, err := os.Create(s.serverKeyFile)
	s.Require().NoError(err)
	defer fkey.Close()
	fkey.Write(testca.PrivKeyToPEM(srv.PrivateKey))

	fkey, err = os.Create(s.clientKeyFile)
	s.Require().NoError(err)
	defer fkey.Close()
	fkey.Write(testca.PrivKeyToPEM(cli.PrivateKey))

	//
	// save server certs
	//
	fcert, err := os.Create(s.serverCertFile)
	s.Require().NoError(err)
	defer fcert.Close()
	certutil.EncodeToPEM(fcert, true, srv.Certificate, inter1.Certificate)

	fcert, err = os.Create(s.serverRootFile)
	s.Require().NoError(err)
	defer fcert.Close()
	certutil.EncodeToPEM(fcert, true, ca1.Certificate)

	//
	// save client certs
	//
	fcert, err = os.Create(s.clientCertFile)
	s.Require().NoError(err)
	defer fcert.Close()
	certutil.EncodeToPEM(fcert, true, cli.Certificate, inter2.Certificate)

	fcert, err = os.Create(s.clientRootFile)
	s.Require().NoError(err)
	defer fcert.Close()
	certutil.EncodeToPEM(fcert, true, ca2.Certificate)

	//
	// save CA certs
	//
	fcert, err = os.Create(s.rootsFile)
	s.Require().NoError(err)
	defer fcert.Close()
	certutil.EncodeToPEM(fcert, true, ca1.Certificate, ca2.Certificate)
}

func (s *testSuite) TearDownTest() {
	os.RemoveAll(s.tmpDir)
}

type tlsConfig struct {
	// CertFile specifies location of the cert
	CertFile string
	// KeyFile specifies location of the key
	KeyFile string
	// TrustedCAFile specifies location of the CA file
	TrustedCAFile string
	// WithClientAuth controls client auth
	WithClientAuth bool
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
func (c *tlsConfig) GetClientCertAuth() bool {
	if c == nil {
		return false
	}
	return c.WithClientAuth
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
func (c *serverConfig) GetDisabled() bool {
	return c.Disabled != nil && *c.Disabled
}

// GetVIPName is the FQ name of the VIP to the cluster [this is used when building the cert requests]
func (c *serverConfig) GetVIPName() string {
	return c.VIPName
}

// GetBindAddr is the address that the HTTPS service should be exposed on
func (c *serverConfig) GetBindAddr() string {
	return c.BindAddr
}

// GetPackageLogger if set, specifies name of the package logger
func (c *serverConfig) GetPackageLogger() string {
	return c.PackageLogger
}

// GetAllowProfiling if set, will allow for per request CPU/Memory profiling triggered by the URI QueryString
func (c *serverConfig) GetAllowProfiling() bool {
	return c.AllowProfiling != nil && *c.AllowProfiling
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

func createServerTLSInfo(cfg *tlsConfig) (*tls.Config, *tlsconfig.KeypairReloader, error) {
	certFile := cfg.GetCertFile()
	keyFile := cfg.GetKeyFile()

	clientauthType := tls.VerifyClientCertIfGiven
	if cfg.GetClientCertAuth() {
		clientauthType = tls.RequireAndVerifyClientCert
	}

	tls, err := tlsconfig.NewServerTLSFromFiles(certFile, keyFile, cfg.GetTrustedCAFile(), clientauthType)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "reason=BuildFromFiles, cert=%q, key=%q",
			certFile, keyFile)
	}

	tlsloader, err := tlsconfig.NewKeypairReloader(certFile, keyFile, 5*time.Second)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "reason=NewKeypairReloader, cert=%q, key=%q",
			certFile, keyFile)
	}
	tls.GetCertificate = tlsloader.GetKeypairFunc()

	return tls, tlsloader, nil
}
