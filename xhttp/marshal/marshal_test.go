package marshal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/stretchr/testify/assert"
)

func Test_WritePlainJSON(t *testing.T) {
	v := &AStruct{
		A: "a",
		B: "b",
	}

	t.Run("DontPrettyPrint", func(t *testing.T) {
		w := httptest.NewRecorder()
		WritePlainJSON(w, http.StatusOK, v, DontPrettyPrint)
		assert.Equal(t, `{"A":"a","B":"b"}`, string(w.Body.Bytes()))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, header.ApplicationJSON, w.Header().Get(header.ContentType))
	})

	t.Run("PrettyPrint", func(t *testing.T) {
		pretty := `{
	"A": "a",
	"B": "b"
}`
		w := httptest.NewRecorder()
		WritePlainJSON(w, http.StatusCreated, v, PrettyPrint)
		assert.Equal(t, pretty, string(w.Body.Bytes()))
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, header.ApplicationJSON, w.Header().Get(header.ContentType))
	})
}
