package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "rest/tls")

// NewServerTLSFromFiles will build a tls.Config from the supplied certificate, key
// and optional trust roots files, these files are all expected to be PEM encoded.
// The file paths are relative to the working directory if not specified in absolute
// format.
// caBundle is optional.
// rootsFile is optional, if not specified the standard OS CA roots will be used.
func NewServerTLSFromFiles(certFile, keyFile, rootsFile string, clientauthType tls.ClientAuthType) (*tls.Config, error) {
	tlscert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var roots *x509.CertPool

	if rootsFile != "" {
		rootsBytes, err := ioutil.ReadFile(rootsFile)
		if err != nil {
			return nil, errors.Trace(err)
		}

		roots = x509.NewCertPool()
		roots.AppendCertsFromPEM(rootsBytes)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"h2", "http/1.1"},
		Certificates: []tls.Certificate{tlscert},
		ClientAuth:   clientauthType,
		ClientCAs:    roots,
		RootCAs:      roots,
	}, nil
}

// NewClientTLSFromFiles will build a tls.Config from the supplied certificate, key
// and optional trust roots files, these files are all expected to be PEM encoded.
// The file paths are relative to the working directory if not specified in absolute
// format.
// caBundle is optional.
// rootsFile is optional, if not specified the standard OS CA roots will be used.
func NewClientTLSFromFiles(certFile, keyFile, rootsFile string) (*tls.Config, error) {
	var roots *x509.CertPool

	if rootsFile != "" {
		rootsBytes, err := ioutil.ReadFile(rootsFile)
		if err != nil {
			return nil, errors.Trace(err)
		}

		roots = x509.NewCertPool()
		roots.AppendCertsFromPEM(rootsBytes)
	}

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
		//Certificates: []tls.Certificate{tlscert},
		ClientCAs: roots,
		RootCAs:   roots,
	}

	if certFile != "" {
		tlscert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, errors.Trace(err)
		}
		cfg.Certificates = []tls.Certificate{tlscert}
	}

	return cfg, nil
}
