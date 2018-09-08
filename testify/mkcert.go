package testify

import (
	"crypto"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math"
	"math/big"
	"math/rand"
	"time"

	"github.com/juju/errors"
)

// MakeSelfCertRSA creates self-signed cert
func MakeSelfCertRSA(hours int) (*x509.Certificate, crypto.PrivateKey, error) {
	// rsa key pair
	key, err := rsa.GenerateKey(crand.Reader, 512)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	// certificate
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(math.MaxInt64)),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore: time.Now().UTC().Add(-time.Hour),
		NotAfter:  time.Now().UTC().Add(time.Hour * time.Duration(hours)),
	}
	der, err := x509.CreateCertificate(crand.Reader, certTemplate, certTemplate, &key.PublicKey, key)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	crt, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	return crt, key, nil
}

// MakeSelfCertPem creates self-signed cert in PEM format
func MakeSelfCertPem(hours int) (pemCert, pemKey []byte, err error) {
	crt, key, err := MakeSelfCertRSA(hours)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	pemKey = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey)),
	})
	pemCert = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crt.Raw,
	})
	return
}
