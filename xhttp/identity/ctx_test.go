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
	d := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	rw := httptest.NewRecorder()
	handler := NewContextHandler(d)
	r, err := http.NewRequest("GET", "/test", nil)
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

func Test_TestIdentity(t *testing.T) {
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
