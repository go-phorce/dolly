package marshal

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_WritePlainJSON(t *testing.T) {
	v := &AStruct{
		A: "a",
		B: "b",
	}

	t.Run("DontPrettyPrint", func(t *testing.T) {
		w := httptest.NewRecorder()
		WritePlainJSON(w, v, DontPrettyPrint)
		assert.Equal(t, `{"A":"a","B":"b"}`, string(w.Body.Bytes()))
	})

	t.Run("PrettyPrint", func(t *testing.T) {
		pretty := `{
	"A": "a",
	"B": "b"
}`
		w := httptest.NewRecorder()
		WritePlainJSON(w, v, PrettyPrint)
		assert.Equal(t, pretty, string(w.Body.Bytes()))
	})
}
