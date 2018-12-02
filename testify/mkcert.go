package testify

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math"
	"math/big"
	"math/rand"
	"time"

	"github.com/juju/errors"
)

// MakeSelfCertECDSA creates self-signed cert
func MakeSelfCertECDSA(hours int) (*x509.Certificate, crypto.PrivateKey, error) {
	// key pair
	key, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
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

// MakeSelfCertECDSAPem creates self-signed cert in PEM format
func MakeSelfCertECDSAPem(hours int) (pemCert, pemKey []byte, err error) {
	crt, key, err := MakeSelfCertECDSA(hours)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(key.(*ecdsa.PrivateKey))
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	pemKey = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})
	pemCert = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crt.Raw,
	})
	return
}

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

// MakeSelfCertRSAPem creates self-signed cert in PEM format
func MakeSelfCertRSAPem(hours int) (pemCert, pemKey []byte, err error) {
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

// GenerateRSAKeyInPEM returns PEM encoded RSA key
func GenerateRSAKeyInPEM(rand io.Reader, size int) ([]byte, error) {
	if rand == nil {
		rand = crand.Reader
	}
	// key pair
	key, err := rsa.GenerateKey(crand.Reader, size)
	if err != nil {
		return nil, errors.Trace(err)
	}

	pemKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return pemKey, nil
}

// GenerateECDSAKeyInPEM returns PEM encoded ECDSA key
func GenerateECDSAKeyInPEM(rand io.Reader, c elliptic.Curve) ([]byte, error) {
	if rand == nil {
		rand = crand.Reader
	}
	// key pair
	key, err := ecdsa.GenerateKey(c, rand)
	if err != nil {
		return nil, errors.Trace(err)
	}
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, errors.Trace(err)
	}

	pemKey := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	return pemKey, nil
}
