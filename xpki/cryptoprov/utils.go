package cryptoprov

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
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
func (c *Crypto) LoadSigner(key []byte) (Provider, crypto.Signer, error) {
	var s crypto.Signer
	var err error
	var provider Provider

	keyPem := string(key)
	if strings.HasPrefix(keyPem, "pkcs11") {
		pkuri, err := ParsePrivateKeyURI(keyPem)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}

		provider, err = c.ByManufacturer(pkuri.Manufacturer())
		if err != nil {
			return nil, nil, errors.Annotate(err, "api=CreateCryptoSignerFromPEM, reason=ByManufacturer")
		}

		s, err = provider.GetCryptoSigner(pkuri.ID())
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
	} else {
		s, err = helpers.ParsePrivateKeyPEM(key)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
	}

	return provider, s, nil
}

// LoadTLSKeyPair reads and parses a public/private key pair from a pair
// of files. The files must contain PEM encoded data. The certificate file
// may contain intermediate certificates following the leaf certificate to
// form a certificate chain. On successful return, Certificate.Leaf will
// be nil because the parsed form of the certificate is not retained.
func (c *Crypto) LoadTLSKeyPair(certFile, keyFile string) (*tls.Certificate, error) {
	certPEMBlock, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	keyPEMBlock, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	return c.TLSKeyPair(certPEMBlock, keyPEMBlock)
}

// TLSKeyPair parses a public/private key pair from a pair of
// PEM encoded data. On successful return, Certificate.Leaf will be nil because
// the parsed form of the certificate is not retained.
func (c *Crypto) TLSKeyPair(certPEMBlock, keyPEMBlock []byte) (*tls.Certificate, error) {
	var err error
	var skippedBlockTypes []string

	cert := &tls.Certificate{}

	for {
		var certDERBlock *pem.Block
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		} else {
			skippedBlockTypes = append(skippedBlockTypes, certDERBlock.Type)
		}
	}

	if len(cert.Certificate) == 0 {
		if len(skippedBlockTypes) == 0 {
			return nil, errors.New("tls: failed to find any PEM data in certificate input")
		}
		if len(skippedBlockTypes) == 1 && strings.HasSuffix(skippedBlockTypes[0], "PRIVATE KEY") {
			return nil, errors.New("tls: failed to find certificate PEM data in certificate input, but did find a private key; PEM inputs may have been switched")
		}
		return nil, errors.Errorf("tls: failed to find \"CERTIFICATE\" PEM block in certificate input after skipping PEM blocks of the following types: %v", skippedBlockTypes)
	}

	// We don't need to parse the public key for TLS, but we so do anyway
	// to check that it looks sane and matches the private key.
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.Trace(err)
	}

	_, signer, err := c.LoadSigner(keyPEMBlock)
	if err != nil {
		return nil, errors.Trace(err)
	}

	cert.PrivateKey = signer

	return cert, nil
}
