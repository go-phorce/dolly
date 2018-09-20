package retriable_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-phorce/dolly/xhttp/context"
	"github.com/go-phorce/dolly/xhttp/retriable"
	"github.com/stretchr/testify/assert"
)

var testCtx = context.NewForRole("TestRole")

func Test_New(t *testing.T) {
	p := &retriable.Policy{
		TotalRetryLimit: 5,
	}

	c, err := retriable.New("test", nil)
	assert.Nil(t, err)
	assert.NotNil(t, c)

	c.SetRetryPolicy(p)
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
		{false, "retry-limit-exceeded", 10, 503, nil},
		// 502 is bad gateway, which is returned during leader transitions
		{true, "gateway", 0, 502, nil},
		{true, "gateway", 1, 502, nil},
		{true, "gateway", 9, 502, nil},
		{false, "retry-limit-exceeded", 10, 502, nil},
		// regardless of config, other status codes shouldn't get retries
		{false, "success", 0, 200, nil},
		{false, "non-retriable error", 0, 400, nil},
		{false, "non-retriable error", 0, 401, nil},
		{false, "non-retriable error", 0, 404, nil},
		{false, "non-retriable error", 0, 500, nil},
		// connection
		{true, "connection", 0, 0, errors.New("some error")},
		{true, "connection", 5, 0, errors.New("some error")},
		{false, "connection", 6, 0, errors.New("some error")},
	}

	p := retriable.NewDefaultPolicy()
	for _, tc := range tcases {
		t.Run(fmt.Sprintf("%s: %d, %d, %t:", tc.reason, tc.retries, tc.statusCode, tc.expected), func(t *testing.T) {
			res := &http.Response{StatusCode: tc.statusCode}
			should, _, reason := p.ShouldRetry(res, tc.err, tc.retries)
			assert.Equal(t, tc.expected, should)
			assert.Equal(t, tc.reason, reason)
		})
	}
}
