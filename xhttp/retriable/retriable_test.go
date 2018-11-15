package retriable_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/go-phorce/dolly/xhttp/marshal"
	"github.com/go-phorce/dolly/xhttp/retriable"
	"github.com/go-phorce/dolly/xlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	p := &retriable.Policy{
		TotalRetryLimit: 5,
	}

	c := retriable.New().WithName("test")
	assert.NotNil(t, c)

	c.WithPolicy(p)
}

func TestDefaultPolicy(t *testing.T) {
	tcases := []struct {
		expected   bool
		reason     string
		retries    int
		statusCode int
		err        error
	}{
		// 429 is rate limit exceeded
		{true, "rate-limit", 0, 429, nil},
		{true, "rate-limit", 1, 429, nil},
		{true, "rate-limit", 3, 429, nil},
		{false, "rate-limit", 4, 429, nil},
		// 503 is service unavailable, which is returned during leader elections
		{true, "unavailable", 0, 503, nil},
		{true, "unavailable", 1, 503, nil},
		{true, "unavailable", 9, 503, nil},
		{false, retriable.LimitExceeded, 10, 503, nil},
		// 502 is bad gateway, which is returned during leader transitions
		{true, "gateway", 0, 502, nil},
		{true, "gateway", 1, 502, nil},
		{true, "gateway", 9, 502, nil},
		{false, retriable.LimitExceeded, 10, 502, nil},
		// regardless of config, other status codes shouldn't get retries
		{false, "success", 0, 200, nil},
		{false, retriable.NonRetriableError, 0, 400, nil},
		{false, retriable.NonRetriableError, 0, 401, nil},
		{false, retriable.NotFound, 0, 404, nil},
		{false, retriable.NonRetriableError, 0, 500, nil},
		// connection
		{true, "connection", 0, 0, errors.New("some error")},
		{true, "connection", 5, 0, errors.New("some error")},
		{false, "connection", 6, 0, errors.New("some error")},
	}

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	p := retriable.NewDefaultPolicy()
	for _, tc := range tcases {
		t.Run(fmt.Sprintf("%s: %d, %d, %t:", tc.reason, tc.retries, tc.statusCode, tc.expected), func(t *testing.T) {
			res := &http.Response{StatusCode: tc.statusCode}
			should, _, reason := p.ShouldRetry(req, res, tc.err, tc.retries)
			assert.Equal(t, tc.expected, should)
			assert.Equal(t, tc.reason, reason)
		})
	}
}

