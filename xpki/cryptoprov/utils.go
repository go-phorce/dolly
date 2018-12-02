package cryptoprov

import (
	"crypto"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/go-phorce/dolly/xpki/gpg"
	"github.com/juju/errors"
	"golang.org/x/crypto/openpgp/packet"
)

// LoadGPGPrivateKey returns GPG private key.
// The input key can be in PEM encoded format, or PKCS11 URI.
func (c *Crypto) LoadGPGPrivateKey(creationTime time.Time, key []byte) (*packet.PrivateKey, error) {
	var pk *packet.PrivateKey
	var err error

	keyPem := string(key)
	if strings.HasPrefix(keyPem, "pkcs11") {
		pkuri, err := ParsePrivateKeyURI(keyPem)
		if err != nil {
			return nil, errors.Trace(err)
		}

		provider, err := c.ByManufacturer(pkuri.Manufacturer())
		if err != nil {
			return nil, errors.Trace(err)
		}

		s, err := provider.GetCryptoSigner(pkuri.ID())
		if err != nil {
			return nil, errors.Trace(err)
		}

		pk, err = gpg.ConvertToPacketPrivateKey(creationTime, s)
		if err != nil {
			return nil, errors.Trace(err)
		}

	} else {
		pk, err = gpg.ConvertPemToPgpPrivateKey(creationTime, key)
		if err != nil {
			return nil, errors.Annotatef(err, "convert PEM key to PGP format: %v", key)
		}
	}
	return pk, nil
}

// LoadSigner returns crypto.Signer.
// The input key can be in PEM encoded format, or PKCS11 URI.
func (c *Crypto) LoadSigner(key []byte) (crypto.Signer, error) {
	var s crypto.Signer
	var err error
	keyPem := string(key)
	if strings.HasPrefix(keyPem, "pkcs11") {
		pkuri, err := ParsePrivateKeyURI(keyPem)
		if err != nil {
			return nil, errors.Trace(err)
		}

		provider, err := c.ByManufacturer(pkuri.Manufacturer())
		if err != nil {
			return nil, errors.Annotate(err, "api=CreateCryptoSignerFromPEM, reason=ByManufacturer")
		}

		s, err = provider.GetCryptoSigner(pkuri.ID())
		if err != nil {
			return nil, errors.Trace(err)
		}
	} else {
		s, err = helpers.ParsePrivateKeyPEM(key)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return s, nil
}
