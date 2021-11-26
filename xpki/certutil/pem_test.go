package certutil_test

import (
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
	pem, err := certutil.EncodeToPEMString(true, crt)
	require.NoError(t, err)
	assert.Equal(t, orig, pem)

	crt1, err := certutil.ParseFromPEM([]byte(issuer1))
	require.NoError(t, err)
	crt2, err := certutil.ParseFromPEM([]byte(issuer2))
	require.NoError(t, err)

	pem, err = certutil.EncodeToPEMString(true, crt1, crt2)
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(issuers), pem)
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

func Test_LoadChainFromPEM(t *testing.T) {
	chain, err := certutil.LoadChainFromPEM("testdata/ca-bundle.pem")
	require.NoError(t, err)
	assert.True(t, len(chain) > 100)
}

func TestLoadPEMFiles(t *testing.T) {
	b, err := certutil.LoadPEMFiles("testdata/ca-bundle.pem", "testdata/int-bundle.pem")
	require.NoError(t, err)

	_, err = certutil.ParseChainFromPEM(b)
	require.NoError(t, err)

	_, err = certutil.CreatePoolFromPEM(b)
	require.NoError(t, err)
}

func TestJoinPEM(t *testing.T) {
	assert.Equal(t, []byte("1\n2"), certutil.JoinPEM([]byte("\n   1  \n\n\n"), []byte("\t  \n   2  \n\n\t \n   ")))
	assert.Equal(t, []byte("1"), certutil.JoinPEM([]byte("\n   1  \n\n\n"), nil))
	assert.Equal(t, []byte("2"), certutil.JoinPEM(nil, []byte("\t  \n   2  \n\n\t \n   ")))
}
