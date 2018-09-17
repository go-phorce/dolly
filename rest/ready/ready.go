package ready

import (
	"net/http"

	"github.com/go-phorce/pkg/xhttp/httperror"
	"github.com/go-phorce/pkg/xhttp/marshal"
)

var (
	errUnavailable = httperror.New(http.StatusServiceUnavailable, "Not ready",
		"The service is not ready yet. Try again later.")
)

// ServiceStatus specifies an interface to check if the service is ready to serve requests
type ServiceStatus interface {
	IsReady() bool
}

// ServiceReadyVerifier is a http.Handler that checks if the service is ready to serve,
// and if so, chain the Delegate handler, otherwise call's the Error handler
type ServiceReadyVerifier struct {
	Status   ServiceStatus
	Delegate http.Handler
	Error    http.Handler
}

// ServeHTTP implements the http.Handler interface
func (c *ServiceReadyVerifier) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.Status.IsReady() {
		c.Delegate.ServeHTTP(w, r)
	} else {
		c.Error.ServeHTTP(w, r)
	}
}

// NewServiceStatusVerifier is a http.Handler that checks if the service is ready to serve,
// and if so, chain the Delegate handler, otherwise call's the Error handler
// it returns an error
func NewServiceStatusVerifier(s ServiceStatus, delegate http.Handler) http.Handler {
	unavailable := func(w http.ResponseWriter, r *http.Request) {
		marshal.WriteJSON(w, r, errUnavailable)
	}
	v := ServiceReadyVerifier{
		Status:   s,
		Delegate: delegate,
		Error:    http.HandlerFunc(unavailable),
	}
	return &v
}
