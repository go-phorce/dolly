package testca

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// RFC-5280: { id-kp 8 }
	// RFC-3161: {iso(1) identified-organization(3) dod(6) internet(1) security(5) mechanisms(5) pkix(7) kp (3) timestamping (8)}
	oidExtKeyUsageTimeStamping = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 8}
	// Certificate extension: "extKeyUsage": {joint-iso-itu-t(2) ds(5) certificateExtension(29) extKeyUsage(37)}
	oidExtKeyUsage = asn1.ObjectIdentifier{2, 5, 29, 37}
)

func getPublicKey(privkey interface{}) interface{} {
	switch privkey.(type) {
	case *ecdsa.PrivateKey:
		return privkey.(*ecdsa.PrivateKey).Public()
	case *rsa.PrivateKey:
		return privkey.(*rsa.PrivateKey).Public()
	default:
		panic("unsupported private key")
	}

}

// MakeValidCertsChainTSA creates valid TSA cert with the only critical EKU extension for timestamping
func MakeValidCertsChainTSA(t *testing.T, hours int, ec bool) (crypto.Signer, *x509.Certificate, []*x509.Certificate, *x509.Certificate) {
	var err error
	var rootkey, cakey, key interface{}

	if ec {
		rootkey, err = ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
		require.NoError(t, err)
		cakey, err = ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
		require.NoError(t, err)
		key, err = ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
		require.NoError(t, err)

	} else {
		rootkey, err = rsa.GenerateKey(crand.Reader, 2048)
		require.NoError(t, err)
		cakey, err = rsa.GenerateKey(crand.Reader, 2048)
		require.NoError(t, err)
		key, err = rsa.GenerateKey(crand.Reader, 2048)
		require.NoError(t, err)
	}
	certRootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(math.MaxInt64)),
		Subject: pkix.Name{
			CommonName: "[TEST] Timestamp Root CA",
		},
		NotBefore:             time.Now().UTC().Add(-time.Hour),
		NotAfter:              time.Now().UTC().Add(time.Hour * time.Duration(hours)),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Root Cert
	der, err := x509.CreateCertificate(crand.Reader, certRootTemplate, certRootTemplate, getPublicKey(rootkey), rootkey)
	require.NoError(t, err)

	certRoot, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	require.True(t, certRoot.IsCA)

	caCertTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(math.MaxInt64)),
		Subject: pkix.Name{
			CommonName: "[TEST] Timestamp Issuing CA Level 1",
		},
		NotBefore:             time.Now().UTC().Add(-time.Hour),
		NotAfter:              time.Now().UTC().Add(time.Hour * time.Duration(hours)),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}

	// CA Cert
	der, err = x509.CreateCertificate(crand.Reader, caCertTemplate, certRoot, getPublicKey(cakey), rootkey)
	require.NoError(t, err)

	certCA, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	require.True(t, certCA.IsCA)

	oids := []asn1.ObjectIdentifier{oidExtKeyUsageTimeStamping}
	eku, err := asn1.Marshal(oids)
	require.NoError(t, err)

	// TS requires only one OidExtKeyUsageTimeStamping extension
	tsCertTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(math.MaxInt64)),
		Subject: pkix.Name{
			CommonName: "[TEST] TSA",
		},
		NotBefore: time.Now().UTC().Add(-time.Hour),
		NotAfter:  time.Now().UTC().Add(time.Hour * time.Duration(hours)),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtraExtensions: []pkix.Extension{
			{
				Id:       oidExtKeyUsage,
				Critical: true,
				Value:    eku,
			},
		},
	}

	der, err = x509.CreateCertificate(crand.Reader, tsCertTemplate, certCA, getPublicKey(key), cakey)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.Equal(t, 1, len(cert.ExtKeyUsage))
	assert.Equal(t, x509.ExtKeyUsageTimeStamping, cert.ExtKeyUsage[0])

	return key.(crypto.Signer), cert, []*x509.Certificate{certCA}, certRoot
}

// MakeInvalidCertsChainTSA creates invalid TSA cert with several critical EKU extensions
func MakeInvalidCertsChainTSA(t *testing.T, hours int) (crypto.Signer, *x509.Certificate, []*x509.Certificate, *x509.Certificate) {
	rootkey, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)

	certRootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(math.MaxInt64)),
		Subject: pkix.Name{
			CommonName: "[TEST] Timestamp Root CA",
		},
		NotBefore:             time.Now().UTC().Add(-time.Hour),
		NotAfter:              time.Now().UTC().Add(time.Hour * time.Duration(hours)),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Root Cert
	der, err := x509.CreateCertificate(crand.Reader, certRootTemplate, certRootTemplate, &rootkey.PublicKey, rootkey)
	require.NoError(t, err)

	certRoot, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	require.True(t, certRoot.IsCA)

	tsCertTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(math.MaxInt64)),
		Subject: pkix.Name{
			CommonName: "[TEST] TSA",
		},
		NotBefore: time.Now().UTC().Add(-time.Hour),
		NotAfter:  time.Now().UTC().Add(time.Hour * time.Duration(hours)),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageTimeStamping,
			x509.ExtKeyUsageCodeSigning,
		},
	}

	der, err = x509.CreateCertificate(crand.Reader, tsCertTemplate, certRoot, &key.PublicKey, rootkey)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.Equal(t, 2, len(cert.ExtKeyUsage))

	return key, cert, []*x509.Certificate{}, certRoot
}
