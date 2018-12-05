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
	"strings"
	"testing"
	"time"

	"github.com/go-phorce/dolly/rest/tlsconfig"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/go-phorce/dolly/xhttp/marshal"
	"github.com/go-phorce/dolly/xhttp/retriable"
	"github.com/go-phorce/dolly/xlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projectPath = "../../"

func Test_New(t *testing.T) {
	p := &retriable.Policy{
		TotalRetryLimit: 5,
	}

	// create without options
	c := retriable.New().WithName("test")
	assert.NotNil(t, c)
	assert.NotNil(t, c.WithPolicy(p))
	assert.NotNil(t, c.WithTLS(nil))
	assert.NotNil(t, c.WithTransport(nil))

	// create with options
	c = retriable.New(
		retriable.WithName("test"),
		retriable.WithPolicy(p),
		retriable.WithTLS(nil),
		retriable.WithTransport(nil),
	)
	assert.NotNil(t, c)
	c.AddHeader("test", "for client")

	// TLS
	clientTls, err := tlsconfig.NewClientTLSFromFiles(
		projectPath+"etc/dev/certs/test_dolly_client.pem",
		projectPath+"etc/dev/certs/test_dolly_client-key.pem",
		projectPath+"etc/dev/certs/rootca/test_dolly_root_CA.pem")
	require.NoError(t, err)
	c = retriable.New().WithTLS(clientTls)
	assert.NotNil(t, c)
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

	client := retriable.New().
		WithHeaders(map[string]string{
			"X-Test-Token": "token1",
		}).
		WithPolicy(&retriable.Policy{
			TotalRetryLimit: 2,
			RequestTimeout:  time.Second,
		})
	require.NotNil(t, client)

	hosts := []string{server.URL}

	t.Run("GET WithTimeout", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		h, status, err := client.Request(ctx, http.MethodGet, hosts, "/v1/test", nil, w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "retriable", h.Get("X-Test-Header"))
		assert.Equal(t, "token1", h.Get("X-Test-Token"))
	})

	t.Run("PUT", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		_, status, err := client.Request(nil, http.MethodPut, hosts, "/v1/test", []byte("{}"), w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("POST", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		_, status, err := client.Request(nil, http.MethodPost, hosts, "/v1/test", []byte("{}"), w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("POST Empty body", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		_, status, err := client.Request(nil, http.MethodPost, hosts, "/v1/test", nil, w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("DELETE", func(t *testing.T) {
		// override per cal headers
		ctx := retriable.WithHeaders(nil, map[string]string{
			"X-Test-Token": "token2",
		})

		w := bytes.NewBuffer([]byte{})
		h, status, err := client.Request(ctx, http.MethodDelete, hosts, "/v1/test", nil, w)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "token2", h.Get("X-Test-Token"))
	})

	t.Run("HEAD", func(t *testing.T) {
		// override per cal headers
		ctx := retriable.WithHeaders(nil, map[string]string{
			"X-Test-Token": "token2",
		})

		h, status, err := client.Head(ctx, hosts, "/v1/test")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "token2", h.Get("X-Test-Token"))
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

		_, status, err := client.Request(context.Background(), http.MethodGet, []string{server.URL}, "/v1/test", nil, w)
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
		_, status, err := client.Request(ctx, http.MethodGet, []string{server.URL}, "/v1/test", nil, w)
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

		_, status, err := client.Request(ctx, http.MethodGet, []string{server.URL}, "/v1/test", nil, w)
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

	_, status, err := client.Request(ctx, http.MethodGet, hosts, "/v1/test", nil, w)
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

	_, status, err := client.Request(ctx, http.MethodGet, hosts, "/v1/test", nil, w)
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

	_, status, err := client.Request(ctx, http.MethodGet, hosts, "/v1/test", nil, w)
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

	_, status, err := client.Request(ctx, http.MethodGet, hosts, "/v1/test", nil, w)
	require.Error(t, err)
	assert.Equal(t, 0, status)
	exp1 := fmt.Sprintf("unexpected: Get %s/v1/test: context deadline exceeded", server1.URL)
	exp2 := fmt.Sprintf("unexpected: Get %s/v1/test: context deadline exceeded", server2.URL)
	assert.Contains(t, err.Error(), exp1)
	assert.Contains(t, err.Error(), exp2)

	// set policy on the client
	client.WithPolicy(&retriable.Policy{
		TotalRetryLimit: 2,
		RequestTimeout:  100 * time.Millisecond,
	})
	_, status, err = client.Request(nil, http.MethodGet, hosts, "/v1/test", nil, w)
	require.Error(t, err)
	assert.Contains(t, err.Error(), exp1)
	assert.Contains(t, err.Error(), exp2)
}

func Test_Retriable_DoWithTimeout(t *testing.T) {
	h := makeTestHandlerSlow(t, "/v1/test/do", http.StatusInternalServerError, 1*time.Second, `{
		"error": "bug"
	}`)
	server1 := httptest.NewServer(h)
	defer server1.Close()

	client := retriable.New()
	require.NotNil(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequest(http.MethodPost, server1.URL+"/v1/test/do", strings.NewReader(`{"test":true}`))
	require.NoError(t, err)
	req = req.WithContext(ctx)

	client.WithPolicy(&retriable.Policy{
		TotalRetryLimit: 2,
		RequestTimeout:  100 * time.Millisecond,
	})
	_, err = client.Do(req)
	require.Error(t, err)
	exp1 := fmt.Sprintf("Post %s/v1/test/do: context deadline exceeded", server1.URL)
	assert.Contains(t, err.Error(), exp1)
}

func Test_Retriable_DoWithRetry(t *testing.T) {
	count := 0
	h := func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		if 2 >= count {
			status = http.StatusServiceUnavailable
		}
		count++
		w.WriteHeader(status)
		io.WriteString(w, fmt.Sprintf(`{"count": "%d"}`, count))
	}

	server1 := httptest.NewServer(http.HandlerFunc(h))
	defer server1.Close()

	client := retriable.New()
	require.NotNil(t, client)

	req, err := http.NewRequest(http.MethodPost, server1.URL+"/v1/test/do", strings.NewReader(`{"test":true}`))
	require.NoError(t, err)

	client.WithPolicy(&retriable.Policy{
		TotalRetryLimit: 3,
		RequestTimeout:  1 * time.Second,
		Retries: map[int]retriable.ShouldRetry{
			http.StatusServiceUnavailable: func(_ *http.Request, re_sp *http.Response, _ error, retries int) (bool, time.Duration, string) {
				return (2 >= retries), time.Millisecond, "retry"
			},
		},
	})

	res, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, 4, count)
}

func Test_DecodeResponse(t *testing.T) {
	res := http.Response{StatusCode: http.StatusNotFound, Body: ioutil.NopCloser(bytes.NewBufferString(`{"code":"MY_CODE","message":"doesn't exist"}`))}
	c := retriable.New()

	var body map[string]string
	_, sc, err := c.DecodeResponse(&res, &body)
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
	_, sc, err = c.DecodeResponse(&res, &body)
	require.Error(t, err)
	assert.Equal(t, invalidResponse, err.Error())

	// error body is valid json, but missing the error field
	res.Body = ioutil.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`))
	_, sc, err = c.DecodeResponse(&res, &body)
	assert.Error(t, err)
	assert.Equal(t, "{\"foo\":\"bar\"}", err.Error())

	// statusCode < 300, with a decodeable body
	res.StatusCode = http.StatusOK
	res.Body = ioutil.NopCloser(bytes.NewBufferString(`{"foo":"baz"}`))
	_, sc, err = c.DecodeResponse(&res, &body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, sc)
	assert.Equal(t, "baz", body["foo"], "decodeResponse hasn't correctly decoded the payload, got %+v", body)

	xlog.SetGlobalLogLevel(xlog.TRACE)

	// statusCode < 300, with a parsing error
	res.Body = ioutil.NopCloser(bytes.NewBufferString(`[}`))
	_, sc, err = c.DecodeResponse(&res, &body)
	assert.Equal(t, http.StatusOK, sc, "decodeResponse returned unexpected statusCode, expecting 200")
	assert.Error(t, err)
	assert.Equal(t, "unable to decode body response to (*map[string]string) type: invalid character '}' looking for beginning of value", err.Error())
}

func makeTestHandler(t *testing.T, expURI string, status int, responseBody string) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expURI, r.URL.Path, "received wrong URI")
		if status == 0 {
			status = http.StatusOK
		}
		w.Header().Add("X-Test-Header", "retriable")
		w.Header().Add("X-Test-Token", r.Header.Get("X-Test-Token"))
		w.WriteHeader(status)
		io.WriteString(w, responseBody)
	}
	return http.HandlerFunc(h)
}

func makeTestHandlerSlow(t *testing.T, expURI string, status int, delay time.Duration, responseBody string) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expURI, r.URL.Path, "received wrong URI")
		if status == 0 {
			status = http.StatusOK
		}

		time.Sleep(delay)
		w.WriteHeader(status)
		io.WriteString(w, responseBody)
	}
	return http.HandlerFunc(h)
}
