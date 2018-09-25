package rest_test

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-phorce/dolly/metrics"
	"github.com/go-phorce/dolly/rest"
	"github.com/go-phorce/dolly/rest/container"
	"github.com/go-phorce/dolly/rest/tlsconfig"
	"github.com/go-phorce/dolly/tasks"
	"github.com/go-phorce/dolly/testify/auditor"
	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "rest_test")

func ExampleServer() {
	sigs := make(chan os.Signal, 2)

	tlsCfg := &tlsConfig{
		CertFile:       "testdata/test-server.pem",
		KeyFile:        "testdata/test-server-key.pem",
		CABundleFile:   "testdata/cabundle.pem",
		TrustedCAFile:  "testdata/test-rootca.pem",
		WithClientAuth: false,
	}

	tlsInfo, tlsloader, err := createServerTLSInfo(tlsCfg)
	if err != nil {
		panic("unable to create TLS config")
	}
	defer tlsloader.Close()

	cfg := &serverConfig{
		BindAddr: ":8080",
	}

	ioc := container.New()
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
