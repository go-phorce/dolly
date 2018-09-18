package context

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	Initialize(nil, nil)

	rc := m.Run()
	os.Exit(rc)
}

func Test_ClientIdentity(t *testing.T) {
	i := clientIdentity{role: "netmgmt", cn: "Ekspand"}
	assert.Equal(t, "netmgmt", i.Role())
	assert.Equal(t, "Ekspand", i.CommonName())
	assert.Equal(t, "netmgmt/Ekspand", i.String())
}

func Test_NewIdentity(t *testing.T) {
	i := NewIdentity("netmgmt", "Ekspand")
	assert.Equal(t, "netmgmt", i.Role())
	assert.Equal(t, "Ekspand", i.CommonName())
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
	c := NewForRole("bob_1").WithRequestID("1234gdhfewq")
	assert.Equal(t, "1234gdhfewq", c.RequestID())
	assert.Equal(t, "bob_1/"+nodeInfo.HostName(), c.Identity().String())
	assert.Equal(t, nodeInfo.HostName(), c.Identity().CommonName())
	assert.Equal(t, "bob_1", c.Identity().Role())
}
