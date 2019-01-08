package cryptoprov

import (
	"crypto"
	"crypto/elliptic"
	"time"

	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xpki", "cryptoprov")

// ErrInvalidURI is returned if the PKCS #11 URI is invalid.
var ErrInvalidURI = errors.New("invalid URI")

// ErrInvalidPrivateKeyURI is returned if the PKCS #11 URI is invalid for the private key object
var ErrInvalidPrivateKeyURI = errors.New("invalid URI for private key object")

// KeyManager defines interface for key management operations
type KeyManager interface {
	CurrentSlotID() uint
	EnumTokens(currentSlotOnly bool, slotInfoFunc func(slotID uint, description, label, manufacturer, model, serial string) error) error
	EnumKeys(slotID uint, prefix string, keyInfoFunc func(id, label, typ, class, currentVersionID string, creationTime *time.Time) error) error
	DestroyKeyPairOnSlot(slotID uint, keyID string) error
	FindKeyPairOnSlot(slotID uint, keyID, label string) (crypto.PrivateKey, error)
	KeyInfo(slotID uint, keyID string, includePublic bool, keyInfoFunc func(id, label, typ, class, currentVersionID, pubKey string, creationTime *time.Time) error) error
}

// KeyGenerator defines interface for key generation operations
type KeyGenerator interface {
	// GenerateRSAKey returns RSA key for purpose: 1-signing, 2-encryption
	GenerateRSAKey(label string, bits int, purpose int) (crypto.PrivateKey, error)
	GenerateECDSAKey(label string, curve elliptic.Curve) (crypto.PrivateKey, error)
	IdentifyKey(crypto.PrivateKey) (keyID, label string, err error)
	ExportKey(keyID string) (string, []byte, error)
	GetKey(keyID string) (crypto.PrivateKey, error)
}

// Provider defines an interface to work with crypto providers: HSM, SoftHSM, KMS, crytpto
type Provider interface {
	KeyGenerator
	Manufacturer() string
	Model() string
}

// Crypto exposes instances of Provider
type Crypto struct {
	provider       Provider
	byManufacturer map[string]Provider
}

// New creates an instance of Crypto providers
func New(defaultProvider Provider, providers []Provider) (*Crypto, error) {
	c := &Crypto{
		provider:       defaultProvider,
		byManufacturer: map[string]Provider{},
	}

	if providers != nil {
		for _, p := range providers {
			if err := c.Add(p); err != nil {
				return nil, errors.Trace(err)
			}
		}
	}
	return c, nil
}

// Default returns a default crypto provider
func (c *Crypto) Default() Provider {
	return c.provider
}

// Add will add new provider
func (c *Crypto) Add(p Provider) error {
	m := p.Manufacturer()
	if _, ok := c.byManufacturer[m]; ok {
		return errors.Errorf("duplicate provider specified for manufacturer: %s", m)

	}
	c.byManufacturer[m] = p
	return nil
}

// ByManufacturer returns a provider by manufacturer
func (c *Crypto) ByManufacturer(manufacturer string) (Provider, error) {
	if c.provider != nil && c.provider.Manufacturer() == manufacturer {
		return c.provider, nil
	}

	p, ok := c.byManufacturer[manufacturer]
	if !ok {
		return nil, errors.NotFoundf("provider for manufacturer %s", manufacturer)
	}
	return p, nil
}
