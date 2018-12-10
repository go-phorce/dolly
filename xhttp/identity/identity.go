package identity

import (
	"context"
	"net/http"
)

// Identity contains information about the identity of an API caller
type Identity interface {
	String() string
	Role() string
	Name() string
}

// Mapper returns Identity from supplied HTTP request
type Mapper func(*http.Request) Identity

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

// default mapper always returns "guest" for the role
func defaultIdentityMapper(r *http.Request) Identity {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return identity{
			name: ClientIPFromRequest(r),
			role: "guest",
		}
	}
	pc := r.TLS.PeerCertificates
	return identity{
		name: pc[0].Subject.CommonName,
		role: "guest",
	}
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
