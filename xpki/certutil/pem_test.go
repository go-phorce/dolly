package certutil_test

import (
	"crypto/x509"
	"strings"
	"testing"

	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_EncodeToPEMString(t *testing.T) {
	orig := strings.TrimSpace(selfSignedCert)
	crt, err := certutil.ParseFromPEM([]byte(orig))
	require.NoError(t, err)

	pem, err := certutil.EncodeToPEMString(crt, true)
	require.NoError(t, err)
	assert.Equal(t, orig, pem)

	pem, err = certutil.EncodeToPEMString(nil, false)
	require.NoError(t, err)
	assert.Equal(t, "", pem)
}

func Test_EncodeAllToPEMString(t *testing.T) {
	crt1, err := certutil.ParseFromPEM([]byte(issuer1))
	require.NoError(t, err)
	crt2, err := certutil.ParseFromPEM([]byte(issuer2))
	require.NoError(t, err)

	pem, err := certutil.EncodeAllToPEMString([]*x509.Certificate{crt1, crt2}, true)
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(issuers), pem)

	pem, err = certutil.EncodeAllToPEMString(nil, false)
	require.NoError(t, err)
	assert.Equal(t, "", pem)

	pem, err = certutil.EncodeAllToPEMString([]*x509.Certificate{}, false)
	require.NoError(t, err)
	assert.Equal(t, "", pem)
}

func Test_ParseChainFromPEM(t *testing.T) {
	list, err := certutil.ParseChainFromPEM([]byte(issuers))
	require.NoError(t, err)
	assert.Equal(t, 2, len(list))
}

func Test_LoadFromPEM(t *testing.T) {
	crt, err := certutil.LoadFromPEM("testdata/selfsigned.pem")
	require.NoError(t, err)

	n := certutil.NameToString(&crt.Subject)
	assert.Equal(t, "C=US, ST=CA, L=San Francisco, O=CloudFlare LLC, OU=Security, CN=testssl.lol", n)
}
