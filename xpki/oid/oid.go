package oid

import (
	"encoding/asn1"
	"strconv"
	"strings"

	"github.com/juju/errors"
)

const algNotFoundFmt = "algorithm %s"

// AlgType specifies OID algorithm type
type AlgType int

const (
	// AlgUnknown specifies unknow algorithm
	AlgUnknown = iota
	// AlgHash specifies hash
	AlgHash
	// AlgPubKey specifies public key
	AlgPubKey
	// AlgSig specifies signature
	AlgSig
)

// Info provides basic OID info: friendly name, OID and registration string
type Info interface {
	// Name is friendly name of the OID: SHA1, etc
	Name() string
	// Type returns AlgType
	Type() AlgType
	// OID is ASN1 ObjectIdentifier
	OID() asn1.ObjectIdentifier
	// Registration returns official registration info in
	// "{iso(1) identified-organization(3) oiw(14) secsig(3) algorithm(2) 26}" format
	Registration() string
	// String returns string representation of OID: "1.2.840.113549.1"
	String() string
}

// Content type OIDs
var (
	Data       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	SignedData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	TSTInfo    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
)

// Attribute OIDs
var (
	AttributeContentType    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	AttributeMessageDigest  = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	AttributeSigningTime    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}
	AttributeTimeStampToken = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 14}
)

// Signature Algorithm  OIDs
var (
	SignatureAlgorithmRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	SignatureAlgorithmECDSA = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
)

// Digest Algorithm OIDs
var (
	DigestAlgorithmSHA1     = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	DigestAlgorithmMD5      = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 5}
	DigestAlgorithmSHA256   = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	DigestAlgorithmSHA384   = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
	DigestAlgorithmSHA512   = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}
	DigestAlgorithmSHA3x224 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 7}
)

// X509 extensions
var (
	SubjectKeyIdentifier = asn1.ObjectIdentifier{2, 5, 29, 14}
)

// NewObjectIdentifier creates an object identifier from it's string representation.
// Supports ASN.1 notation and dot notation. OID-IRI notation is not supported.
func NewObjectIdentifier(oid string) (oi asn1.ObjectIdentifier, err error) {
	if len(oid) == 0 {
		return nil, errors.Errorf("zero length OBJECT IDENTIFIER")
	}

	if oid[0] == '{' {
		// ASN.1 notation. (eg {iso(1) member-body(2) us(840) rsadsi(113549) pkcs(1) pkcs-9(9) messageDigest(4)})
		parts := strings.Split(oid[1:len(oid)-1], " ")
		oi = make(asn1.ObjectIdentifier, len(parts), len(parts))
		for i, part := range parts {
			idx := strings.IndexRune(part, '(')
			oi[i], err = strconv.Atoi(part[idx+1 : len(part)-1])
			if err != nil {
				return
			}
		}
	} else {
		// Dot notation. (eg 1.2.840.113549.1.9.4)
		parts := strings.Split(oid, ".")
		oi = make(asn1.ObjectIdentifier, len(parts), len(parts))
		for i, part := range parts {
			oi[i], err = strconv.Atoi(part)
			if err != nil {
				return
			}
		}
	}
	return oi, nil
}
