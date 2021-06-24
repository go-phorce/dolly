package csr

import (
	"crypto"
	"crypto/elliptic"
	"crypto/x509"
	"strings"

	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/juju/errors"
)

const (
	// CurveP256 specifies curve P-256 for ESDCA
	CurveP256 = 256

	// CurveP384 specifies curve P-384 for ESDCA
	CurveP384 = 384

	// CurveP521 specifies curve P-521 for ESDCA
	CurveP521 = 521
)

// KeyPurpose declares the purpose for keys
type KeyPurpose int

const (
	// Undefined purpose of key
	Undefined KeyPurpose = 0
	// SigningKey specifies the purpose of key to be used in signing/verification operations
	SigningKey KeyPurpose = 1
	// EncryptionKey specifies the purpose of key to be used in encryption/decryption operations
	EncryptionKey KeyPurpose = 2
)

// KeyRequest contains the algorithm and key size for a new private key.
type KeyRequest interface {
	Algo() string
	Label() string
	Size() int
	Generate() (crypto.PrivateKey, error)
	SigAlgo() x509.SignatureAlgorithm
	Purpose() int
}

// keyRequest contains the algorithm and key size for a new private key.
type keyRequest struct {
	L    string     `json:"label"`
	A    string     `json:"algo"`
	S    int        `json:"size"`
	P    KeyPurpose `json:"purpose"`
	prov cryptoprov.Provider
}

// Label returns the requested key label.
func (kr *keyRequest) Label() string {
	return kr.L
}

// Algo returns the requested key algorithm represented as a string.
func (kr *keyRequest) Algo() string {
	return kr.A
}

// Size returns the requested key size.
func (kr *keyRequest) Size() int {
	return kr.S
}

// Purpose returns the purpose of the key .
func (kr *keyRequest) Purpose() int {
	return int(kr.P)
}

// SigAlgo returns an appropriate X.509 signature algorithm given the
// key request's type and size.
func (kr *keyRequest) SigAlgo() x509.SignatureAlgorithm {
	return SigAlgo(kr.Algo(), kr.Size())
}

// Generate generates a key as specified in the request. Currently,
// only ECDSA and RSA are supported.
func (kr *keyRequest) Generate() (crypto.PrivateKey, error) {
	switch algo := kr.Algo(); strings.ToUpper(algo) {
	case "RSA":
		err := validateRSAKeyPairInfoHandler(kr.Size())
		if err != nil {
			return nil, errors.Annotate(err, "validate RSA key")
		}
		pk, err := kr.prov.GenerateRSAKey(kr.Label(), kr.Size(), kr.Purpose())
		if err != nil {
			return nil, errors.Annotate(err, "generate RSA key")
		}
		return pk, nil
	case "ECDSA":
		err := validateECDSAKeyPairCurveInfoHandler(kr.Size())
		if err != nil {
			return nil, errors.Annotate(err, "validate ECDSA key")
		}

		var curve elliptic.Curve
		switch kr.Size() {
		case CurveP256:
			curve = elliptic.P256()
		case CurveP384:
			curve = elliptic.P384()
		case CurveP521:
			curve = elliptic.P521()
		default:
			return nil, errors.New("invalid curve")
		}
		pk, err := kr.prov.GenerateECDSAKey(kr.Label(), curve)
		if err != nil {
			return nil, errors.Annotate(err, "generate ECDSA key")
		}
		return pk, nil
	default:
		return nil, errors.Errorf("invalid algorithm: %s", algo)
	}
}

// NewKeyRequest returns KeyRequest from given parameters
func (c *Provider) NewKeyRequest(label, algo string, keySize int, purpose KeyPurpose) KeyRequest {
	return &keyRequest{
		L:    label,
		A:    algo,
		S:    keySize,
		P:    purpose,
		prov: c.provider,
	}
}

// NewKeyRequest returns KeyRequest from given parameters
func NewKeyRequest(prov cryptoprov.Provider, label, algo string, keySize int, purpose KeyPurpose) KeyRequest {
	return &keyRequest{
		L:    label,
		A:    algo,
		S:    keySize,
		P:    purpose,
		prov: prov,
	}
}

// validateRSAKeyPairInfoHandler validates size of the RSA key
func validateRSAKeyPairInfoHandler(size int) error {
	if size < 2048 {
		return errors.Errorf("RSA key is too weak: %d", size)
	}
	if size > 4096 {
		return errors.Errorf("RSA key size too large: %d", size)
	}

	return nil
}

// validateECDSAKeyPairCurveInfoHandler validates size of the ECDSA key
func validateECDSAKeyPairCurveInfoHandler(size int) error {
	switch size {
	case CurveP256, CurveP384, CurveP521:
		return nil
	}
	return errors.Errorf("invalid curve size: %d", size)
}

// SigAlgo returns signature algorithm for the given algorithm name and key size
// TODO: use oid pkg
func SigAlgo(algo string, size int) x509.SignatureAlgorithm {
	switch strings.ToUpper(algo) {
	case "RSA":
		switch {
		case size >= 4096:
			return x509.SHA512WithRSA
		case size >= 3072:
			return x509.SHA384WithRSA
		case size >= 2048:
			return x509.SHA256WithRSA
		default:
			return x509.SHA1WithRSA
		}
	case "ECDSA":
		switch size {
		case CurveP521:
			return x509.ECDSAWithSHA512
		case CurveP384:
			return x509.ECDSAWithSHA384
		case CurveP256:
			return x509.ECDSAWithSHA256
		default:
			return x509.ECDSAWithSHA1
		}
	default:
		return x509.UnknownSignatureAlgorithm
	}
}
