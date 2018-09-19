package oid

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/asn1"

	"github.com/juju/errors"
)

// HashAlgorithmInfo provides OID info for Hash algorithms
type HashAlgorithmInfo struct {
	name         string
	oid          asn1.ObjectIdentifier
	oidstr       string
	registration string
	hash         crypto.Hash
}

// HashFunc allows HashAlgorithmInfo to satisfry the
// crypto.SignerOpts interface for signing digests.
// You can use a cryptoid.HashAlgorithm directly when
// using a crypto.Signer interface to sign digests.
func (h HashAlgorithmInfo) HashFunc() crypto.Hash {
	return h.hash
}

// Name is friendly name of the OID: SHA1, etc
func (h HashAlgorithmInfo) Name() string {
	return h.name
}

// OID is ASN1 ObjectIdentifier
func (h HashAlgorithmInfo) OID() asn1.ObjectIdentifier {
	return h.oid
}

// Registration returns official registration info in
// "{iso(1) identified-organization(3) oiw(14) secsig(3) algorithm(2) 26}" format
func (h HashAlgorithmInfo) Registration() string {
	return h.registration
}

// String returns string representation of OID: "1.2.840.113549.1"
func (h HashAlgorithmInfo) String() string {
	if h.oidstr == "" {
		h.oidstr = h.oid.String()
	}
	return h.oidstr
}

// Type specifies OID algorithm type for Hash
func (h HashAlgorithmInfo) Type() AlgType {
	return AlgHash
}

//
// Hash Algorithms
//

// SHA1 described in RFC 3370, Cryptographic Message Syntax (CMS) Algorithms
var SHA1 = HashAlgorithmInfo{
	name:         "SHA1",
	oid:          DigestAlgorithmSHA1,
	oidstr:       DigestAlgorithmSHA1.String(),
	registration: "{iso(1) identified-organization(3) oiw(14) secsig(3) algorithm(2) 26}",
	hash:         crypto.SHA1,
}

// SHA256 described in RFC 3560, Use of the RSAES-OAEP Key Transport Algorithm in the Cryptographic Message Syntax (CMS)
var SHA256 = HashAlgorithmInfo{
	name:         "SHA256",
	oid:          DigestAlgorithmSHA256,
	oidstr:       DigestAlgorithmSHA256.String(),
	registration: "{joint-iso-itu-t(2) country(16) us(840) organization(1) gov(101) csor(3) nistalgorithm(4) hashalgs(2) 1}",
	hash:         crypto.SHA256,
}

// SHA384 described in RFC 3560, Use of the RSAES-OAEP Key Transport Algorithm in the Cryptographic Message Syntax (CMS)
var SHA384 = HashAlgorithmInfo{
	name:         "SHA384",
	oid:          DigestAlgorithmSHA384,
	oidstr:       DigestAlgorithmSHA384.String(),
	registration: "{joint-iso-itu-t(2) country(16) us(840) organization(1) gov(101) csor(3) nistalgorithm(4) hashalgs(2) 2}",
	hash:         crypto.SHA384,
}

// SHA512 described in RFC 3560, Use of the RSAES-OAEP Key Transport Algorithm in the Cryptographic Message Syntax (CMS)
var SHA512 = HashAlgorithmInfo{
	name:         "SHA512",
	oid:          DigestAlgorithmSHA512,
	oidstr:       DigestAlgorithmSHA512.String(),
	registration: "{joint-iso-itu-t(2) country(16) us(840) organization(1) gov(101) csor(3) nistalgorithm(4) hashalgs(2) 3}",
	hash:         crypto.SHA512,
}

// SHA3x224 described in RFC for SHA-3 is pending
var SHA3x224 = HashAlgorithmInfo{
	name:         "SHA3-224",
	oid:          DigestAlgorithmSHA3x224,
	oidstr:       DigestAlgorithmSHA3x224.String(),
	registration: "{joint-iso-itu-t(2) country(16) us(840) organization(1) gov(101) csor(3) nistalgorithm(4) hashalgs(2) 7}",
	hash:         crypto.SHA3_224,
}

// SHA3x256 described in RFC for SHA-3 is pending
var SHA3x256 = HashAlgorithmInfo{
	name:         "SHA3-256",
	oid:          asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 8},
	registration: "{joint-iso-itu-t(2) country(16) us(840) organization(1) gov(101) csor(3) nistalgorithm(4) hashalgs(2) 8}",
	hash:         crypto.SHA3_256,
}

// SHA3x384 described in RFC for SHA-3 is pending
var SHA3x384 = HashAlgorithmInfo{
	name:         "SHA3-384",
	oid:          asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 9},
	registration: "{joint-iso-itu-t(2) country(16) us(840) organization(1) gov(101) csor(3) nistalgorithm(4) hashalgs(2) 9}",
	hash:         crypto.SHA3_384,
}

