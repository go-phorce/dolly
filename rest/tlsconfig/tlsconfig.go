package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/go-phorce/xlog"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/pkg", "rest/tls")

// BuildFromFiles will build a tls.Config from the supplied certificate, key
// and optional trust roots files, these files are all expected to be PEM encoded.
// The file paths are relative to the working directory if not specified in absolute
// format.
// rootsFile is optional, if not specified the standard OS CA roots will be used.
func BuildFromFiles(certFile, keyFile, rootsFile string, withClientAuth bool) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
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

	clientauthType := tls.NoClientCert
	if withClientAuth {
		clientauthType = tls.RequireAndVerifyClientCert
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"h2", "http/1.1"},
		RootCAs:      roots,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   clientauthType,
		ClientCAs:    roots,
	}, nil
}
