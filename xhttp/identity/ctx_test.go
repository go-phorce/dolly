package identity

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	Initialize(nil, nil)

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

func Test_CallerHost(t *testing.T) {
	d := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller := ForRequest(r)
		assert.Equal(t, "somehost", caller.Host())
	})
	rw := httptest.NewRecorder()
	handler := NewContextHandler(d)
	r, err := http.NewRequest("GET", "/test", nil)
	r.Host = "somehost:2323"
	assert.NoError(t, err)
	handler.ServeHTTP(rw, r)
	assert.NotEqual(t, "", rw.Header().Get(header.XHostname))
}

func Test_RoleContext(t *testing.T) {
	c := NewForRole("bob_1").WithCorrelationID("1234gdhfewq")
	assert.Equal(t, "1234gdhfewq", c.CorrelationID())
	assert.Equal(t, "bob_1/"+nodeInfo.HostName(), c.Identity().String())
	assert.Equal(t, nodeInfo.HostName(), c.Identity().Name())
	assert.Equal(t, "bob_1", c.Identity().Role())
}

func Test_RequestorIdentity(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		ctx := ForRequest(r)
		identity := ctx.Identity()
		responseBody := fmt.Sprintf("{\"role\": \"%s\", \"name\": \"%s\" }", identity.Role(), identity.Name())
		io.WriteString(w, responseBody)
	}

	server := httptest.NewServer(http.HandlerFunc(h))
	defer server.Close()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	identity := NewIdentity("enrollme_dev", "localhost")
	r = WithTestIdentity(r, identity)

	w := httptest.NewRecorder()
	h(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	type rt struct {
		Role string `json:"role,omitempty"`
		Name string `json:"name,omitempty"`
	}

	res := &rt{}
	body := w.Body.Bytes()
	s := string(body)
	assert.NoError(t, json.Unmarshal(body, res))
	assert.Equal(t, identity.Role(), res.Role, s)
	assert.Equal(t, identity.Name(), res.Name, s)
}

func Test_RequestorHost(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		ctx := ForRequest(r)
		host := ctx.Host()
		responseBody := fmt.Sprintf(`{"host":"%s"}`, host)
		io.WriteString(w, responseBody)
	}

	type rt struct {
		Host string `json:"host,omitempty"`
	}

	handler := NewContextHandler(http.HandlerFunc(h))
	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("request_host", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		r.Host = "testhost"
		require.NoError(t, err)

		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		res := &rt{}
		body := w.Body.Bytes()
		s := string(body)
		assert.NoError(t, json.Unmarshal(body, res))
		assert.Equal(t, "testhost", res.Host, s)
	})

	t.Run("request_host_port", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		r.Host = "testhost:123"
		require.NoError(t, err)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		res := &rt{}
		body := w.Body.Bytes()
		s := string(body)
		assert.NoError(t, json.Unmarshal(body, res))
		assert.Equal(t, "testhost", res.Host, s)
	})
	t.Run("request_host_port_invalid", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		r.Host = "[testhost:123:"
		require.NoError(t, err)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		res := &rt{}
		body := w.Body.Bytes()
		s := string(body)
		assert.NoError(t, json.Unmarshal(body, res))
		assert.Equal(t, "[testhost:123:", res.Host, s)
	})
	t.Run("request_host_header", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/", nil)
		r.Host = "testhost:123"
		r.Header.Set(header.XHostname, "newhostname:190")
		require.NoError(t, err)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		res := &rt{}
		body := w.Body.Bytes()
		s := string(body)
		assert.NoError(t, json.Unmarshal(body, res))
		assert.Equal(t, "newhostname", res.Host, s)
	})
}
