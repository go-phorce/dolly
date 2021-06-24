package authority

import (
	"crypto"
	"io/ioutil"
	"strings"

	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/juju/errors"
)

// TODO: move to cryptoprov.Crypto

// NewSignerFromFromFile generates a new signer from a caFile
// and a caKey file, both PEM encoded or caKey contains PKCS#11 Uri
func NewSignerFromFromFile(crypto *cryptoprov.Crypto, caKeyFile string) (crypto.Signer, error) {
	cakey, err := ioutil.ReadFile(caKeyFile)
	if err != nil {
		return nil, errors.Annotatef(err, "load key file")
	}
	// remove trailing space and end-of-line
	cakey = []byte(strings.TrimSpace(string(cakey)))

	return NewSignerFromPEM(crypto, cakey)
}

// NewSignerFromPEM generates a new crypto signer from PEM encoded blocks,
// or caKey contains PKCS#11 Uri
func NewSignerFromPEM(prov *cryptoprov.Crypto, caKey []byte) (crypto.Signer, error) {
	_, pvk, err := prov.LoadPrivateKey(caKey)
	if err != nil {
		return nil, errors.Trace(err)
	}

	signer, supported := pvk.(crypto.Signer)
	if !supported {
		return nil, errors.Errorf("loaded key of %T type does not support crypto.Signer", pvk)
	}

	return signer, nil
}
