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
)

var (
	nodeInfo      netutil.NodeInfo
	roleExtractor ExtractRole = extractRoleFromPKIX
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
	nodeInfo = n
}

// Initialize allows to customize NodeInfo and ExtractRoleName
func Initialize(n netutil.NodeInfo, e ExtractRole) {
	if n != nil {
		nodeInfo = n
	}

	if e != nil {
		roleExtractor = e
	}
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
		ctx := &RequestContext{
			identity:      extractIdentityFromRequest(r),
			correlationID: extractCorrelationID(r),
			clientIP:      ClientIPFromRequest(r),
		}

		c := context.WithValue(r.Context(), keyContext, ctx)

		// Set XHostname on the response
		w.Header().Set(header.XHostname, nodeInfo.HostName())
		w.Header().Set(header.XCorrelationID, ctx.correlationID)

		delegate.ServeHTTP(w, r.WithContext(c))
	}
	return http.HandlerFunc(h)
}

func (c *RequestContext) copy() *RequestContext {
	return &RequestContext{
		identity:      c.identity,
		correlationID: c.correlationID,
		clientIP:      c.clientIP,
	}
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

// WithCorrelationID sets correlationID
func (c *RequestContext) WithCorrelationID(correlationID string) *RequestContext {
	copy := c.copy()
	copy.correlationID = correlationID
	return copy
}

// extractCorrelationID will find or create a requestID for this http request.
func extractCorrelationID(req *http.Request) string {
	corID := req.Header.Get(header.XCorrelationID)
	if corID == "" {
		corID = guid.MustCreate()
	}
	return corID
}
