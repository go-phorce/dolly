// Package context extracts the callers contextual information from the HTTP/TLS
// requests and exposes them for access via the generalized go context model.
package context

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/netutil"
	"github.com/go-phorce/dolly/xhttp"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "xhttp/context")

// ExtractRoleName will parse out from the supplied Name the clients roleName
type ExtractRoleName func(*pkix.Name) string

type contextKey int

const (
	keyContext contextKey = iota
)

var (
	emptyContext  Context
	nodeInfo      netutil.NodeInfo
	roleExtractor ExtractRoleName
)

type requestContext struct {
	identity  ClientIdentity
	requestID string
	hostname  string
	ipaddr    string
	accepts   string
	headers   map[string]string
}

// Context represents user contextual information about a request being processed by the server,
// it includes identity, requestID [for cross system request correlation].
type Context interface {
	Accepts() string
	Identity() ClientIdentity
	RequestID() string
	Host() string
	IP() string
	Headers() map[string]string

	WithRequestID(requestID string) Context
	WithAccepts(accepts string) Context
	WithHeaders(map[string]string) Context

	SetHeaders(r *http.Request)
}

func init() {
	Initialize(nil, nil)
}

// Initialize allows to customize NodeInfo and ExtractRoleName
func Initialize(n netutil.NodeInfo, e ExtractRoleName) {
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
		roleExtractor = extractRoleName
	}

	emptyContext = New("", "", "", "", "")
}

// RoleFromRequest is a authz.RoleMapper that extracts the role from the request context
func RoleFromRequest(r *http.Request) string {
	return ForRequest(r).Identity().Role()
}

// CorrelationIDFromRequest will extract a request/context-ID for this particular request
func CorrelationIDFromRequest(r *http.Request) string {
	return ForRequest(r).RequestID()
}

// ForRequest returns the full context ascocicated with this http request.
func ForRequest(r *http.Request) Context {
	v := r.Context().Value(keyContext)
	if v == nil {
		return emptyContext
	}
	return v.(Context)
}

// NewForRole returns a new context for a task asociated with the role itself,
// rather than a client request
func NewForRole(role string) Context {
	return New(role, nodeInfo.HostName(), "", nodeInfo.HostName(), nodeInfo.LocalIP())
}

// NewForClientCert returns a new context for Client's TLS cert
func NewForClientCert(c *x509.Certificate) Context {
	identity := NewIdentityFromCert(c)
	return New(identity.Role(), identity.CommonName(), "", nodeInfo.HostName(), nodeInfo.LocalIP())
}

// NewContextHandler returns a handler that will extact the role & contextID from the request
// and stash them away in the request context for later handlers to use.
// Also adds header to indicate which host is currently servicing the request
func NewContextHandler(delegate http.Handler) http.Handler {
	h := func(w http.ResponseWriter, r *http.Request) {
		ctx := &requestContext{
			identity:  extractRoleFromRequest(r),
			requestID: extractCorrelationID(r),
			hostname:  extractClientHost(r),
			ipaddr:    xhttp.ClientIPFromRequest(r),
		}
		c := r.Context()
		c = context.WithValue(c, keyContext, ctx)

		// Set XHostname on the response
		w.Header().Set(header.XHostname, nodeInfo.HostName())
		w.Header().Set(header.XCorrelationID, ctx.requestID)

		delegate.ServeHTTP(w, r.WithContext(c))
	}
	return http.HandlerFunc(h)
}

// WithTestContext is used in unit tests to set HTTP context
func WithTestContext(r *http.Request, identity ClientIdentity, requestID string) *http.Request {
	ctx := &requestContext{
		identity:  identity,
		requestID: requestID,
		hostname:  nodeInfo.HostName(),
		ipaddr:    nodeInfo.LocalIP(),
	}
	c := r.Context()
	c = context.WithValue(c, keyContext, ctx)

	return r.WithContext(c)
}

// New returns a new Context with the supplied identity & request identifiers
// typically you should be using ForRequest or Context to get a context
// this primarily exists for tests
func New(role, commonName, requestID, host, ip string) Context {
	if requestID == "" {
		requestID = guid.MustCreate()
	}
	if host == "" {
		host = nodeInfo.HostName()
	}
	if ip == "" {
		ip = nodeInfo.LocalIP()
	}
	return &requestContext{
		identity:  NewIdentity(role, commonName),
		requestID: requestID,
		hostname:  host,
		ipaddr:    ip,
		accepts:   header.ApplicationJSON,
		headers:   map[string]string{},
	}
}

