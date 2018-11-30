// Package identity extracts the callers contextual identity information from the HTTP/TLS
// requests and exposes them for access via the generalized go context model.
package identity

import (
	"context"
	"net/http"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/netutil"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "xhttp/context")

type contextKey int

const (
	keyContext contextKey = iota
	keyIdentity
)

var (
	nodeInfo      netutil.NodeInfo
	roleExtractor ExtractRole = defaultExtractRole
)

// RequestContext represents user contextual information about a request being processed by the server,
// it includes identity, CorrelationID [for cross system request correlation].
type RequestContext struct {
	identity      Identity
	correlationID string
	clientIP      string
}

// Context represents user contextual information about a request being processed by the server,
// it includes identity, CorrelationID [for cross system request correlation].
type Context interface {
	Identity() Identity
	CorrelationID() string
	ClientIP() string
}

func init() {
	n, err := netutil.NewNodeInfo(nil)
	if err != nil {
		logger.Panicf("context package not initialized: %s", errors.ErrorStack(err))
	}
	SetGlobalNodeInfo(n)
}

// SetGlobalNodeInfo applies NodeInfo for the application
func SetGlobalNodeInfo(n netutil.NodeInfo) {
	if n == nil {
		logger.Panic("NodeInfo must not be nil")
	}
	nodeInfo = n
}

// SetGlobalRoleExtractor applies ExtractRole for the application
func SetGlobalRoleExtractor(e ExtractRole) {
	if e == nil {
		logger.Panic("ExtractRole must not be nil")
	}
	roleExtractor = e
}

// ForRequest returns the full context ascocicated with this http request.
func ForRequest(r *http.Request) *RequestContext {
	v := r.Context().Value(keyContext)
	if v == nil {
		return &RequestContext{
			identity:      extractIdentityFromRequest(r),
			correlationID: extractCorrelationID(r),
			clientIP:      ClientIPFromRequest(r),
		}
	}
	return v.(*RequestContext)
}

// NewContextHandler returns a handler that will extact the role & contextID from the request
// and stash them away in the request context for later handlers to use.
// Also adds header to indicate which host is currently servicing the request
func NewContextHandler(delegate http.Handler) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		var rctx *RequestContext
		v := r.Context().Value(keyContext)
		if v == nil {
			rctx = &RequestContext{
				identity:      extractIdentityFromRequest(r),
				correlationID: extractCorrelationID(r),
				clientIP:      ClientIPFromRequest(r),
			}
			r = r.WithContext(context.WithValue(r.Context(), keyContext, rctx))
		} else {
			rctx = v.(*RequestContext)
		}

		// Set XHostname on the response
		w.Header().Set(header.XHostname, nodeInfo.HostName())
		w.Header().Set(header.XCorrelationID, rctx.correlationID)

		delegate.ServeHTTP(w, r)
	}
	return http.HandlerFunc(h)
}

// Identity returns request's identity
func (c *RequestContext) Identity() Identity {
	return c.identity
}

// CorrelationID returns request's CorrelationID, extracted from X-CorrelationID header.
// If it was not provided by the client, the a random will be generated.
func (c *RequestContext) CorrelationID() string {
	return c.correlationID
}

// ClientIP returns request's IP
func (c *RequestContext) ClientIP() string {
	return c.clientIP
}

// extractCorrelationID will find or create a requestID for this http request.
func extractCorrelationID(req *http.Request) string {
	corID := req.Header.Get(header.XCorrelationID)
	if corID == "" {
		corID = guid.MustCreate()
	}
	return corID
}
