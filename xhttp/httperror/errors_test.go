package httperror_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorCode_JSON(t *testing.T) {
	v := map[string]string{"foo": httperror.InvalidJSON}
	b, err := json.Marshal(&v)
	require.NoError(t, err, "Unable to marshal to json")
	exp := `{"foo":"invalid_json"}`
	assert.Equal(t, exp, string(b), "Unexpected JSON serializtion of ErrorCode")
}

func TestError_Error(t *testing.T) {
	// compile error if Error doesn't impl error
	var _ error = &httperror.Error{}

	e := httperror.New(http.StatusBadRequest, httperror.InvalidJSON, "Bob")
	assert.Equal(t, "invalid_json: Bob", e.Error())

	e.WithCause(errors.New("some other error"))
	assert.Equal(t, "invalid_json: Bob", e.Error())
}

func TestError_ManyErrorIsError(t *testing.T) {
	err := httperror.NewMany(http.StatusBadRequest, httperror.RateLimitExceeded, "There were 42 errors!")
	var _ error = err // won't compile if ManyError doesn't impl error
	assert.Equal(t, "rate_limit_exceeded: There were 42 errors!", err.Error())
}

func TestError_AddErrorToManyError(t *testing.T) {
	me := httperror.NewMany(http.StatusBadRequest, httperror.RateLimitExceeded, "There were 42 errors!")
	me.Add("one", errors.Errorf("test error 1"))
	assert.Equal(t, 1, len(me.Errors))
	me.Add("two", httperror.New(http.StatusBadRequest, httperror.InvalidJSON, "test error 2"))
	assert.Equal(t, 2, len(me.Errors))
	assert.True(t, me.HasErrors(), "many error contains two errors")
	assert.Contains(t, me.Errors, "one")
	assert.Contains(t, me.Errors, "two")
}

func TestError_AddErrorToNilManyError(t *testing.T) {
	var me httperror.ManyError
	me.Add("one", errors.Errorf("test error 1"))
	assert.Equal(t, 1, len(me.Errors))
	me.Add("two", httperror.New(http.StatusBadRequest, httperror.InvalidJSON, "test error 2"))
	assert.Equal(t, 2, len(me.Errors))
	assert.True(t, me.HasErrors(), "many error contains two errors")
	assert.Contains(t, me.Errors, "one")
	assert.Contains(t, me.Errors, "two")
}

func TestError_WriteHTTPResponse(t *testing.T) {
	single := httperror.New(http.StatusBadRequest, httperror.InvalidJSON, "test error 2")

	many := httperror.NewMany(http.StatusBadRequest, httperror.RateLimitExceeded, "There were 2 errors!")
	many.Add("one", errors.Errorf("test error 1"))
	many.Add("two", httperror.New(http.StatusBadRequest, httperror.InvalidJSON, "test error 2"))

	manyNil := &httperror.ManyError{HTTPStatus: http.StatusBadRequest}
	manyNil.Add("one", errors.Errorf("test error 1"))
	manyNil.Add("two", httperror.New(http.StatusBadRequest, httperror.InvalidJSON, "test error 2"))

	cases := []struct {
		name     string
		err      error
		urlPath  string
		expected string
	}{
		{
			name:     "single_raw_json",
			err:      single,
			urlPath:  "/",
			expected: `{"code":"invalid_json","message":"test error 2"}`,
		},
		{
			name:    "single_pretty_json",
			err:     single,
			urlPath: "/?pp",
			expected: `{
	"code": "invalid_json",
	"message": "test error 2"
}`,
		},
		{
			name:    "many_pretty_json",
			err:     many,
			urlPath: "/?pp",
			expected: `{
	"code": "rate_limit_exceeded",
	"errors": {
		"one": {
			"code": "unexpected",
			"message": "test error 1"
		},
		"two": {
			"code": "invalid_json",
			"message": "test error 2"
		}
	},
	"message": "There were 2 errors!"
}`,
		},
		{
			name:    "manynil_pretty_json",
			err:     manyNil,
			urlPath: "/?pp",
			expected: `{
	"errors": {
		"one": {
			"code": "unexpected",
			"message": "test error 1"
		},
		"two": {
			"code": "invalid_json",
			"message": "test error 2"
		}
	}
}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, err := http.NewRequest(http.MethodGet, tc.urlPath, nil)
			require.NoError(t, err)

			switch tc.err.(type) {
			case *httperror.ManyError:
				tc.err.(*httperror.ManyError).WriteHTTPResponse(w, r)
			default:
				tc.err.(*httperror.Error).WriteHTTPResponse(w, r)
			}
			assert.Equal(t, tc.expected, string(w.Body.Bytes()))
		})
	}
}
