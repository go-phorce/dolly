package crypto11

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"fmt"
	"io"
	"strings"

	"github.com/juju/errors"
	pkcs11 "github.com/miekg/pkcs11"
)

// KeyPurpose declares the purpose for keys
type KeyPurpose int

const (
	// Undefined purpose of key
	Undefined KeyPurpose = 0
	// Signing specifies the purpose of key to be used in signing/verification operations
	Signing KeyPurpose = 1
	// Encryption specifies the purpose of key to be used in encryption/decryption operations
	Encryption KeyPurpose = 2
)

// Identify returns the ID and label for a PKCS#11 object.
//
// Either of these values may be used to retrieve the key for later use.
func (lib *PKCS11Lib) Identify(object *PKCS11Object) (keyID, label string, err error) {
	logger.Tracef("api=Identify, slot=0x%X, obj=0x%X", object.Slot, object.Handle)

	a := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_ID, nil),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, nil),
	}
	if err = lib.withSession(object.Slot, func(session pkcs11.SessionHandle) error {
		a, err = lib.Ctx.GetAttributeValue(session, object.Handle, a)
		return err
	}); err != nil {
		return "", "", errors.Trace(err)
	}
	return string(a[0].Value), string(a[1].Value), nil
}

// Find a key object.  For asymmetric keys this only finds one half so
// callers will call it twice.
func (lib *PKCS11Lib) findKey(session pkcs11.SessionHandle, keyID, label string, keyclass uint, keytype uint) (pkcs11.ObjectHandle, error) {
	var err error
	var handles []pkcs11.ObjectHandle
	template := []*pkcs11.Attribute{}
	if keyclass != ^uint(0) {
		template = append(template, pkcs11.NewAttribute(pkcs11.CKA_CLASS, keyclass))
	}
	if keytype != ^uint(0) {
		template = append(template, pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, keytype))
	}
	if keyID != "" {
		template = append(template, pkcs11.NewAttribute(pkcs11.CKA_ID, []byte(keyID)))
	}
	if label != "" {
		template = append(template, pkcs11.NewAttribute(pkcs11.CKA_LABEL, []byte(label)))
	}
	if err = lib.Ctx.FindObjectsInit(session, template); err != nil {
		return 0, errors.Trace(err)
	}
	defer lib.Ctx.FindObjectsFinal(session)
	if handles, _, err = lib.Ctx.FindObjects(session, 1); err != nil {
		return 0, errors.Trace(err)
	}
	if len(handles) == 0 {
		return 0, errors.Trace(errKeyNotFound)
	}
	return handles[0], nil
}

// ListKeys returns key objects on the slot matching the key class and type
func (lib *PKCS11Lib) ListKeys(session pkcs11.SessionHandle, keyclass uint, keytype uint) ([]pkcs11.ObjectHandle, error) {
	var err error
	var handles []pkcs11.ObjectHandle
	template := []*pkcs11.Attribute{}
	if keyclass != ^uint(0) {
		template = append(template, pkcs11.NewAttribute(pkcs11.CKA_CLASS, keyclass))
	}
	if keytype != ^uint(0) {
		template = append(template, pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, keytype))
	}
	if err = lib.Ctx.FindObjectsInit(session, template); err != nil {
		return nil, errors.Trace(err)
	}
	defer lib.Ctx.FindObjectsFinal(session)
	if handles, _, err = lib.Ctx.FindObjects(session, 100); err != nil {
		return nil, errors.Trace(err)
	}

	return handles, nil
}

// FindKeys returns key objects on the slot matching label and key type
func (lib *PKCS11Lib) FindKeys(session pkcs11.SessionHandle, keylabel string, keyclass uint, keytype uint) ([]pkcs11.ObjectHandle, error) {
	var err error
	var handles []pkcs11.ObjectHandle
	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, []byte(keylabel)),
	}
	if keyclass != ^uint(0) {
		template = append(template, pkcs11.NewAttribute(pkcs11.CKA_CLASS, keyclass))
	}
	if keytype != ^uint(0) {
		template = append(template, pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, keytype))
	}
	if err = lib.Ctx.FindObjectsInit(session, template); err != nil {
		return nil, errors.Trace(err)
	}
	defer lib.Ctx.FindObjectsFinal(session)
	if handles, _, err = lib.Ctx.FindObjects(session, 100); err != nil {
		return nil, errors.Trace(err)
	}

	return handles, nil
}

// FindKeyPair retrieves a previously created asymmetric key.
//
// Either (but not both) of id and label may be nil, in which case they are ignored.
func (lib *PKCS11Lib) FindKeyPair(keyID, label string) (crypto.PrivateKey, error) {
	return lib.FindKeyPairOnSlot(lib.Slot.id, keyID, label)
}