func Test_Retriable_OK(t *testing.T) {
	h := makeTestHandler(t, "/v1/test", http.StatusOK, `{
		"status": "ok"
	}`)
	server := httptest.NewServer(h)
	defer server.Close()

	client := retriable.New()
	require.NotNil(t, client)

	hosts := []string{server.URL}

	t.Run("GET WithTimeout", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		status, err := client.Get(ctx, hosts, "/v1/test", w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("PUT", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		status, err := client.PutBody(nil, hosts, "/v1/test", []byte("{}"), w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("POST", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		status, err := client.PostBody(nil, hosts, "/v1/test", []byte("{}"), w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("POST Empty body", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		status, err := client.PostBody(nil, hosts, "/v1/test", nil, w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("DELETE", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		status, err := client.Delete(nil, hosts, "/v1/test", w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})
}

func Test_RetriableWithHeaders(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		headers := map[string]string{
			header.Accept:      r.Header.Get(header.Accept),
			header.ContentType: r.Header.Get(header.ContentType),
			"h1":               r.Header.Get("header1"),
			"h2":               r.Header.Get("header2"),
			"h3":               r.Header.Get("header3"),
			"h4":               r.Header.Get("header4"),
		}

		marshal.WriteJSON(w, r, headers)
	}

	server := httptest.NewServer(http.HandlerFunc(h))
	defer server.Close()

	client := retriable.New()
	require.NotNil(t, client)

	client.WithHeaders(map[string]string{
		"header1": "val1",
		"header2": "val2",
	})

	client.AddHeader("header3", "val3")

	t.Run("clientHeaders", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})

		status, err := client.Get(context.Background(), []string{server.URL}, "/v1/test", w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)

		var headers map[string]string
		require.NoError(t, json.Unmarshal(w.Bytes(), &headers))

		assert.Equal(t, "val1", headers["h1"])
		assert.Equal(t, "val2", headers["h2"])
		assert.Equal(t, "val3", headers["h3"])
		assert.Empty(t, headers["h4"])
	})

	t.Run("call.setHeader", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		// set custom header via request context
		callSpecific := map[string]string{
			"header4": "val4",
		}
		ctx := retriable.WithHeaders(context.Background(), callSpecific)
		status, err := client.Get(ctx, []string{server.URL}, "/v1/test", w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)

		var headers map[string]string
		require.NoError(t, json.Unmarshal(w.Bytes(), &headers))

		assert.Equal(t, "val1", headers["h1"])
		assert.Equal(t, "val2", headers["h2"])
		assert.Equal(t, "val3", headers["h3"])
		assert.Equal(t, "val4", headers["h4"])
	})

	t.Run("from request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/test", nil)
		require.NoError(t, err)
		req.Header.Set("header4", "val4")
		req.Header.Set(header.Accept, "custom")
		req.Header.Set(header.ContentType, "test")

		w := bytes.NewBuffer([]byte{})
		ctx := retriable.PropagateHeadersFromRequest(context.Background(), req, header.Accept, "header4", header.ContentType)

		status, err := client.Get(ctx, []string{server.URL}, "/v1/test", w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)

		var headers map[string]string
		require.NoError(t, json.Unmarshal(w.Bytes(), &headers))

		assert.Equal(t, "val4", headers["h4"])
		assert.Equal(t, "custom", headers[header.Accept])
		assert.Equal(t, "test", headers[header.ContentType])
	})
}

func Test_Retriable500(t *testing.T) {
	h := makeTestHandler(t, "/v1/test", http.StatusInternalServerError, `{
		"error": "bug!"
	}`)
	server := httptest.NewServer(h)
	defer server.Close()

	client := retriable.New()
	require.NotNil(t, client)

	hosts := []string{server.URL}

	w := bytes.NewBuffer([]byte{})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	status, err := client.Get(ctx, hosts, "/v1/test", w)
	require.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, status)
}

func Test_RetriableMulti500Error(t *testing.T) {
	errResponse := `{
	"code": "unexpected",
	"message": "internal server error"
}`

	h := makeTestHandler(t, "/v1/test", http.StatusInternalServerError, errResponse)
	server1 := httptest.NewServer(h)
	defer server1.Close()

	server2 := httptest.NewServer(h)
	defer server2.Close()

	client := retriable.New()
	require.NotNil(t, client)

	hosts := []string{server1.URL, server2.URL}

	w := bytes.NewBuffer([]byte{})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	status, err := client.Get(ctx, hosts, "/v1/test", w)
	assert.Equal(t, http.StatusInternalServerError, status)
	require.Error(t, err)
	assert.Equal(t, "unexpected: internal server error", err.Error())
}

func Test_RetriableMulti500Custom(t *testing.T) {
	errResponse := `{
	"error": "bug!"
}`

	h := makeTestHandler(t, "/v1/test", http.StatusInternalServerError, errResponse)
	server1 := httptest.NewServer(h)
	defer server1.Close()

	server2 := httptest.NewServer(h)
	defer server2.Close()

	client := retriable.New()
	require.NotNil(t, client)

	hosts := []string{server1.URL, server2.URL}

	w := bytes.NewBuffer([]byte{})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	status, err := client.Get(ctx, hosts, "/v1/test", w)
	assert.Equal(t, http.StatusInternalServerError, status)
	require.Error(t, err)
	assert.Equal(t, errResponse, err.Error())
}

