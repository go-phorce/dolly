package retriable_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-phorce/dolly/xhttp/retriable"
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
		{false, retriable.NonRetriableError, 0, 404, nil},
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

	w := bytes.NewBuffer([]byte{})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	status, err := client.Get(ctx, hosts, "/v1/test", w)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
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
