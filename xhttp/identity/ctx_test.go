package identity

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
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

func Test_SetGlobal(t *testing.T) {
	assert.Panics(t, func() { SetGlobalIdentityMapper(nil) })
	assert.Panics(t, func() { SetGlobalNodeInfo(nil) })
}

func Test_Identity(t *testing.T) {
	i := identity{role: "netmgmt", name: "Ekspand"}
	assert.Equal(t, "netmgmt", i.Role())
	assert.Equal(t, "Ekspand", i.Name())
	assert.Equal(t, "netmgmt/Ekspand", i.String())

	id := NewIdentity("netmgmt", "Ekspand")
	assert.Equal(t, "netmgmt", id.Role())
	assert.Equal(t, "Ekspand", id.Name())
	assert.Equal(t, "netmgmt/Ekspand", id.String())
}

func Test_ForRequest(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	ctx := ForRequest(r)
	assert.NotNil(t, ctx)
}

func Test_HostnameHeader(t *testing.T) {
	d := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
	rw := httptest.NewRecorder()
	handler := NewContextHandler(d)
	r, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)
	handler.ServeHTTP(rw, r)
	assert.NotEmpty(t, rw.Header().Get(header.XHostname))
}

func Test_ClientIP(t *testing.T) {
	d := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller := ForRequest(r)
		assert.Equal(t, "10.0.0.1", caller.ClientIP())
		assert.NotEmpty(t, caller.CorrelationID())
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
		assert.Equal(t, GuestRoleName, rn.Role)
		assert.Equal(t, "dolly", rn.Name)
	})

	t.Run("cn_extractor", func(t *testing.T) {
		SetGlobalIdentityMapper(identityMapperFromCN)
		// restore
		defer SetGlobalIdentityMapper(GuestIdentityMapper)
		r, err := http.NewRequest(http.MethodGet, "/dolly", nil)
		require.NoError(t, err)

		r.TLS = &tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{
				{
					Subject: pkix.Name{
						CommonName:   "cn-dolly",
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
		assert.Equal(t, "cn-dolly", rn.Role)
		assert.Equal(t, "cn-dolly", rn.Name)
	})

	t.Run("cn_extractor_must", func(t *testing.T) {
		SetGlobalIdentityMapper(identityMapperFromCNMust)
		// restore
		defer SetGlobalIdentityMapper(GuestIdentityMapper)
		r, err := http.NewRequest(http.MethodGet, "/dolly", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusUnauthorized, w.Code)

		assert.Equal(t, `{"code":"unauthorized","message":"missing client certificate"}`, string(w.Body.Bytes()))
	})
	t.Run("cn_extractor_must_ForRequest", func(t *testing.T) {
		SetGlobalIdentityMapper(identityMapperFromCNMust)
		// restore
		defer SetGlobalIdentityMapper(GuestIdentityMapper)
		r, err := http.NewRequest(http.MethodGet, "/dolly", nil)
		require.NoError(t, err)

		ctx := ForRequest(r)
		assert.Equal(t, GuestRoleName, ctx.Identity().Role())
	})
}

func identityMapperFromCN(r *http.Request) (Identity, error) {
	var role string
	var name string
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		name = ClientIPFromRequest(r)
		role = GuestRoleName
	} else {
		name = r.TLS.PeerCertificates[0].Subject.CommonName
		role = r.TLS.PeerCertificates[0].Subject.CommonName
	}
	return identity{name: name, role: role}, nil
}

func identityMapperFromCNMust(r *http.Request) (Identity, error) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return nil, errors.New("missing client certificate")
	}
	return identity{name: r.TLS.PeerCertificates[0].Subject.CommonName, role: r.TLS.PeerCertificates[0].Subject.CommonName}, nil
}
