package xhttp

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-phorce/dolly/xlog"
)

var errNoHandler = errors.New("NewRequestLogger was supplied a nil handler to delegate to")

// AdditionalLogExtractor allows for a user of RequestLogger to extract and have additional fields recorded in the log
type AdditionalLogExtractor func(resp *ResponseCapture, req *http.Request) []string

// RequestLogger is a http.Handler that logs requests and forwards them on down the chain.
type RequestLogger struct {
	handler     http.Handler
	prefix      string
	granularity int64
	extractor   AdditionalLogExtractor
	logger      xlog.Logger
}

// NewRequestLogger create a new RequestLogger handler, requests are chained to the supplied handler.
// The log includes the clock time to handle the request, with specified granularity (e.g. time.Millisecond).
// The generated Log lines are in the format
// <prefix>:<HTTP Method>:<ClientCertSubjectCN>:<Path>:<RemoteIP>:<RemotePort>:<StatusCode>:<HTTP Version>:<Response Body Size>:<Request Duration>:<Additional Fields>
func NewRequestLogger(handler http.Handler, prefix string, additionalEntries AdditionalLogExtractor, granularity time.Duration, packageLogger string) http.Handler {
	if handler == nil {
		panic(errNoHandler)
	}

	l := logger
	if packageLogger != "" {
		l = xlog.NewPackageLogger(packageLogger, "xhttp")
	}

	if l == nil {
		return handler
	}
	return &RequestLogger{
		handler:     handler,
		prefix:      prefix,
		granularity: int64(granularity),
		extractor:   additionalEntries,
		logger:      l,
	}
}

// ServeHTTP implements the http.Handler interface. We wrap the call to the
// real handler to collect info about the response, and then write out the log line
func (l *RequestLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UTC()
	rw := NewResponseCapture(w)
	l.handler.ServeHTTP(rw, r)
	dur := time.Since(start)
	clientCertUser := l.client(r)
	extra := ""
	if l.extractor != nil {
		fields := l.extractor(rw, r)
		if len(fields) > 0 {
			extra = ":" + strings.Join(fields, ":")
		}
	}
	agent := r.Header.Get("User-Agent")
	if agent == "" {
		agent = "no-agent"
	}
	if rw.statusCode < 400 {
		l.logger.Infof("%s:%s:%s:%s:%s:%d:%d.%d:%d:%v:%q%s",
			l.prefix, clientCertUser, r.Method, r.URL.Path, r.RemoteAddr, rw.statusCode, r.ProtoMajor, r.ProtoMinor, rw.bodySize, dur.Nanoseconds()/l.granularity, agent, extra)
	} else {
		l.logger.Errorf("%s:%s:%s:%s:%s:%d:%d.%d:%d:%v:%q%s",
			l.prefix, clientCertUser, r.Method, r.URL.Path, r.RemoteAddr, rw.statusCode, r.ProtoMajor, r.ProtoMinor, rw.bodySize, dur.Nanoseconds()/l.granularity, agent, extra)
	}
}

func (l *RequestLogger) client(r *http.Request) string {
	if r.TLS == nil {
		return ""
	}
	pc := r.TLS.PeerCertificates
	if len(pc) == 0 {
		return ""
	}
	return pc[0].Subject.CommonName
}
