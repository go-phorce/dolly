package xhttp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gm "github.com/armon/go-metrics"
	"github.com/go-phorce/dolly/metrics"
	"github.com/go-phorce/dolly/xhttp/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RequestMetricsStatusCode(t *testing.T) {
	rm := NewRequestMetrics(nil).(*requestMetrics)

	assert.Equal(t, "200", rm.statusCode("/", 200))
	assert.Equal(t, "500", rm.statusCode("/", 500))
	assert.Equal(t, "700", rm.statusCode("/what", 700))
	assert.Equal(t, "0", rm.statusCode("/what", 0))
}

func Test_RequestMetrics(t *testing.T) {
	im := gm.NewInmemSink(time.Minute, time.Minute*5)
	_, err := gm.NewGlobal(gm.DefaultConfig("test"), im)
	require.NoError(t, err)

	metrics.SetProvider(metrics.NewStandardProvider())

	handlerStatusCode := 200
	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(handlerStatusCode)
		io.WriteString(w, `"Helo World"`)
	}
	rm := NewRequestMetrics(http.HandlerFunc(h))
	req := func(method, uri string, sc int) {
		r, err := http.NewRequest(method, uri, nil)
		r.RequestURI = uri
		require.NoError(t, err)
		testIdentity := identity.NewIdentity("enrollme_dev", "localhost")
		r = identity.WithTestIdentity(r, testIdentity)
		w := httptest.NewRecorder()
		handlerStatusCode = sc
		rm.ServeHTTP(w, r)
	}
	req(http.MethodGet, "/", http.StatusOK)
	req(http.MethodGet, "/foo", http.StatusOK)
	req(http.MethodPost, "/", http.StatusOK)
	req(http.MethodPost, "/", http.StatusOK)
	req(http.MethodPost, "/", http.StatusBadRequest)
	req(http.MethodPost, "/bar", http.StatusBadRequest)
	id := im.Data()
	require.NotEqual(t, 0, len(id))
	assertSample := func(key string, expectedCount int) {
		s, exists := id[0].Samples[key]
		require.True(t, exists, "Expected metric with key %s to exist, but it doesn't", key)
		assert.Equal(t, expectedCount, s.Count, "Unexpected count for metric %s", key)
	}
	assertSample("test.http.request;method=GET;role=enrollme_dev;status=200;uri=/", 1)
	assertSample("test.http.request;method=GET;role=enrollme_dev;status=200;uri=/foo", 1)
	assertSample("test.http.request;method=POST;role=enrollme_dev;status=200;uri=/", 2)
	assertSample("test.http.request;method=POST;role=enrollme_dev;status=400;uri=/", 1)
	assertSample("test.http.request;method=POST;role=enrollme_dev;status=400;uri=/bar", 1)
}