func (c *requestContext) WithRequestID(requestID string) Context {
	copy := c.copy()
	copy.requestID = requestID
	return copy
}

func (c *requestContext) WithAccepts(accepts string) Context {
	copy := c.copy()
	copy.accepts = accepts
	return copy
}

func (c *requestContext) WithHeaders(headers map[string]string) Context {
	copy := c.copy()
	for k, v := range headers {
		copy.headers[k] = v
	}
	return copy
}

func (c *requestContext) copy() *requestContext {
	headers := map[string]string{}
	if c.headers != nil {
		for k, v := range c.headers {
			headers[k] = v
		}
	}
	return &requestContext{
		accepts:   c.accepts,
		identity:  c.identity,
		requestID: c.requestID,
		hostname:  c.hostname,
		ipaddr:    c.ipaddr,
		headers:   headers,
	}
}

func (c *requestContext) Identity() ClientIdentity {
	return c.identity
}

func (c *requestContext) RequestID() string {
	return c.requestID
}

func (c *requestContext) Host() string {
	return c.hostname
}

func (c *requestContext) IP() string {
	return c.ipaddr
}

func (c *requestContext) Accepts() string {
	if c.accepts == "" {
		return header.ApplicationJSON
	}
	return c.accepts
}

func (c *requestContext) Headers() map[string]string {
	return c.headers
}

func (c *requestContext) IsPlainText() bool {
	return c.accepts == header.TextPlain
}

// SetHeaders updates the supplied request with details from the current context
func (c *requestContext) SetHeaders(r *http.Request) {
	if c != nil {
		r.Header.Set(header.Accept, c.Accepts())

		if c.identity != nil {
			r.Header.Set(header.XIdentity, c.Identity().String())
		}

		if c.requestID != "" {
			r.Header.Set(header.XCorrelationID, c.requestID)
		}

		r.Header.Set(header.XHostname, c.Host())

		if c.headers != nil {
			for k, v := range c.headers {
				r.Header.Set(k, v)
			}
		}
	}
}

// ClientIdentity contains information about the identity of an API caller
type ClientIdentity interface {
	String() string
	Role() string
	CommonName() string
}

// NewIdentity returns a new ClientIdentity instance with the indicated role & CommonName
func NewIdentity(role string, commonName string) ClientIdentity {
	return clientIdentity{role: role, cn: commonName}
}

// NewIdentityFromCert returns a new ClientIdentity instance from client's Certificate
func NewIdentityFromCert(c *x509.Certificate) ClientIdentity {
	if c == nil {
		return clientIdentity{}
	}
	return clientIdentity{
		cn:   c.Subject.CommonName,
		role: roleExtractor(&c.Subject),
	}
}

type clientIdentity struct {
	// cn, this is the commonName from the client cert
	cn string
	// role, this is the roleName extracted from the client cert OU
	role string
}

// CommonName returns the clients commonName [typically hostname]
func (c clientIdentity) CommonName() string {
	return c.cn
}

// Role returns the clients role if we were able to determine one [from the OU]
func (c clientIdentity) Role() string {
	return c.role
}

// String returns the identity as a single string value [e.g. for use with a HSM identity]
// in the format of role/cn
func (c clientIdentity) String() string {
	if c.role != "" {
		return c.role + "/" + c.cn
	}
	return "unknown"
}

// extractRoleFromRequest determines the requests role from the request details
// currently we map this from the client cert CommonName, but this may change
func extractRoleFromRequest(r *http.Request) ClientIdentity {
	if r.TLS == nil {
		return clientIdentity{}
	}
	pc := r.TLS.PeerCertificates
	if len(pc) == 0 {
		return clientIdentity{}
	}
	return clientIdentity{
		cn:   pc[0].Subject.CommonName,
		role: roleExtractor(&pc[0].Subject),
	}
}

// ExtractRoleName will parse out from the supplied Name the clients roleName
func extractRoleName(n *pkix.Name) string {
	return n.CommonName
}

// extractCorrelationID will find or create a requestID for this http request.
func extractCorrelationID(req *http.Request) string {
	corID := req.Header.Get(header.XCorrelationID)
	if corID == "" {
		corID = guid.MustCreate()
	}
	return corID
}

// extractClientHost extracts the client Host from req, if present.
func extractClientHost(req *http.Request) string {
	// TODO: check headers "X-Forwarded-For"
	host := req.Header.Get(header.XHostname)
	if host == "" {
		host = req.Host
	}
	return host
}
