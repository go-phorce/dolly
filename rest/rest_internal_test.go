package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_notFoundHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/blah", nil)
	require.NoError(t, err)
	notFoundHandler(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, header.ApplicationJSON, w.Header().Get(header.ContentType))
	assert.Equal(t, `{"code":"not_found","message":"/blah"}`, string(w.Body.Bytes()))
}