func Test_RetriableTimeout(t *testing.T) {
	h := makeTestHandlerSlow(t, "/v1/test", http.StatusInternalServerError, 1*time.Second, `{
		"error": "bug!"
	}`)
	server1 := httptest.NewServer(h)
	defer server1.Close()

	server2 := httptest.NewServer(h)
	defer server2.Close()

	client := retriable.New()
	require.NotNil(t, client)

	hosts := []string{server1.URL, server2.URL}

	w := bytes.NewBuffer([]byte{})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	status, err := client.Get(ctx, hosts, "/v1/test", w)
	require.Error(t, err)
	assert.Equal(t, 0, status)
	exp1 := fmt.Sprintf("unexpected: Get %s/v1/test: context deadline exceeded", server1.URL)
	exp2 := fmt.Sprintf("unexpected: Get %s/v1/test: context deadline exceeded", server2.URL)
	assert.Contains(t, err.Error(), exp1)
	assert.Contains(t, err.Error(), exp2)
}

func Test_DecodeResponse(t *testing.T) {
	res := http.Response{StatusCode: http.StatusNotFound, Body: ioutil.NopCloser(bytes.NewBufferString(`{"code":"MY_CODE","message":"doesn't exist"}`))}
	c := retriable.New()

	var body map[string]string
	sc, err := c.DecodeResponse(&res, &body)
	require.Equal(t, res.StatusCode, sc)
	require.Error(t, err)

	ge, ok := err.(*httperror.Error)
	require.True(t, ok, "Expecting decodeResponse to map a valid error to the Error struct, but was %T %v", err, err)
	assert.Equal(t, "MY_CODE", ge.Code)
	assert.Equal(t, "doesn't exist", ge.Message)
	assert.Equal(t, http.StatusNotFound, ge.HTTPStatus)

	// if the body isn't valid json, we should get returned a json parser error, as well as the body
	invalidResponse := `["foo"}`
	res.Body = ioutil.NopCloser(bytes.NewBufferString(invalidResponse))
	sc, err = c.DecodeResponse(&res, &body)
	require.Error(t, err)
	assert.Equal(t, invalidResponse, err.Error())

	// error body is valid json, but missing the error field
	res.Body = ioutil.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`))
	sc, err = c.DecodeResponse(&res, &body)
	assert.Error(t, err)
	assert.Equal(t, "{\"foo\":\"bar\"}", err.Error())

	// statusCode < 300, with a decodeable body
	res.StatusCode = http.StatusOK
	res.Body = ioutil.NopCloser(bytes.NewBufferString(`{"foo":"baz"}`))
	sc, err = c.DecodeResponse(&res, &body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, sc)
	assert.Equal(t, "baz", body["foo"], "decodeResponse hasn't correctly decoded the payload, got %+v", body)

	xlog.SetGlobalLogLevel(xlog.TRACE)

	// statusCode < 300, with a parsing error
	res.Body = ioutil.NopCloser(bytes.NewBufferString(`[}`))
	sc, err = c.DecodeResponse(&res, &body)
	assert.Equal(t, http.StatusOK, sc, "decodeResponse returned unexpected statusCode, expecting 200")
	assert.Error(t, err)
	assert.Equal(t, "unable to decode body response to (*map[string]string) type: invalid character '}' looking for beginning of value", err.Error())
}

func makeTestHandler(t *testing.T, expURI string, status int, responseBody string) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expURI, r.RequestURI, "received wrong URI")
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		io.WriteString(w, responseBody)
	}
	return http.HandlerFunc(h)
}

func makeTestHandlerSlow(t *testing.T, expURI string, status int, delay time.Duration, responseBody string) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expURI, r.RequestURI, "received wrong URI")
		if status == 0 {
			status = http.StatusOK
		}

		time.Sleep(delay)
		w.WriteHeader(status)
		io.WriteString(w, responseBody)
	}
	return http.HandlerFunc(h)
}
