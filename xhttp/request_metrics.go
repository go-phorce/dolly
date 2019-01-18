package xhttp

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-phorce/dolly/algorithms/math"
	"github.com/go-phorce/dolly/metrics"
	"github.com/go-phorce/dolly/metrics/tags"
	"github.com/go-phorce/dolly/xhttp/identity"
)

// a http.Handler that records execution metrics of the wrapper handler
type requestMetrics struct {
	handler       http.Handler
	responseCodes []string
	extraMetrics  map[string]string
}

// NewRequestMetrics creates a wrapper handler to produce metrics for each request
func NewRequestMetrics(h http.Handler) http.Handler {
	return NewRequestWithExtraMetrics(h, nil)
}

// NewRequestWithExtraMetrics creates a wrapper handler to produce additional metrics for each request
func NewRequestWithExtraMetrics(h http.Handler, extraMetrics map[string]string) http.Handler {
	rm := requestMetrics{
		handler:       h,
		responseCodes: make([]string, 599),
		extraMetrics:  extraMetrics,
	}
	for idx := range rm.responseCodes {
		rm.responseCodes[idx] = strconv.Itoa(idx)
	}
	return &rm
}

func (rm *requestMetrics) statusCode(statusCode int) string {
	if (statusCode < len(rm.responseCodes)) && (statusCode > 0) {
		return rm.responseCodes[statusCode]
	}

	return strconv.Itoa(statusCode)
}

var (
	keyForHTTPReqPerf       = []string{"http", "request", "perf"}
	keyForHTTPReqSuccessful = []string{"http", "request", "status", "successful"}
	keyForHTTPReqFailed     = []string{"http", "request", "status", "failed"}
)

func (rm *requestMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UTC()
	rc := NewResponseCapture(w)
	rm.handler.ServeHTTP(rc, r)
	role := identity.ForRequest(r).Identity().Role()
	sc := rc.StatusCode()

	tags := []metrics.Tag{
		{Name: tags.Method, Value: r.Method},
		{Name: tags.Role, Value: role},
		{Name: tags.Status, Value: rm.statusCode(sc)},
		{Name: tags.URI, Value: r.URL.Path},
	}

	// add additional metrics
	keys := math.SortedKeys(rm.extraMetrics)
	for _, n := range keys {
		tags = append(tags, metrics.Tag{Name: n, Value: rm.extraMetrics[n]})
	}

	if sc >= 400 {
		metrics.IncrCounter(keyForHTTPReqFailed, 1, tags...)
	} else {
		metrics.MeasureSince(keyForHTTPReqPerf, start, tags...)
		metrics.IncrCounter(keyForHTTPReqSuccessful, 1, tags...)
	}
}