// SHA3x512 described in RFC for SHA-3 is pending
var SHA3x512 = HashAlgorithmInfo{
	name:         "SHA3-512",
	oid:          asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 10},
	registration: "{joint-iso-itu-t(2) country(16) us(840) organization(1) gov(101) csor(3) nistalgorithm(4) hashalgs(2) 10}",
	hash:         crypto.SHA3_512,
}

// SHAKE128 described in RFC for SHA-3 is pending
var SHAKE128 = HashAlgorithmInfo{
	name:         "SHAKE128",
	oid:          asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 11},
	registration: "{joint-iso-itu-t(2) country(16) us(840) organization(1) gov(101) csor(3) nistalgorithm(4) hashalgs(2) 11}",
}

// SHAKE256 described in RFC for SHA-3 is pending
var SHAKE256 = HashAlgorithmInfo{
	name:         "SHAKE256",
	oid:          asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 12},
	registration: "{joint-iso-itu-t(2) country(16) us(840) organization(1) gov(101) csor(3) nistalgorithm(4) hashalgs(2) 12}",
}

// DigestAlgorithmToHash maps digest OIDs to crypto.Hash values.
var DigestAlgorithmToHash = map[string]crypto.Hash{
	DigestAlgorithmSHA1.String():   crypto.SHA1,
	DigestAlgorithmMD5.String():    crypto.MD5,
	DigestAlgorithmSHA256.String(): crypto.SHA256,
	DigestAlgorithmSHA384.String(): crypto.SHA384,
	DigestAlgorithmSHA512.String(): crypto.SHA512,
}

// HashToDigestAlgorithm maps crypto.Hash values to digest OIDs.
var HashToDigestAlgorithm = map[crypto.Hash]asn1.ObjectIdentifier{
	crypto.SHA1:   DigestAlgorithmSHA1,
	crypto.MD5:    DigestAlgorithmMD5,
	crypto.SHA256: DigestAlgorithmSHA256,
	crypto.SHA384: DigestAlgorithmSHA384,
	crypto.SHA512: DigestAlgorithmSHA512,
}

// HashAlgorithmByOID returns an algorithm by OID
func HashAlgorithmByOID(oid string) (*HashAlgorithmInfo, error) {
	item := LookupByOID(oid)
	algo, ok := item.(HashAlgorithmInfo)
	if !ok {
		return nil, errors.NotFoundf(algNotFoundFmt, oid)
	}
	return &algo, nil
}

// HashAlgorithmForPublicKey returns a suitable hash algorithm for public key
func HashAlgorithmForPublicKey(pub crypto.PublicKey) *HashAlgorithmInfo {
	if ecPub, ok := pub.(*ecdsa.PublicKey); ok {
		switch ecPub.Curve {
		case elliptic.P256():
			return &SHA256
		case elliptic.P384():
			return &SHA384
		case elliptic.P521():
			return &SHA512
		}
	} else if rsaPub, ok := pub.(*rsa.PublicKey); ok {
		size := rsaPub.N.BitLen()
		if size >= 4096 {
			return &SHA512
		}
		if size > 2048 {
			return &SHA384
		}
		if size > 1024 {
			return &SHA256
		}
		return &SHA1
	}

	if ecPriv, ok := pub.(*ecdsa.PrivateKey); ok {
		return HashAlgorithmForPublicKey(&ecPriv.PublicKey)
	} else if rsaPriv, ok := pub.(*rsa.PrivateKey); ok {
		return HashAlgorithmForPublicKey(&rsaPriv.PublicKey)
	}

	return &SHA256
}

// HashAlgorithmByName returns an algorithm by name
func HashAlgorithmByName(name string) (*HashAlgorithmInfo, error) {
	item := LookupByName(name)
	algo, ok := item.(*HashAlgorithmInfo)
	if !ok {
		return nil, errors.NotFoundf(algNotFoundFmt, name)
	}
	return algo, nil
}

// HashAlgorithmByCrypto returns an algorithm by crypto identifier
func HashAlgorithmByCrypto(hash crypto.Hash) *HashAlgorithmInfo {
	switch hash {
	case crypto.SHA1:
		return &SHA1
	case crypto.SHA256:
		return &SHA256
	case crypto.SHA384:
		return &SHA384
	case crypto.SHA512:
		return &SHA512
	case crypto.SHA3_224:
		return &SHA3x224
	case crypto.SHA3_256:
		return &SHA3x256
	case crypto.SHA3_384:
		return &SHA3x384
	case crypto.SHA3_512:
		return &SHA3x512
	default:
		panic("Invalid crypto.Hash") // This shouldn't be possible
	}
}
