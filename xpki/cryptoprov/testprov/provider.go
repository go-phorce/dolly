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
	"github.com/juju/errors"
)

// inMemProv stores keyID to signer mapping in memory. Private keys are not exportable.
type inMemProv struct {
	keyIDToSigner map[string]crypto.Signer
}

// registerSigner registers signer for the given key id in HSM
func (h *inMemProv) registerSigner(keyID string, signer crypto.Signer) {
	h.keyIDToSigner[keyID] = signer
}

// getSigner returns signer for the given key id in HSM
func (h *inMemProv) getSigner(keyID string) (crypto.Signer, error) {
	signer, ok := h.keyIDToSigner[keyID]
	if !ok {
		return nil, fmt.Errorf("signer not found: %s", keyID)
	}
	return signer, nil
}

type signerImpl struct {
	id     string
	label  string
	signer crypto.Signer
}

// KeyID returns key id of the signer
func (s *signerImpl) KeyID() string {
	return s.id
}

// Label returns key label of the signer
func (s *signerImpl) Label() string {
	return s.label
}

// Public returns public key of the signer
func (s *signerImpl) Public() crypto.PublicKey {
	return s.signer.Public()
}

// Sign signs data
func (s *signerImpl) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	if opts == nil {
		opts = crypto.SHA256
	}
	return s.signer.Sign(rand, digest, opts)
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
		keyIDToSigner: make(map[string]crypto.Signer),
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
	return "test"
}

// Model return model for the provider
func (p *Provider) Model() string {
	return "inmem"
}

// Serial return serial number for the provider
func (p *Provider) Serial() string {
	return "20764350726"
}

// GetCryptoSigner returns signer for the given key id
func (p *Provider) GetCryptoSigner(keyID string) (crypto.Signer, error) {
	signer, err := p.inMemProv.getSigner(keyID)
	if err != nil {
		return nil, errors.Annotatef(err, "api=GetCryptoSigner, reason=GetSigner, keyId=%s", keyID)
	}
	return signer, nil
}

// GenerateRSAKey creates signer using randomly generated RSA key
func (p *Provider) GenerateRSAKey(label string, bits int, purpose int) (crypto.PrivateKey, error) {
	reader := rand.Reader
	key, err := p.rsaKeyGenerator.GenerateKey(reader, bits)
	if err != nil {
		return nil, errors.Annotatef(err, "api=GenerateRSAKey, bitSize=%d", bits)
	}

	if len(label) == 0 {
		label = fmt.Sprintf("%x", guid.MustCreate())
	}

	id := p.idGenerator.Generate()

	si := &signerImpl{
		id:     id,
		label:  label,
		signer: key,
	}
	p.inMemProv.registerSigner(id, si)
	return si, nil

}

// GenerateECDSAKey creates signer using randomly generated ECDSA key
func (p *Provider) GenerateECDSAKey(label string, curve elliptic.Curve) (crypto.PrivateKey, error) {
	reader := rand.Reader
	key, err := p.ecdsaKeyGenerator.GenerateKey(curve, reader)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(label) == 0 {
		label = fmt.Sprintf("%x", guid.MustCreate())
	}

	id := p.idGenerator.Generate()

	si := &signerImpl{
		id:     id,
		label:  label,
		signer: key,
	}
	p.inMemProv.registerSigner(id, si)
	return si, nil
}

// IdentifyKey returns key id and label for the given private key
func (p *Provider) IdentifyKey(priv crypto.PrivateKey) (keyID, label string, err error) {
	if ki, ok := priv.(*signerImpl); ok {
		return ki.KeyID(), ki.Label(), nil
	}
	return "", "", errors.New("unsupported key")
}

// GetKey returns private key handle
func (p *Provider) GetKey(keyID string) (crypto.PrivateKey, error) {
	s, err := p.inMemProv.getSigner(keyID)
	if err != nil {
		return nil, errors.Annotatef(err, "api=GetKey, reason=getSigner, keyID=%s", keyID)
	}

	pvk, ok := s.(crypto.PrivateKey)
	if !ok {
		return nil, errors.Errorf("no private key %q", keyID)
	}

	return pvk, nil
}

// ExportKey returns pkcs11 uri for the given key id
func (p *Provider) ExportKey(keyID string) (string, []byte, error) {
	s, err := p.inMemProv.getSigner(keyID)
	if err != nil {
		return "", nil, errors.Annotatef(err, "api=ExportKey, reason=getSigner, keyID=%s", keyID)
	}

	si, ok := s.(*signerImpl)
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
