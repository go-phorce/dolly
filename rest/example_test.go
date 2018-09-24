package rest_test

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/metrics"
	"github.com/go-phorce/dolly/rest"
	"github.com/go-phorce/dolly/rest/tlsconfig"
	"github.com/go-phorce/dolly/tasks"
	"github.com/go-phorce/dolly/testify/auditor"
	"github.com/go-phorce/dolly/testify/testca"
	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/juju/errors"
	"go.uber.org/dig"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "rest_test")

func ExampleServer() {
	sigs := make(chan os.Signal, 2)

	tlsCfg, err := newTLSConfig(true)
	if err != nil {
		panic("unable to create TLS config")
	}

	tlsInfo, tlsloader, err := createTLSInfo(tlsCfg)
	if err != nil {
		panic("unable to create TLS config")
	}
	defer tlsloader.Close()

	cfg := &serverConfig{
		BindAddr: ":8080",
	}

	ioc := dig.New()
	ioc.Provide(func() rest.HTTPServerConfig {
		return cfg
	})
	ioc.Provide(func() rest.Auditor {
		return auditor.NewInMemory()
	})
	ioc.Provide(func() *tls.Config {
		return tlsInfo
	})
	server, err := rest.New("test", "v1.0.123", ioc)
	if err != nil {
		panic("unable to create the server")
	}

	// execute and schedule
	go certExpirationPublisherTask(tlsloader)
	server.Scheduler().Add(tasks.NewTaskAtIntervals(1, tasks.Hours).Do("servertls", certExpirationPublisherTask, tlsloader))

	svc := NewService(server)
	server.AddService(svc)

	err = server.StartHTTP()
	if err != nil {
		panic("unable to start the server")
	}

	go func() {
		// Send STOP signal after few seconds,
		// in production the service should listen to
		// os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGUSR2, syscall.SIGABRT events
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

func certExpirationPublisherTask(tlsloader *tlsconfig.KeypairReloader) {
	certFile, keyFile := tlsloader.CertAndKeyFiles()

	logger.Tracef("api=certExpirationPublisherTask, cert='%s', key='%s'", certFile, keyFile)

	tlsloader.Reload()
	pair := tlsloader.Keypair()
	if pair != nil {
		cert, err := x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			errors.Annotatef(err, "api=certExpirationPublisherTask, reason=unable_parse_tls_cert, file='%s'", certFile)
		} else {
			metrics.PublishCertExpirationInDays(cert, "server")
		}
	} else {
		logger.Warningf("api=certExpirationPublisherTask, reason=Keypair, cert='%s', key='%s'", certFile, keyFile)
	}
}

var trueVal = true

func newTLSConfig(withClientAuth bool) (*tlsConfig, error) {
	var (
		ca = testca.NewEntity(
			testca.Authority,
			testca.Subject(pkix.Name{
				CommonName: "[TEST] Root CA",
			}),
			testca.KeyUsage(x509.KeyUsageCertSign|x509.KeyUsageCRLSign|x509.KeyUsageDigitalSignature),
		)
		inter = ca.Issue(
			testca.Authority,
			testca.Subject(pkix.Name{
				CommonName: "[TEST] Issuing CA Level 1",
			}),
			testca.KeyUsage(x509.KeyUsageCertSign|x509.KeyUsageCRLSign|x509.KeyUsageDigitalSignature),
		)
		leaf = inter.Issue(
			testca.Subject(pkix.Name{
				CommonName: "localhost",
			}),
			testca.ExtKeyUsage(x509.ExtKeyUsageServerAuth),
			testca.ExtKeyUsage(x509.ExtKeyUsageClientAuth),
		)
	)

	tmpDir := filepath.Join(os.TempDir(), "tests", "rest", guid.MustCreate())
	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		return nil, errors.Trace(err)
	}

	tlsConfig := &tlsConfig{
		CertFile:      filepath.Join(tmpDir, "test-server.pem"),
		KeyFile:       filepath.Join(tmpDir, "test-server-key.pem"),
		TrustedCAFile: filepath.Join(tmpDir, "test-rootca.pem"),
	}

	if withClientAuth {
		tlsConfig.ClientCertAuth = &trueVal
	}

	fkey, err := os.Create(tlsConfig.KeyFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer fkey.Close()
	fkey.Write(testca.PrivKeyToPEM(leaf.PrivateKey))

	fcert, err := os.Create(tlsConfig.CertFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer fcert.Close()
	certutil.EncodeAllToPEM(fcert, append([]*x509.Certificate{}, leaf.Certificate, inter.Certificate), true)

	froot, err := os.Create(tlsConfig.TrustedCAFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer fcert.Close()
	certutil.EncodeToPEM(froot, ca.Certificate, true)

	return tlsConfig, nil
}

func createTLSInfo(cfg *tlsConfig) (*tls.Config, *tlsconfig.KeypairReloader, error) {
	withClientAuth := cfg.GetClientCertAuth()
	certFile := cfg.GetCertFile()
	keyFile := cfg.GetKeyFile()

	tls, err := tlsconfig.BuildFromFiles(certFile, keyFile, cfg.GetTrustedCAFile(), withClientAuth != nil && *withClientAuth)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "api=createTLSInfo, reason=BuildFromFiles, cert='%s', key='%s'",
			certFile, keyFile)
	}

	tlsloader, err := tlsconfig.NewKeypairReloader(certFile, keyFile, 5*time.Second)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "api=createTLSInfo, reason=NewKeypairReloader, cert='%s', key='%s'",
			certFile, keyFile)
	}
	tls.GetCertificate = tlsloader.GetKeypairFunc()

	return tls, tlsloader, nil
}
