package identity

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xhttp/marshal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	rc := m.Run()
	os.Exit(rc)
}

func Test_Identity(t *testing.T) {
	i := identity{role: "netmgmt", name: "Ekspand"}
	assert.Equal(t, "netmgmt", i.Role())
	assert.Equal(t, "Ekspand", i.Name())
	assert.Equal(t, "netmgmt/Ekspand", i.String())
}

func Test_NewIdentity(t *testing.T) {
	i := NewIdentity("netmgmt", "Ekspand")
	assert.Equal(t, "netmgmt", i.Role())
	assert.Equal(t, "Ekspand", i.Name())
	assert.Equal(t, "netmgmt/Ekspand", i.String())
}

func Test_HostnameHeader(t *testing.T) {
	d := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
	rw := httptest.NewRecorder()
	handler := NewContextHandler(d)
	r, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)
	handler.ServeHTTP(rw, r)
	assert.NotEqual(t, "", rw.Header().Get(header.XHostname))
}

func Test_ClientIP(t *testing.T) {
	d := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller := ForRequest(r)
		assert.Equal(t, "10.0.0.1", caller.ClientIP())
	})
	rw := httptest.NewRecorder()
	handler := NewContextHandler(d)
	r, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	r.RemoteAddr = "10.0.0.1"

	handler.ServeHTTP(rw, r)
	assert.NotEqual(t, "", rw.Header().Get(header.XHostname))
}

func Test_RequestorIdentity(t *testing.T) {
	type roleName struct {
		Role string `json:"role,omitempty"`
		Name string `json:"name,omitempty"`
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		ctx := ForRequest(r)
		identity := ctx.Identity()
		res := &roleName{
			Role: identity.Role(),
			Name: identity.Name(),
		}
		marshal.WriteJSON(w, r, res)
	}

	handler := NewContextHandler(http.HandlerFunc(h))

	t.Run("default_extractor", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/dolly", nil)
		require.NoError(t, err)

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

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		resp := w.Result()
		defer resp.Body.Close()

		rn := &roleName{}
		require.NoError(t, marshal.Decode(resp.Body, rn))
		assert.Equal(t, "guest", rn.Role)
		assert.Equal(t, "dolly", rn.Name)
	})

	t.Run("cn_extractor", func(t *testing.T) {
		SetGlobalRoleExtractor(func(n *pkix.Name) string {
			return n.CommonName
		})
		// restore
		defer SetGlobalRoleExtractor(defaultExtractRole)

		r, err := http.NewRequest(http.MethodGet, "/dolly", nil)
		require.NoError(t, err)

		r = WithTestIdentity(r, NewIdentity("dolly", "dolly.com"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		resp := w.Result()
		defer resp.Body.Close()

		rn := &roleName{}
		require.NoError(t, marshal.Decode(resp.Body, rn))
		assert.Equal(t, "dolly", rn.Role)
		assert.Equal(t, "dolly.com", rn.Name)
	})
}
