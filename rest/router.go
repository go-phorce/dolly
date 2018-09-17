package rest

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params httprouter.Params

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	for i := range ps {
		if ps[i].Key == name {
			return ps[i].Value
		}
	}
	return ""
}

// Handle is a function that can be registered to a route to handle HTTP
// requests. Like http.HandlerFunc, but has a third parameter for the values of
// wildcards (variables).
type Handle func(http.ResponseWriter, *http.Request, Params)

// Router provides a router interface
type Router interface {
	Handler() http.Handler
	GET(path string, handle Handle)
	HEAD(path string, handle Handle)
	OPTIONS(path string, handle Handle)
	POST(path string, handle Handle)
	PUT(path string, handle Handle)
	PATCH(path string, handle Handle)
	DELETE(path string, handle Handle)
}

type proxy struct {
	router *httprouter.Router
}

// NewRouter returns a new initialized Router.
func NewRouter(notfoundhandler http.HandlerFunc) Router {
	r := &proxy{
		router: httprouter.New(),
	}
	r.router.NotFound = notfoundhandler
	return r
}

func proxyHandle(handle Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		handle(w, r, Params(p))
	}
}

func (p *proxy) Handler() http.Handler {
	return p.router
}

// GET is a shortcut for router.Handle("GET", path, handle)
func (p *proxy) GET(path string, handle Handle) {
	p.router.Handle("GET", path, proxyHandle(handle))
}

// HEAD is a shortcut for router.Handle("HEAD", path, handle)
func (p *proxy) HEAD(path string, handle Handle) {
	p.router.Handle("HEAD", path, proxyHandle(handle))
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", path, handle)
func (p *proxy) OPTIONS(path string, handle Handle) {
	p.router.Handle("OPTIONS", path, proxyHandle(handle))
}

// POST is a shortcut for router.Handle("POST", path, handle)
func (p *proxy) POST(path string, handle Handle) {
	p.router.Handle("POST", path, proxyHandle(handle))
}

// PUT is a shortcut for router.Handle("PUT", path, handle)
func (p *proxy) PUT(path string, handle Handle) {
	p.router.Handle("PUT", path, proxyHandle(handle))
}

// PATCH is a shortcut for router.Handle("PATCH", path, handle)
func (p *proxy) PATCH(path string, handle Handle) {
	p.router.Handle("PATCH", path, proxyHandle(handle))
}

// DELETE is a shortcut for router.Handle("DELETE", path, handle)
func (p *proxy) DELETE(path string, handle Handle) {
	p.router.Handle("DELETE", path, proxyHandle(handle))
}
