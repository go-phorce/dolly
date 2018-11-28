package xhttp

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-phorce/dolly/metrics"
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
	im := metrics.NewInmemSink(time.Minute, time.Minute*5)
	_, err := metrics.NewGlobal(metrics.DefaultConfig("test"), im)
	require.NoError(t, err)

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
		r.TLS = &tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{
				{
					Subject: pkix.Name{
						CommonName:   "dolly",
						Organization: []string{"org"},
					},
				},
			},
		}

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
	req(http.MethodPost, "/bar", http.StatusBadRequest)

	data := im.Data()
	require.NotEqual(t, 0, len(data))
	assertSample := func(key string, expectedCount int) {
		s, exists := data[0].Samples[key]
		require.True(t, exists, "Expected metric with key %s to exist, but it doesn't", key)
		assert.Equal(t, expectedCount, s.Count, "Unexpected count for metric %s", key)
	}
	assertSample("test.http.request.perf;method=GET;role=dolly;status=200;uri=/", 1)
	assertSample("test.http.request.perf;method=GET;role=dolly;status=200;uri=/foo", 1)
	assertSample("test.http.request.perf;method=POST;role=dolly;status=200;uri=/", 2)
	assertSample("test.http.request.perf;method=POST;role=dolly;status=400;uri=/", 1)
	assertSample("test.http.request.perf;method=POST;role=dolly;status=400;uri=/bar", 2)

	assertCounter := func(key string, expectedCount int) {
		s, exists := data[0].Counters[key]
		require.True(t, exists, "Expected metric with key %s to exist, but it doesn't", key)
		assert.Equal(t, expectedCount, s.Count, "Unexpected count for metric %s", key)
	}
	assertCounter("test.http.request.status.successful;method=GET;role=dolly;status=200;uri=/", 1)
	assertCounter("test.http.request.status.successful;method=GET;role=dolly;status=200;uri=/foo", 1)
	assertCounter("test.http.request.status.successful;method=POST;role=dolly;status=200;uri=/", 2)
	assertCounter("test.http.request.status.failed;method=POST;role=dolly;status=400;uri=/", 1)
	assertCounter("test.http.request.status.failed;method=POST;role=dolly;status=400;uri=/bar", 2)
}
