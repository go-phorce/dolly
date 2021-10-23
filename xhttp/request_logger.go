package xhttp

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xlog"
)

var errNoHandler = errors.New("NewRequestLogger was supplied a nil handler to delegate to")

// AdditionalLogExtractor allows for a user of RequestLogger to extract and have additional fields recorded in the log
type AdditionalLogExtractor func(resp *ResponseCapture, req *http.Request) []string

// RequestLoggerOption is an option that can be passed to New().
type RequestLoggerOption option
type option func(c *configuration)

// LoggerSkipPath allows to skip a log for specified Path and Agent
type LoggerSkipPath struct {
	Path  string `json:"path,omitempty" yaml:"path,omitempty"`
	Agent string `json:"agent,omitempty" yaml:"agent,omitempty"`
}

type configuration struct {
	skippaths   []LoggerSkipPath
	prefix      string
	granularity int64
	extractor   AdditionalLogExtractor
	logger      xlog.Logger
}

// WithLoggerSkipPaths is an Option allows to skip logs on path/agent match
func WithLoggerSkipPaths(value []LoggerSkipPath) RequestLoggerOption {
	return func(c *configuration) {
		c.skippaths = value
	}
}

// RequestLogger is a http.Handler that logs requests and forwards them on down the chain.
type RequestLogger struct {
	handler http.Handler
	cfg     configuration
}

// NewRequestLogger create a new RequestLogger handler, requests are chained to the supplied handler.
// The log includes the clock time to handle the request, with specified granularity (e.g. time.Millisecond).
// The generated Log lines are in the format
// <prefix>:<HTTP Method>:<ClientCertSubjectCN>:<Path>:<RemoteIP>:<RemotePort>:<StatusCode>:<HTTP Version>:<Response Body Size>:<Request Duration>:<Additional Fields>
// skippath parameter allows to specify a list of paths to not log.
func NewRequestLogger(handler http.Handler, prefix string, additionalEntries AdditionalLogExtractor, granularity time.Duration, packageLogger string, opts ...RequestLoggerOption) http.Handler {
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

	cfg := configuration{
		granularity: int64(granularity),
		extractor:   additionalEntries,
		logger:      l,
		prefix:      prefix,
	}

	for _, opt := range opts {
		option(opt)(&cfg)
	}

	return &RequestLogger{
		handler: handler,
		cfg:     cfg,
	}
}

// ServeHTTP implements the http.Handler interface. We wrap the call to the
// real handler to collect info about the response, and then write out the log line
func (l *RequestLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UTC()
	rw := NewResponseCapture(w)
	l.handler.ServeHTTP(rw, r)

	agent := r.Header.Get(header.UserAgent)
	if agent == "" {
		agent = "no-agent"
	}

	for _, skip := range l.cfg.skippaths {
		pathMatch := skip.Path == "*" || r.URL.Path == skip.Path
		agentMatch := skip.Agent == "*" || strings.Contains(agent, skip.Agent)
		if pathMatch && agentMatch {
			return
		}
	}

	dur := time.Since(start)
	clientCertUser := l.client(r)
	extra := ""
	if l.cfg.extractor != nil {
		fields := l.cfg.extractor(rw, r)
		if len(fields) > 0 {
			extra = ":" + strings.Join(fields, ":")
		}
	}
	l.cfg.logger.Infof("%s:%s:%s:%s:%s:%d:%d.%d:%d:%v:%q%s",
		l.cfg.prefix,
		clientCertUser,
		r.Method,
		r.URL.Path,
		r.RemoteAddr,
		rw.statusCode,
		r.ProtoMajor, r.ProtoMinor,
		rw.bodySize,
		dur.Nanoseconds()/l.cfg.granularity,
		agent,
		extra)
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