// FindKeyPairOnSlot retrieves a previously created asymmetric key, using a specified slot.
//
// Either (but not both) of id and label may be nil, in which case they are ignored.
func (lib *PKCS11Lib) FindKeyPairOnSlot(slot uint, keyID, label string) (crypto.PrivateKey, error) {
	logger.Tracef("api=FindKeyPairOnSlot, slot=0x%X", slot)
	var err error
	var k crypto.PrivateKey
	if err = lib.setupSessions(slot, 0); err != nil {
		return nil, errors.Trace(err)
	}
	err = lib.withSession(slot, func(session pkcs11.SessionHandle) error {
		k, err = lib.FindKeyPairOnSession(session, slot, keyID, label)
		return err
	})
	return k, err
}

// FindKeyPairOnSession retrieves a previously created asymmetric key, using a specified session.
//
// Either (but not both) of id and label may be nil, in which case they are ignored.
func (lib *PKCS11Lib) FindKeyPairOnSession(session pkcs11.SessionHandle, slot uint, keyID, label string) (crypto.PrivateKey, error) {
	logger.Tracef("api=FindKeyPairOnSession, slot=0x%X, session=0x%X", slot, session)
	var err error
	var privHandle, pubHandle pkcs11.ObjectHandle
	var pub crypto.PublicKey

	if privHandle, err = lib.findKey(session, keyID, label, pkcs11.CKO_PRIVATE_KEY, ^uint(0)); err != nil {
		return nil, errors.Trace(err)
	}
	attributes := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, 0),
	}
	if attributes, err = lib.Ctx.GetAttributeValue(session, privHandle, attributes); err != nil {
		return nil, errors.Trace(err)
	}
	keyType := BytesToUlong(attributes[0].Value)
	if pubHandle, err = lib.findKey(session, keyID, label, pkcs11.CKO_PUBLIC_KEY, keyType); err != nil {
		return nil, errors.Trace(err)
	}
	switch keyType {
	case pkcs11.CKK_DSA:
		if pub, err = lib.exportDSAPublicKey(session, pubHandle); err != nil {
			return nil, errors.Annotate(err, "exportDSAPublicKey")
		}
		return &PKCS11PrivateKeyDSA{key: &PKCS11PrivateKey{PKCS11Object{privHandle, slot}, pub}, lib: lib}, nil
	case pkcs11.CKK_RSA:
		if pub, err = lib.exportRSAPublicKey(session, pubHandle); err != nil {
			return nil, errors.Annotate(err, "exportRSAPublicKey")
		}
		return &PKCS11PrivateKeyRSA{key: &PKCS11PrivateKey{PKCS11Object{privHandle, slot}, pub}, lib: lib}, nil
	case pkcs11.CKK_ECDSA:
		if pub, err = lib.exportECDSAPublicKey(session, pubHandle); err != nil {
			return nil, errors.Annotate(err, "exportECDSAPublicKey")
		}
		return &PKCS11PrivateKeyECDSA{key: &PKCS11PrivateKey{PKCS11Object{privHandle, slot}, pub}, lib: lib}, nil
	default:
		return nil, errors.Annotatef(errUnsupportedKeyType, "key type: %v", keyType)
	}
}

// ConvertToPublic converts a private key interface to crypto.PublicKey type
func ConvertToPublic(priv crypto.PrivateKey) (crypto.PublicKey, error) {
	switch t := priv.(type) {
	case *rsa.PrivateKey:
		return t.Public(), nil
	case *ecdsa.PrivateKey:
		return t.Public(), nil
	case *PKCS11PrivateKeyRSA:
		return t.Public(), nil
	case *PKCS11PrivateKeyECDSA:
		return t.Public(), nil
	case *PKCS11PrivateKeyDSA:
		return t.Public(), nil
	}
	return nil, errors.Trace(errUnsupportedKeyType)
}

// GetKey returns private key handle
func (lib *PKCS11Lib) GetKey(keyID string) (crypto.PrivateKey, error) {
	key, err := lib.FindKeyPair(keyID, "")
	if err != nil {
		return nil, errors.Annotatef(err, "unable to find key %q", keyID)
	}

	return key, err
}

