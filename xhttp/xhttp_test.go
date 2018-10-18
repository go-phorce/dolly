package xhttp

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func Test_GetRelativeURL(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "/v1/somepath", nil)
	require.NoError(t, err)

	u := GetRelativeURL(r, "/v1/a123/b456")
	assert.Equal(t, "http://localhost/v1/a123/b456", u.String())

	r.Host = "1.1.1.1"
	u = GetRelativeURL(r, "/v1/a123/b456")
	assert.Equal(t, "http://1.1.1.1/v1/a123/b456", u.String())

	r.TLS = &tls.ConnectionState{}
	u = GetRelativeURL(r, "/v1/a123/b456")
	assert.Equal(t, "https://1.1.1.1/v1/a123/b456", u.String())
}
