package crypto11

import (
	"crypto"
	"crypto/dsa"
	"io"
	"math/big"

	pkcs11 "github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

// PKCS11PrivateKeyDSA contains a reference to a loaded PKCS#11 DSA private key object.
type PKCS11PrivateKeyDSA struct {
	key *PKCS11PrivateKey
	lib *PKCS11Lib
}

// Export the public key corresponding to a private DSA key.
func (lib *PKCS11Lib) exportDSAPublicKey(session pkcs11.SessionHandle, pubHandle pkcs11.ObjectHandle) (crypto.PublicKey, error) {
	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_PRIME, nil),
		pkcs11.NewAttribute(pkcs11.CKA_SUBPRIME, nil),
		pkcs11.NewAttribute(pkcs11.CKA_BASE, nil),
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, nil),
	}
	exported, err := lib.Ctx.GetAttributeValue(session, pubHandle, template)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var p, q, g, x big.Int
	p.SetBytes(exported[0].Value)
	q.SetBytes(exported[1].Value)
	g.SetBytes(exported[2].Value)
	x.SetBytes(exported[3].Value)
	result := dsa.PublicKey{
		Parameters: dsa.Parameters{P: &p, Q: &q, G: &g},
		Y:          &x,
	}
	return &result, nil
}

// GenerateDSAKeyPair creates a DSA private key on the default slot
//
// The key will have a random label and ID.
func (lib *PKCS11Lib) GenerateDSAKeyPair(params *dsa.Parameters) (*PKCS11PrivateKeyDSA, error) {
	return lib.GenerateDSAKeyPairOnSlot(lib.Slot.id, nil, nil, params)
}

// GenerateDSAKeyPairOnSlot creates a DSA private key on a specified slot
//
// Either or both label and/or id can be nil, in which case a random values will be generated.
func (lib *PKCS11Lib) GenerateDSAKeyPairOnSlot(slot uint, id []byte, label []byte, params *dsa.Parameters) (*PKCS11PrivateKeyDSA, error) {
	var k *PKCS11PrivateKeyDSA
	var err error
	if err = lib.setupSessions(slot, 0); err != nil {
		return nil, errors.WithStack(err)
	}
	err = lib.withSession(slot, func(session pkcs11.SessionHandle) error {
		k, err = lib.GenerateDSAKeyPairOnSession(session, slot, id, label, params)
		return err
	})
	return k, err
}

// GenerateDSAKeyPairOnSession creates a DSA private key using a specified session
//
// Either or both label and/or id can be nil, in which case a random values will be generated.
func (lib *PKCS11Lib) GenerateDSAKeyPairOnSession(session pkcs11.SessionHandle, slot uint, id []byte, label []byte, params *dsa.Parameters) (*PKCS11PrivateKeyDSA, error) {
	var err error
	var pub crypto.PublicKey

	if label == nil {
		if label, err = lib.generateKeyLabel(); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	if id == nil {
		if id, err = lib.generateKeyID(); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	p := params.P.Bytes()
	q := params.Q.Bytes()
	g := params.G.Bytes()
	publicKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_DSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
		pkcs11.NewAttribute(pkcs11.CKA_PRIME, p),
		pkcs11.NewAttribute(pkcs11.CKA_SUBPRIME, q),
		pkcs11.NewAttribute(pkcs11.CKA_BASE, g),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
	}
	privateKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
		pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
		pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
	}
	mech := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_DSA_KEY_PAIR_GEN, nil)}
	pubHandle, privHandle, err := lib.Ctx.GenerateKeyPair(session,
		mech,
		publicKeyTemplate,
		privateKeyTemplate)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if pub, err = lib.exportDSAPublicKey(session, pubHandle); err != nil {
		return nil, errors.WithStack(err)
	}
	priv := PKCS11PrivateKeyDSA{
		key: &PKCS11PrivateKey{PKCS11Object{privHandle, slot}, pub},
		lib: lib,
	}
	return &priv, nil
}

// Sign signs a message using a DSA key.
//
// This completes the implemention of crypto.Signer for PKCS11PrivateKeyDSA.
//
// PKCS#11 expects to pick its own random data for signatures, so the rand argument is ignored.
//
// The return value is a DER-encoded byteblock.
func (priv *PKCS11PrivateKeyDSA) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	return priv.lib.dsaGeneric(priv.lib.Slot.id, priv.key.Handle, pkcs11.CKM_DSA, digest)
}

// Public returns the public half of a private key.
//
// This partially implements the go.crypto.Signer and go.crypto.Decrypter interfaces for
// PKCS11PrivateKey. (The remains of the implementation is in the
// key-specific types.)
func (priv *PKCS11PrivateKeyDSA) Public() crypto.PublicKey {
	return priv.key.PubKey
}