// ExportKey returns PKCS#11 URI for specified key ID.
// It does not return key bytes.
func (lib *PKCS11Lib) ExportKey(keyID string) (string, []byte, error) {
	hi, err := lib.Ctx.GetInfo()
	if err != nil {
		return "", nil, errors.Annotate(err, "module info")
	}

	// ensure that key exists
	_, err = lib.FindKeyPair(keyID, "")
	if err != nil {
		return "", nil, errors.Annotatef(err, "unable to find key %q", keyID)
	}

	ti, err := lib.Ctx.GetTokenInfo(lib.Slot.id)
	if err != nil {
		return "", nil, errors.Annotate(err, "token info")
	}

	var uri string
	uri = fmt.Sprintf("pkcs11:manufacturer=%s;model=%s;serial=%s;token=%s;id=%s;type=private",
		strings.TrimSpace(strings.TrimRight(hi.ManufacturerID, "\x00")),
		strings.TrimSpace(ti.Model),
		strings.TrimSpace(ti.SerialNumber),
		strings.TrimSpace(ti.Label),
		strings.TrimSpace(keyID),
	)

	return uri, nil, nil
}

// KeyIdentifier interface provides key ID and label
type KeyIdentifier interface {
	KeyID() string
	Label() string
}

// PrivateKeyGen contains a reference to a loaded PKCS#11 private key object.
type privateKeyGen struct {
	id    string
	label string
	*PKCS11PrivateKey
	crypto.PrivateKey
}

func (p *privateKeyGen) KeyID() string {
	return p.id
}

func (p *privateKeyGen) Label() string {
	return p.label
}

func (p *privateKeyGen) Public() crypto.PublicKey {
	return p.PKCS11PrivateKey.PubKey
}

func (p *privateKeyGen) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if signer, ok := p.PrivateKey.(crypto.Signer); ok {
		b, err := signer.Sign(rand, digest, opts)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return b, nil
	}

	return nil, errors.Trace(errUnsupportedKeyType)
}

// Decrypt decrypts ciphertext with priv.
// If opts is nil or of type *PKCS1v15DecryptOptions then PKCS#1 v1.5 decryption is performed.
// Otherwise opts must have type *OAEPOptions and OAEP decryption is done.
func (p *privateKeyGen) Decrypt(rand io.Reader, ciphertext []byte, opts crypto.DecrypterOpts) (plaintext []byte, err error) {
	if decrypter, ok := p.PrivateKey.(crypto.Decrypter); ok {
		b, err := decrypter.Decrypt(rand, ciphertext, opts)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return b, nil
	}

	return nil, errors.Trace(errUnsupportedKeyType)
}

// GenerateRSAKey generates RSA key pair
func (lib *PKCS11Lib) GenerateRSAKey(label string, bits int, purpose int) (crypto.PrivateKey, error) {
	keypurpose := KeyPurpose(purpose)
	priv, err := lib.GenerateRSAKeyPairWithLabel(label, bits, keypurpose)
	if err != nil {
		return nil, errors.Trace(err)
	}

	id, l, err := lib.Identify(&priv.key.PKCS11Object)
	if err != nil {
		return nil, errors.Trace(err)
	}

	k := &privateKeyGen{
		id:               string(id),
		label:            string(l),
		PKCS11PrivateKey: priv.key,
		PrivateKey:       priv,
	}

	return k, nil
}

// GenerateECDSAKey generates ECDSA key pair
func (lib *PKCS11Lib) GenerateECDSAKey(label string, curve elliptic.Curve) (crypto.PrivateKey, error) {
	priv, err := lib.GenerateECDSAKeyPairWithLabel(label, curve)
	if err != nil {
		return nil, errors.Trace(err)
	}
	id, l, err := lib.Identify(&priv.key.PKCS11Object)
	if err != nil {
		return nil, errors.Trace(err)
	}

	k := &privateKeyGen{
		id:               string(id),
		label:            string(l),
		PKCS11PrivateKey: priv.key,
		PrivateKey:       priv,
	}

	return k, nil
}

// IdentifyKey returns the ID and label for a private key.
func (lib *PKCS11Lib) IdentifyKey(priv crypto.PrivateKey) (keyID, label string, err error) {
	if ki, ok := priv.(KeyIdentifier); ok {
		return ki.KeyID(), ki.Label(), nil
	}

	var p11o *PKCS11Object
	switch t := priv.(type) {
	case *PKCS11PrivateKeyRSA:
		p11o = &t.key.PKCS11Object
	case *PKCS11PrivateKeyECDSA:
		p11o = &t.key.PKCS11Object
	case *PKCS11PrivateKeyDSA:
		p11o = &t.key.PKCS11Object
	default:
		return "", "", errors.Trace(errUnsupportedKeyType)
	}

	id, l, err := lib.Identify(p11o)
	if err != nil {
		return "", "", errors.Trace(err)
	}
	keyID = string(id)
	label = string(l)
	return
}
