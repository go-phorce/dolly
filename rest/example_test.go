package rest_test

import (
	"crypto/x509"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	metricsutil "github.com/go-phorce/dolly/metrics/util"
	"github.com/go-phorce/dolly/rest"
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
		TrustedCAFile:  "testdata/test-server-rootca.pem",
		WithClientAuth: false,
	}

	tlsInfo, tlsloader, err := createServerTLSInfo(tlsCfg)
	if err != nil {
		panic("unable to create TLS config")
	}
	defer tlsloader.Close()

	cfg := &serverConfig{
		BindAddr: ":8181",
	}

	server, err := rest.New("v1.0.123", "", cfg, tlsInfo)
	if err != nil {
		panic("unable to create the server")
	}
	server.WithAuditor(auditor.NewInMemory())

	// execute and schedule
	go certExpirationPublisherTask(tlsloader)
	server.Scheduler().Add(tasks.NewTaskAtIntervals(1, tasks.Hours).Do("servertls", certExpirationPublisherTask, tlsloader))

	svc := NewService(server)
	server.AddService(svc)

	fmt.Println("starting server")
	err = server.StartHTTP()
	if err != nil {
		panic("unable to start the server: " + errors.ErrorStack(err))
	}

	go func() {
		// Send STOP signal after few seconds,
		// in production the service should listen to
		// os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGUSR2, syscall.SIGABRT events
		time.Sleep(3 * time.Second)
		fmt.Println("sending syscall.SIGTERM signal")
		sigs <- syscall.SIGTERM
	}()

	// register for signals, and wait to be shutdown
	signal.Notify(sigs, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGUSR2, syscall.SIGABRT)
	// Block until a signal is received.
	sig := <-sigs

	server.StopHTTP()
	fmt.Println("stopped server")

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

	// Output:
	// starting server
	// sending syscall.SIGTERM signal
	// stopped server
}

func certExpirationPublisherTask(tlsloader *tlsconfig.KeypairReloader) {
	certFile, keyFile := tlsloader.CertAndKeyFiles()

	logger.Tracef("api=certExpirationPublisherTask, cert=%q, key=%q", certFile, keyFile)

	tlsloader.Reload()
	pair := tlsloader.Keypair()
	if pair != nil {
		cert, err := x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			errors.Annotatef(err, "api=certExpirationPublisherTask, reason=unable_parse_tls_cert, file=%q", certFile)
		} else {
			metricsutil.PublishCertExpirationInDays(cert, "server")
		}
	} else {
		logger.Warningf("api=certExpirationPublisherTask, reason=Keypair, cert=%q, key=%q", certFile, keyFile)
	}
}
