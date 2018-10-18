package xhttp

import (
	"net/http"
	"net/url"

	"github.com/go-phorce/dolly/xlog"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "xhttp")

// GetRelativeURL takes a path component of URL and constructs a new URL using
// the host and port from the request combined the provided path.
func GetRelativeURL(request *http.Request, endpoint string) url.URL {
	proto := "http"
	host := request.Host

	// If the request was received via TLS, use `https://` for the protocol
	if request.TLS != nil {
		proto = "https"
	}

	// Allow upstream proxies  to specify the forwarded protocol. Allow this value
	// to override our own guess.
	if specifiedProto := request.Header.Get("X-Forwarded-Proto"); specifiedProto != "" {
		proto = specifiedProto
	}

	// Default to "localhost" when no request.Host is provided.
	if request.Host == "" {
		host = "localhost"
	}

	return url.URL{Scheme: proto, Host: host, Path: endpoint}
}
