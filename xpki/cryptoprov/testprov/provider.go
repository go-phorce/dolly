package testprov

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"strings"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/pkg/errors"
)

// inMemProv stores keyID to signer mapping in memory. Private keys are not exportable.
type inMemProv struct {
	keyIDToPvk map[string]crypto.PrivateKey
}

// registerKey registers key for the given id in HSM
func (h *inMemProv) registerKey(keyID string, pvk crypto.PrivateKey) {
	h.keyIDToPvk[keyID] = pvk
}

// getSigner returns signer for the given key id in HSM
func (h *inMemProv) getKey(keyID string) (crypto.PrivateKey, error) {
	pvk, ok := h.keyIDToPvk[keyID]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return pvk, nil
}

type provImpl struct {
	id    string
	label string
	pvk   crypto.PrivateKey
}

// KeyID returns key id of the signer
func (s *provImpl) KeyID() string {
	return s.id
}

// Label returns key label of the signer
func (s *provImpl) Label() string {
	return s.label
}

// Public returns public key of the signer
func (s *provImpl) Public() crypto.PublicKey {
	if signer, ok := s.pvk.(crypto.Signer); ok {
		return signer.Public()
	} else if decrypter, ok := s.pvk.(crypto.Decrypter); ok {
		return decrypter.Public()
	}
	return s.pvk
}

// Sign signs data
func (s *provImpl) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if opts == nil {
		opts = crypto.SHA256
	}
	if signer, ok := s.pvk.(crypto.Signer); ok {
		return signer.Sign(rand, digest, opts)
	}

	return nil, errors.Errorf("crypto.Signer is not supported")
}

// Decrypt data
func (s *provImpl) Decrypt(rand io.Reader, ciphertext []byte, opts crypto.DecrypterOpts) (plaintext []byte, err error) {
	if decrypter, ok := s.pvk.(crypto.Decrypter); ok {
		return decrypter.Decrypt(rand, ciphertext, opts)
	}

	return nil, errors.Errorf("crypto.Decrypter is not supported")
}

type rsaKeyGenerator interface {
	GenerateKey(random io.Reader, bits int) (*rsa.PrivateKey, error)
}

type defaultRsaKeyGenerator struct {
}

func (g *defaultRsaKeyGenerator) GenerateKey(random io.Reader, bits int) (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(random, bits)
}

type ecdsaKeyGenerator interface {
	GenerateKey(c elliptic.Curve, rand io.Reader) (*ecdsa.PrivateKey, error)
}

type defaultEcdsaKeyGenerator struct {
}

func (g *defaultEcdsaKeyGenerator) GenerateKey(c elliptic.Curve, rand io.Reader) (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(c, rand)
}

type idGenerator interface {
	Generate() string
}

type defaultIDGenerator struct {
}

func (g *defaultIDGenerator) Generate() string {
	return guid.MustCreate()
}

// Provider defines an interface to work with crypto providers
type Provider struct {
	idGenerator
	rsaKeyGenerator
	ecdsaKeyGenerator
	inMemProv *inMemProv
}

// Init creates new provider for in memory based HSM
func Init() (*Provider, error) {
	inMemProv := inMemProv{
		keyIDToPvk: make(map[string]crypto.PrivateKey),
	}

	return &Provider{
		inMemProv:         &inMemProv,
		rsaKeyGenerator:   &defaultRsaKeyGenerator{},
		ecdsaKeyGenerator: &defaultEcdsaKeyGenerator{},
		idGenerator:       &defaultIDGenerator{},
	}, nil
}

// Manufacturer return manufacturer for the provider
func (p *Provider) Manufacturer() string {
	return "testprov"
}

// Model return model for the provider
func (p *Provider) Model() string {
	return "inmem"
}

// Serial return serial number for the provider
func (p *Provider) Serial() string {
	return "20764350726"
}

// GetKey returns key for the given id
func (p *Provider) GetKey(keyID string) (crypto.PrivateKey, error) {
	pvk, err := p.inMemProv.getKey(keyID)
	if err != nil {
		return nil, errors.WithMessagef(err, "GetKey(%s)", keyID)
	}
	return pvk, nil
}

// GenerateRSAKey creates signer using randomly generated RSA key
func (p *Provider) GenerateRSAKey(label string, bits int, purpose int) (crypto.PrivateKey, error) {
	reader := rand.Reader
	key, err := p.rsaKeyGenerator.GenerateKey(reader, bits)
	if err != nil {
		return nil, errors.WithMessagef(err, "bitSize=%d", bits)
	}

	if len(label) == 0 {
		label = fmt.Sprintf("%x", guid.MustCreate())
	}

	id := p.idGenerator.Generate()

	si := &provImpl{
		id:    id,
		label: label,
		pvk:   key,
	}
	p.inMemProv.registerKey(id, si)
	return si, nil

}

// GenerateECDSAKey creates signer using randomly generated ECDSA key
func (p *Provider) GenerateECDSAKey(label string, curve elliptic.Curve) (crypto.PrivateKey, error) {
	reader := rand.Reader
	key, err := p.ecdsaKeyGenerator.GenerateKey(curve, reader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(label) == 0 {
		label = fmt.Sprintf("%x", guid.MustCreate())
	}

	id := p.idGenerator.Generate()

	si := &provImpl{
		id:    id,
		label: label,
		pvk:   key,
	}
	p.inMemProv.registerKey(id, si)
	return si, nil
}

// IdentifyKey returns key id and label for the given private key
func (p *Provider) IdentifyKey(priv crypto.PrivateKey) (keyID, label string, err error) {
	if ki, ok := priv.(*provImpl); ok {
		return ki.KeyID(), ki.Label(), nil
	}
	return "", "", errors.Errorf("unsupported key: %T", priv)
}

// ExportKey returns pkcs11 uri for the given key id
func (p *Provider) ExportKey(keyID string) (string, []byte, error) {
	s, err := p.inMemProv.getKey(keyID)
	if err != nil {
		return "", nil, errors.WithMessagef(err, "keyID=%s", keyID)
	}

	si, ok := s.(*provImpl)
	if !ok {
		return "", nil, errors.New("unsupported signer")
	}

	var uri string
	uri = fmt.Sprintf("pkcs11:manufacturer=%s;model=%s;serial=%s;token=%s;id=%s;type=private",
		strings.TrimSpace(strings.TrimRight(p.Manufacturer(), "\x00")),
		strings.TrimSpace(p.Model()),
		strings.TrimSpace(p.Serial()),
		strings.TrimSpace(si.Label()),
		strings.TrimSpace(keyID),
	)

	return uri, nil, nil
}
