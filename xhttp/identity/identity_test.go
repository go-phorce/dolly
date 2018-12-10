package identity

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-phorce/dolly/netutil"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_extractIdentityFromRequest(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	t.Run("when IP is not set", func(t *testing.T) {
		ip, err := netutil.GetLocalIP()
		require.NoError(t, err)

		idn := identityMapper(r)
		assert.Equal(t, "guest/"+ip, idn.String())
	})

	t.Run("when IP is set", func(t *testing.T) {
		r.RemoteAddr = "10.0.1.2:443"

		idn := identityMapper(r)
		assert.Equal(t, "guest/10.0.1.2", idn.String())
	})

	t.Run("when TLS is set and defaultExtractor", func(t *testing.T) {
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

		idn := identityMapper(r)
		assert.Equal(t, "guest/dolly", idn.String())
	})
}

func Test_WithTestIdentityDirect(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	r = WithTestIdentity(r, NewIdentity("role1", "name1"))
	ctx := ForRequest(r)

	assert.Equal(t, "role1/name1", ctx.Identity().String())
}
func Test_WithTestIdentityServeHTTP(t *testing.T) {
	d := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller := ForRequest(r)
		assert.Equal(t, "role1/name2", caller.Identity().String())
	})
	rw := httptest.NewRecorder()
	handler := NewContextHandler(d)
	r, _ := http.NewRequest("GET", "/test", nil)
	r = WithTestIdentity(r, NewIdentity("role1", "name2"))
	handler.ServeHTTP(rw, r)
	assert.NotEqual(t, "", rw.Header().Get(header.XHostname))
}
