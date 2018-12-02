package testify_test

import (
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/go-phorce/dolly/testify"
	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeCertRSA(t *testing.T) {
	pemCert, keyPem, err := testify.MakeSelfCertRSAPem(720)
	require.NoError(t, err)
	_, err = certutil.ParseFromPEM(pemCert)
	require.NoError(t, err)

	block, _ := pem.Decode(keyPem)
	assert.Equal(t, "RSA PRIVATE KEY", block.Type)

	_, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
}

func TestMakeCertECDSA(t *testing.T) {
	pemCert, keyPem, err := testify.MakeSelfCertECDSAPem(720)
	require.NoError(t, err)
	_, err = certutil.ParseFromPEM(pemCert)
	require.NoError(t, err)

	block, _ := pem.Decode(keyPem)
	assert.Equal(t, "EC PRIVATE KEY", block.Type)

	_, err = x509.ParseECPrivateKey(block.Bytes)
	require.NoError(t, err)
}

func TestGenerateECDSAKeyInPEM(t *testing.T) {
	keyPem, err := testify.GenerateECDSAKeyInPEM(nil, elliptic.P256())
	require.NoError(t, err)

	block, _ := pem.Decode(keyPem)
	assert.Equal(t, "EC PRIVATE KEY", block.Type)

	_, err = x509.ParseECPrivateKey(block.Bytes)
	require.NoError(t, err)
}

func TestGenerateRSAKeyInPEM(t *testing.T) {
	keyPem, err := testify.GenerateRSAKeyInPEM(nil, 1024)
	require.NoError(t, err)

	block, _ := pem.Decode(keyPem)
	assert.Equal(t, "RSA PRIVATE KEY", block.Type)

	_, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)
}
