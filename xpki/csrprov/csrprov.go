package csrprov

import (
	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xpki", "csrprov")

// Provider extends cryptoprov.Crypto functionality to support CSP procesing
// and certificate signing
type Provider struct {
	provider cryptoprov.Provider
}

// New returns an instance of CSR provider
func New(provider cryptoprov.Provider) *Provider {
	return &Provider{
		provider: provider,
	}
}
