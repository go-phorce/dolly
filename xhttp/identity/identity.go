package identity

import (
	"context"
	"crypto/x509"
	"net/http"
)

// ExtractRole will parse out from the supplied Name the clients roleName
type ExtractRole func(*x509.Certificate) string

// Identity contains information about the identity of an API caller
type Identity interface {
	String() string
	Role() string
	Name() string
}

// NewIdentity returns a new Identity instance with the indicated role & CommonName
func NewIdentity(role string, name string) Identity {
	return identity{role: role, name: name}
}

type identity struct {
	// name of identity
	// It can be CommonName extracted from certificate
	name string
	// role of identity
	role string
}

// Name returns the clients name
func (c identity) Name() string {
	return c.name
}

// Role returns the clients role
func (c identity) Role() string {
	return c.role
}

// String returns the identity as a single string value
// in the format of role/name
func (c identity) String() string {
	if c.role != c.name {
		return c.role + "/" + c.name
	}
	return c.role
}

func extractIdentityFromRequest(r *http.Request) Identity {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return identity{
			name: ClientIPFromRequest(r),
			role: "guest",
		}
	}
	pc := r.TLS.PeerCertificates
	return identity{
		name: pc[0].Subject.CommonName,
		role: roleExtractor(pc[0]),
	}
}

// defaultExtractRole always returns "guest" as role name.
// Applications should initialize Role Mapper by calling identity.Initialize
func defaultExtractRole(_ *x509.Certificate) string {
	return "guest"
}

// WithTestIdentity is used in unit tests to set HTTP request identity
func WithTestIdentity(r *http.Request, identity Identity) *http.Request {
	ctx := &RequestContext{
		identity:      identity,
		correlationID: extractCorrelationID(r),
		clientIP:      nodeInfo.LocalIP(),
	}
	c := context.WithValue(r.Context(), keyContext, ctx)
	return r.WithContext(c)
}
