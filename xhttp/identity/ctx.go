// Package identity extracts the callers contextual identity information from the HTTP/TLS
// requests and exposes them for access via the generalized go context model.
package identity

import (
	"context"
	"crypto/x509"
	"net/http"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/netutil"
	"github.com/go-phorce/dolly/xhttp"
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
	emptyContext  *RequestContext
	nodeInfo      netutil.NodeInfo
	roleExtractor ExtractRole
)

// RequestContext represents user contextual information about a request being processed by the server,
// it includes identity, CorrelationID [for cross system request correlation].
type RequestContext struct {
	identity      Identity
	correlationID string
	hostname      string
	ipaddr        string
}

// Context represents user contextual information about a request being processed by the server,
// it includes identity, CorrelationID [for cross system request correlation].
type Context interface {
	Identity() Identity
	CorrelationID() string
	Host() string
	IP() string
}

func init() {
	Initialize(nil, nil)
}

// Initialize allows to customize NodeInfo and ExtractRoleName
func Initialize(n netutil.NodeInfo, e ExtractRole) {
	roleExtractor = e
	nodeInfo = n

	if nodeInfo == nil {
		n, err := netutil.NewNodeInfo(nil)
		if err != nil {
			logger.Panicf("context package not initialized: %s", errors.ErrorStack(err))
		}
		nodeInfo = n
	}

	if roleExtractor == nil {
		roleExtractor = extractCommonName
	}

	emptyContext = New("", "", "", "", "")
}

// ForRequest returns the full context ascocicated with this http request.
func ForRequest(r *http.Request) *RequestContext {
	v := r.Context().Value(keyContext)
	if v == nil {
		return emptyContext
	}
	return v.(*RequestContext)
}

// NewForRole returns a new context for a task asociated with the role itself,
// rather than a client request
func NewForRole(role string) *RequestContext {
	return New(role, nodeInfo.HostName(), "", nodeInfo.HostName(), nodeInfo.LocalIP())
}

// NewForClientCert returns a new context for Client's TLS cert
func NewForClientCert(c *x509.Certificate) *RequestContext {
	identity := NewIdentityFromCert(c)
	return New(identity.Role(), identity.Name(), "", nodeInfo.HostName(), nodeInfo.LocalIP())
}

// NewContextHandler returns a handler that will extact the role & contextID from the request
// and stash them away in the request context for later handlers to use.
// Also adds header to indicate which host is currently servicing the request
func NewContextHandler(delegate http.Handler) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		ctx := &RequestContext{
			identity:      extractIdentityFromRequest(r),
			correlationID: extractCorrelationID(r),
			hostname:      extractHostname(r),
			ipaddr:        xhttp.ClientIPFromRequest(r),
		}

		c := context.WithValue(r.Context(), keyContext, ctx)

		// Set XHostname on the response
		w.Header().Set(header.XHostname, nodeInfo.HostName())
		w.Header().Set(header.XCorrelationID, ctx.correlationID)

		delegate.ServeHTTP(w, r.WithContext(c))
	}
	return http.HandlerFunc(h)
}

// WithTestContext is used in unit tests to set HTTP context
func WithTestContext(r *http.Request, identity Identity, correlationID string) *http.Request {
	ctx := &RequestContext{
		identity:      identity,
		correlationID: correlationID,
		hostname:      nodeInfo.HostName(),
		ipaddr:        nodeInfo.LocalIP(),
	}
	c := r.Context()
	c = context.WithValue(c, keyContext, ctx)

	return r.WithContext(c)
}

// New returns a new Context with the supplied identity & request identifiers
// typically you should be using ForRequest or Context to get a context
// this primarily exists for tests
func New(role, commonName, correlationID, host, ip string) *RequestContext {
	if correlationID == "" {
		correlationID = guid.MustCreate()
	}
	if host == "" {
		host = nodeInfo.HostName()
	}
	if ip == "" {
		ip = nodeInfo.LocalIP()
	}
	return &RequestContext{
		identity:      NewIdentity(role, commonName),
		correlationID: correlationID,
		hostname:      host,
		ipaddr:        ip,
	}
}

func (c *RequestContext) copy() *RequestContext {
	return &RequestContext{
		identity:      c.identity,
		correlationID: c.correlationID,
		hostname:      c.hostname,
		ipaddr:        c.ipaddr,
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

// Host returns request's hoste name
func (c *RequestContext) Host() string {
	return c.hostname
}

// IP returns request's IP
func (c *RequestContext) IP() string {
	return c.ipaddr
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

// extractHost extracts the client Host from req, if present.
func extractHostname(req *http.Request) string {
	// TODO: check headers "X-Forwarded-For"
	host := req.Header.Get(header.XHostname)
	if host == "" {
		host = req.Host
	}
	return host
}
