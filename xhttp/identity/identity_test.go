package identity

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"testing"

	"github.com/go-phorce/dolly/netutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_extractIdentityFromRequest(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	t.Run("when IP is not set", func(t *testing.T) {
		ip, err := netutil.GetLocalIP()
		require.NoError(t, err)

		idn := extractIdentityFromRequest(r)
		assert.Equal(t, "guest/"+ip, idn.String())
	})

	t.Run("when IP is set", func(t *testing.T) {
		r.RemoteAddr = "10.0.1.2:443"

		idn := extractIdentityFromRequest(r)
		assert.Equal(t, "guest/10.0.1.2", idn.String())
	})

	t.Run("when TLS is set", func(t *testing.T) {
		r.TLS = &tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{
				{
					Subject: pkix.Name{
						CommonName:   "dolly",
						Organization: []string{"org"},
					},
				},
			},
		}

		idn := extractIdentityFromRequest(r)
		assert.Equal(t, "dolly", idn.String())
	})
}

func Test_WithTestIdentity(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	r = WithTestIdentity(r, NewIdentity("role1", "name1"))
	ctx := ForRequest(r)

	assert.Equal(t, "role1/name1", ctx.Identity().String())
}
