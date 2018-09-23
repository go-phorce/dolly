package testca

import (
	"crypto"
	"crypto/x509"
)

// Entity is a certificate and private key.
type Entity struct {
	Issuer      *Entity
	PrivateKey  crypto.Signer
	Certificate *x509.Certificate
	NextSN      int64
}

// NewEntity creates a new CA.
func NewEntity(opts ...Option) *Entity {
	c := &configuration{}

	for _, opt := range opts {
		option(opt)(c)
	}

	return c.generate()
}

// Issue issues a new Entity with this one as its parent.
func (id *Entity) Issue(opts ...Option) *Entity {
	opts = append(opts, Issuer(id))
	return NewEntity(opts...)
}

// PFX wraps the certificate and private key in an encrypted PKCS#12 packet. The
// provided password must be alphanumeric.
func (id *Entity) PFX(password string) []byte {
	return toPFX(id.Certificate, id.PrivateKey, password)
}

// Chain builds a slice of *x509.Certificate from this CA and its issuers.
func (id *Entity) Chain() []*x509.Certificate {
	chain := []*x509.Certificate{}
	for this := id; this != nil; this = this.Issuer {
		chain = append(chain, this.Certificate)
	}

	return chain
}

// ChainPool builds an *x509.CertPool from this CA and its issuers.
func (id *Entity) ChainPool() *x509.CertPool {
	chain := x509.NewCertPool()
	for this := id; this != nil; this = this.Issuer {
		chain.AddCert(this.Certificate)
	}

	return chain
}

// IncrementSN returns the next serial number.
func (id *Entity) IncrementSN() int64 {
	defer func() {
		id.NextSN++
	}()

	return id.NextSN
}
