package xhttp

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-phorce/dolly/metrics"
	"github.com/go-phorce/dolly/metrics/tags"
)

// a http.Handler that records execution metrics of the wrapper handler
type requestMetrics struct {
	handler       http.Handler
	responseCodes []string
}

// NewRequestMetrics creates a wrapper handler to produce metrics for each request
func NewRequestMetrics(h http.Handler) http.Handler {
	rm := requestMetrics{
		handler:       h,
		responseCodes: make([]string, 599),
	}
	for idx := range rm.responseCodes {
		rm.responseCodes[idx] = strconv.Itoa(idx)
	}
	return &rm
}

func (rm *requestMetrics) statusCode(reqURI string, statusCode int) string {
	if (statusCode < len(rm.responseCodes)) && (statusCode > 0) {
		return rm.responseCodes[statusCode]
	}
	logger.Warningf("request for %s returned unexpected status code of %d [expected to be <599]",
		reqURI, statusCode)
	return strconv.Itoa(statusCode)
}

func (rm *requestMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UTC()
	rc := NewResponseCapture(w)
	rm.handler.ServeHTTP(rc, r)

	metricKey := []string{
		"http", "request",
		tags.Separator,
		tags.Method, r.Method,
		tags.Status, rm.statusCode(r.RequestURI, rc.StatusCode()),
		tags.URI, r.RequestURI,
	}
	metrics.MeasureSince(metricKey, start)
}
