package authz

import (
	"net/http"

	"github.com/go-phorce/dolly/algorithms/slices"
	"github.com/go-phorce/dolly/xhttp/httperror"
	"github.com/go-phorce/dolly/xhttp/marshal"
)

var (
	errForbidden = httperror.New(http.StatusForbidden, "Access denied",
		"Provided client certificate does not grant access to this service.")
)

// ClientCertVerifier is a http.Handler that checks the client cert is one
// we allow access to grok, and if so, chain the Delegate handler, otherwise
// call's the Error handler
type ClientCertVerifier struct {
	Delegate           http.Handler
	Error              http.Handler
	ValidOrganizations []string
}

// ServeHTTP implements the http.Handler interface
func (c *ClientCertVerifier) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	peers := r.TLS.PeerCertificates
	if len(peers) == 0 ||
		len(peers[0].Subject.Organization) == 0 ||
		!slices.ContainsString(c.ValidOrganizations, peers[0].Subject.Organization[0]) {

		c.Error.ServeHTTP(w, r)
	} else {
		c.Delegate.ServeHTTP(w, r)
	}
}

// NewClientCertVerifier is a http.Handler that checks the client cert is one
// we allow access to the service, and if so, chain the delegate handler, otherwise
// it returns an error
func NewClientCertVerifier(validOrganizations []string, delegate http.Handler) http.Handler {
	forbidden := func(w http.ResponseWriter, r *http.Request) {
		marshal.WriteJSON(w, r, errForbidden)
	}
	v := ClientCertVerifier{
		Delegate:           delegate,
		Error:              http.HandlerFunc(forbidden),
		ValidOrganizations: validOrganizations,
	}
	return &v
}
