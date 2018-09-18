package certutil

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/juju/errors"
)

// LoadFromPEM returns Certificate loaded from the file
func LoadFromPEM(certFile string) (*x509.Certificate, error) {
	bytes, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, errors.Trace(err)
	}

	cert, err := ParseFromPEM(bytes)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return cert, nil
}

// ParseFromPEM returns Certificate parsed from PEM
func ParseFromPEM(bytes []byte) (*x509.Certificate, error) {
	var block *pem.Block
	block, bytes = pem.Decode(bytes)
	if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
		return nil, errors.Errorf("api=LoadFromPEM, reason=invalid_pem")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Annotatef(err, "api=LoadFromPEM, reason=ParseCertificate")
	}

	return cert, nil
}

// ParseChainFromPEM returns Certificates parsed from PEM
func ParseChainFromPEM(certificateChainPem []byte) ([]*x509.Certificate, error) {
	list := make([]*x509.Certificate, 0)
	var block *pem.Block
	for rest := certificateChainPem; len(rest) != 0; {
		block, rest = pem.Decode(rest)
		if block != nil && block.Type == "CERTIFICATE" {
			x509Certificate, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, errors.Annotate(err, "failed to parse certificate")
			}
			list = append(list, x509Certificate)
		}
	}
	return list, nil
}

// EncodeToPEM converts certificate to PEM format, with optional comments
func EncodeToPEM(out io.Writer, crt *x509.Certificate, withComments bool) error {
	if withComments {
		fmt.Fprintf(out, "\n#   Issuer: %s", NameToString(&crt.Issuer))
		fmt.Fprintf(out, "\n#   Subject: %s", NameToString(&crt.Subject))
		fmt.Fprint(out, "\n#   Validity")
		fmt.Fprintf(out, "\n#       Not Before: %s", crt.NotBefore.UTC().Format(certTimeFormat))
		fmt.Fprintf(out, "\n#       Not After : %s", crt.NotAfter.UTC().Format(certTimeFormat))
		fmt.Fprint(out, "\n")
	}

	err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: crt.Raw})
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// EncodeAllToPEM converts certificate bundle to PEM format, with optional comments
func EncodeAllToPEM(out io.Writer, certs []*x509.Certificate, withComments bool) error {
	for _, crt := range certs {
		err := EncodeToPEM(out, crt, withComments)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// EncodeToPEMString converts certificate to PEM format, with optional comments
func EncodeToPEMString(crt *x509.Certificate, withComments bool) (string, error) {
	if crt == nil {
		return "", nil
	}
	b := bytes.NewBuffer([]byte{})
	err := EncodeToPEM(b, crt, withComments)
	if err != nil {
		return "", errors.Trace(err)
	}
	pem := string(b.Bytes())
	pem = strings.TrimSpace(pem)
	pem = strings.Replace(pem, "\n\n", "\n", -1)
	return pem, nil
}

// EncodeAllToPEMString converts certificate bundle to PEM format, with optional comments
func EncodeAllToPEMString(certs []*x509.Certificate, withComments bool) (string, error) {
	if len(certs) == 0 {
		return "", nil
	}

	b := bytes.NewBuffer([]byte{})
	err := EncodeAllToPEM(b, certs, withComments)
	if err != nil {
		return "", errors.Trace(err)
	}
	pem := string(b.Bytes())
	pem = strings.TrimSpace(pem)
	pem = strings.Replace(pem, "\n\n", "\n", -1)
	return pem, nil
}
