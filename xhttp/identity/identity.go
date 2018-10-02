package identity

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"

	"github.com/go-phorce/dolly/xhttp"
)

// ExtractRole will parse out from the supplied Name the clients roleName
type ExtractRole func(*pkix.Name) string

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

// NewIdentityFromCert returns a new Identity instance from client's Certificate
func NewIdentityFromCert(c *x509.Certificate) Identity {
	if c == nil {
		return identity{}
	}
	return identity{
		name: c.Subject.CommonName,
		role: roleExtractor(&c.Subject),
	}
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
	if c.role != "" {
		return c.role + "/" + c.name
	}
	return "unknown"
}

func extractIdentityFromRequest(r *http.Request) Identity {
	if r.TLS == nil {
		return identity{
			name: xhttp.ClientIPFromRequest(r),
			role: "guest",
		}
	}
	pc := r.TLS.PeerCertificates
	if len(pc) == 0 {
		return identity{}
	}
	return identity{
		name: pc[0].Subject.CommonName,
		role: roleExtractor(&pc[0].Subject),
	}
}

func extractRoleFromPKIX(n *pkix.Name) string {
	return n.CommonName
}
