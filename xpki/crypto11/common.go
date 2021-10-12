package crypto11

import (
	"C"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
	"unsafe"

	"github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

// AttributeNames maps PKCS11 atribute to string
var AttributeNames = map[uint]string{
	pkcs11.CKA_ID:       "ID",
	pkcs11.CKA_LABEL:    "Label",
	pkcs11.CKA_KEY_TYPE: "Key type",
	pkcs11.CKA_CLASS:    "Class",
}

// ObjectClassNames maps PKCS11 object class to string
var ObjectClassNames = map[uint]string{
	pkcs11.CKO_DATA:        "Data",
	pkcs11.CKO_CERTIFICATE: "Certificate",
	pkcs11.CKO_PUBLIC_KEY:  "Public key",
	pkcs11.CKO_PRIVATE_KEY: "Private key",
	pkcs11.CKO_SECRET_KEY:  "Secret key",
}

// KeyTypeNames maps PKCS11 key type to string
var KeyTypeNames = map[uint]string{
	pkcs11.CKK_RSA:   "RSA",
	pkcs11.CKK_DSA:   "DSA",
	pkcs11.CKK_DH:    "DH",
	pkcs11.CKK_ECDSA: "ECDSA",
}

// UlongToBytes converts Ulong to []byte
func UlongToBytes(n uint) []byte {
	return C.GoBytes(unsafe.Pointer(&n), C.sizeof_ulong) // ugh!
}

// BytesToUlong converts []byte to Ulong
func BytesToUlong(bs []byte) (n uint) {
	return *(*uint)(unsafe.Pointer(&bs[0])) // ugh
}

func concat(slices ...[]byte) []byte {
	n := 0
	for _, slice := range slices {
		n += len(slice)
	}
	r := make([]byte, n)
	n = 0
	for _, slice := range slices {
		n += copy(r[n:], slice)
	}
	return r
}

// Representation of a *DSA signature
type dsaSignature struct {
	R, S *big.Int
}

// Populate a dsaSignature from a raw byte sequence
func (sig *dsaSignature) unmarshalBytes(sigBytes []byte) error {
	if len(sigBytes) == 0 || len(sigBytes)%2 != 0 {
		return errMalformedSignature
	}
	n := len(sigBytes) / 2
	sig.R, sig.S = new(big.Int), new(big.Int)
	sig.R.SetBytes(sigBytes[:n])
	sig.S.SetBytes(sigBytes[n:])
	return nil
}

// Populate a dsaSignature from DER encoding
func (sig *dsaSignature) unmarshalDER(sigDER []byte) error {
	if rest, err := asn1.Unmarshal(sigDER, sig); err != nil {
		return err
	} else if len(rest) > 0 {
		return errMalformedDER
	}
	return nil
}

// Return the DER encoding of a dsaSignature
func (sig *dsaSignature) marshalDER() ([]byte, error) {
	return asn1.Marshal(*sig)
}

// Pick a random label for a key
func (lib *PKCS11Lib) generateKeyLabel() ([]byte, error) {
	const labelSize = 32
	rawLabel := make([]byte, labelSize)
	sz, err := lib.GenRandom(rawLabel)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if sz < len(rawLabel) {
		return nil, errors.WithStack(errCannotGetRandomData)
	}

	t := time.Now().UTC()
	label := fmt.Sprintf("%04d%02d%02d%02d%02d%02d_%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), hex.EncodeToString(rawLabel))
	return []byte(label[:32]), nil
}

// Pick a random ID for a key
func (lib *PKCS11Lib) generateKeyID() ([]byte, error) {
	const labelSize = 32
	rawLabel := make([]byte, labelSize)
	sz, err := lib.GenRandom(rawLabel)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if sz < len(rawLabel) {
		return nil, errors.WithStack(errCannotGetRandomData)
	}

	label := hex.EncodeToString(rawLabel)
	return []byte(label[:32]), nil
}

// Compute DSA/ECDSA signature and marshal the result in DER fform
func (lib *PKCS11Lib) dsaGeneric(slot uint, key pkcs11.ObjectHandle, mechanism uint, digest []byte) ([]byte, error) {
	var err error
	var sigBytes []byte
	var sig dsaSignature
	mech := []*pkcs11.Mechanism{pkcs11.NewMechanism(mechanism, nil)}
	err = lib.withSession(slot, func(session pkcs11.SessionHandle) error {
		if err = lib.Ctx.SignInit(session, mech, key); err != nil {
			return err
		}
		sigBytes, err = lib.Ctx.Sign(session, digest)
		return err
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	sig.unmarshalBytes(sigBytes)
	return sig.marshalDER()
}
