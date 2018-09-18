package oid

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"

	"github.com/juju/errors"
)

// SignatureAlgorithmInfo provides OID info for Signature algorithms
type SignatureAlgorithmInfo struct {
	name               string
	oid                asn1.ObjectIdentifier
	oidstr             string
	registration       string
	X509               x509.SignatureAlgorithm
	PublicKeyAlgorithm *PublicKeyAlgorithmInfo
	HashAlgorithm      *HashAlgorithmInfo
}

// HashFunc allows SignatureAlgorithmInfo to satisfry the
// crypto.SignerOpts interface for signing digests.
// You can use a cryptoid.HashAlgorithm directly when
// using a crypto.Signer interface to sign digests.
func (h SignatureAlgorithmInfo) HashFunc() crypto.Hash {
	return h.HashAlgorithm.HashFunc()
}

// Name is friendly name of the OID: SHA1, etc
func (h SignatureAlgorithmInfo) Name() string {
	return h.name
}

// OID is ASN1 ObjectIdentifier
func (h SignatureAlgorithmInfo) OID() asn1.ObjectIdentifier {
	return h.oid
}

// Registration returns official registration info in
// "{iso(1) identified-organization(3) oiw(14) secsig(3) algorithm(2) 26}" format
func (h SignatureAlgorithmInfo) Registration() string {
	return h.registration
}

// String returns string representation of OID: "1.2.840.113549.1"
func (h SignatureAlgorithmInfo) String() string {
	if h.oidstr == "" {
		h.oidstr = h.oid.String()
	}
	return h.oidstr
}

// Type specifies OID algorithm type for Sig
func (h SignatureAlgorithmInfo) Type() AlgType {
	return AlgSig
}

//
// Signature Algorithms
//

// RSAWithSHA1 described in RFC 3279 2.2.1 RSA Signature Algorithms
var RSAWithSHA1 = SignatureAlgorithmInfo{
	name:               "RSA-SHA1",
	oid:                asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 5},
	registration:       "{iso(1) member-body(2) us(840) rsadsi(113549) pkcs(1) pkcs-1(1) sha1-with-rsa-signature(5)}",
	X509:               x509.SHA1WithRSA,
	PublicKeyAlgorithm: &RSA,
	HashAlgorithm:      &SHA1,
}

// RSAWithSHA256 described in RFC 4055 5 PKCS #1 Version 1.5
var RSAWithSHA256 = SignatureAlgorithmInfo{
	name:               "RSA-SHA256",
	oid:                asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11},
	registration:       "{iso(1) member-body(2) us(840) rsadsi(113549) pkcs(1) pkcs-1(1) RSAWithSHA256Encryption(11)}",
	X509:               x509.SHA256WithRSA,
	PublicKeyAlgorithm: &RSA,
	HashAlgorithm:      &SHA256,
}

// RSAWithSHA384 described in RFC 4055 5 PKCS #1 Version 1.5
var RSAWithSHA384 = SignatureAlgorithmInfo{
	name:               "RSA-SHA384",
	oid:                asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12},
	registration:       "{iso(1) member-body(2) us(840) rsadsi(113549) pkcs(1) pkcs-1(1) RSAWithSHA384Encryption(12)}",
	X509:               x509.SHA384WithRSA,
	PublicKeyAlgorithm: &RSA,
	HashAlgorithm:      &SHA384,
}

// RSAWithSHA512 described in RFC 4055 5 PKCS #1 Version 1.5
var RSAWithSHA512 = SignatureAlgorithmInfo{
	name:               "RSA-SHA512",
	oid:                asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13},
	registration:       "{iso(1) member-body(2) us(840) rsadsi(113549) pkcs(1) pkcs-1(1) RSAWithSHA384Encryption(13)}",
	X509:               x509.SHA512WithRSA,
	PublicKeyAlgorithm: &RSA,
	HashAlgorithm:      &SHA512,
}

// ECDSAWithSHA1 described in RFC 3279 2.2.3 ECDSA Signature Algorithm
var ECDSAWithSHA1 = SignatureAlgorithmInfo{
	name:               "ECDSA-SHA1",
	oid:                asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 1},
	registration:       "{iso(1) member-body(2) us(840) ansi-x962(10045) signatures(4) ecdsa-with-SHA1(1)}",
	X509:               x509.ECDSAWithSHA1,
	PublicKeyAlgorithm: &ECDSA,
	HashAlgorithm:      &SHA1,
}

// ECDSAWithSHA256 described in RFC 5758 3.2 ECDSA Signature Algorithm
var ECDSAWithSHA256 = SignatureAlgorithmInfo{
	name:               "ECDSA-SHA256",
	oid:                asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2},
	registration:       "{iso(1) member-body(2) us(840) ansi-X9-62(10045) signatures(4) ecdsa-with-SHA2(3) 2}",
	X509:               x509.ECDSAWithSHA256,
	PublicKeyAlgorithm: &ECDSA,
	HashAlgorithm:      &SHA256,
}

// ECDSAWithSHA384 described in RFC 5758 3.2 ECDSA Signature Algorithm
var ECDSAWithSHA384 = SignatureAlgorithmInfo{
	name:               "ECDSA-SHA384",
	oid:                asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3},
	registration:       "{iso(1) member-body(2) us(840) ansi-X9-62(10045) signatures(4) ecdsa-with-SHA2(3) 3}",
	X509:               x509.ECDSAWithSHA384,
	PublicKeyAlgorithm: &ECDSA,
	HashAlgorithm:      &SHA384,
}

// ECDSAWithSHA512 described in RFC 5758 3.2 ECDSA Signature Algorithm
var ECDSAWithSHA512 = SignatureAlgorithmInfo{
	name:               "ECDSA-SHA512",
	oid:                asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4},
	registration:       "{iso(1) member-body(2) us(840) ansi-X9-62(10045) signatures(4) ecdsa-with-SHA2(3) 4}",
	X509:               x509.ECDSAWithSHA512,
	PublicKeyAlgorithm: &ECDSA,
	HashAlgorithm:      &SHA512,
}

// SignatureAlgorithmToDigestAlgorithm maps x509.SignatureAlgorithm to
// digestAlgorithm OIDs.
var SignatureAlgorithmToDigestAlgorithm = map[x509.SignatureAlgorithm]asn1.ObjectIdentifier{
	x509.SHA1WithRSA:     DigestAlgorithmSHA1,
	x509.MD5WithRSA:      DigestAlgorithmMD5,
	x509.SHA256WithRSA:   DigestAlgorithmSHA256,
	x509.SHA384WithRSA:   DigestAlgorithmSHA384,
	x509.SHA512WithRSA:   DigestAlgorithmSHA512,
	x509.ECDSAWithSHA1:   DigestAlgorithmSHA1,
	x509.ECDSAWithSHA256: DigestAlgorithmSHA256,
	x509.ECDSAWithSHA384: DigestAlgorithmSHA384,
	x509.ECDSAWithSHA512: DigestAlgorithmSHA512,
}

// SignatureAlgorithmToSignatureAlgorithm maps x509.SignatureAlgorithm to
// signatureAlgorithm OIDs.
var SignatureAlgorithmToSignatureAlgorithm = map[x509.SignatureAlgorithm]asn1.ObjectIdentifier{
	x509.SHA1WithRSA:     SignatureAlgorithmRSA,
	x509.MD5WithRSA:      SignatureAlgorithmRSA,
	x509.SHA256WithRSA:   SignatureAlgorithmRSA,
	x509.SHA384WithRSA:   SignatureAlgorithmRSA,
	x509.SHA512WithRSA:   SignatureAlgorithmRSA,
	x509.ECDSAWithSHA1:   SignatureAlgorithmECDSA,
	x509.ECDSAWithSHA256: SignatureAlgorithmECDSA,
	x509.ECDSAWithSHA384: SignatureAlgorithmECDSA,
	x509.ECDSAWithSHA512: SignatureAlgorithmECDSA,
}

// SignatureAlgorithms maps digest and signature OIDs to
// x509.SignatureAlgorithm values.
var SignatureAlgorithms = map[string]map[string]x509.SignatureAlgorithm{
	SignatureAlgorithmRSA.String(): {
		DigestAlgorithmSHA1.String():   x509.SHA1WithRSA,
		DigestAlgorithmMD5.String():    x509.MD5WithRSA,
		DigestAlgorithmSHA256.String(): x509.SHA256WithRSA,
		DigestAlgorithmSHA384.String(): x509.SHA384WithRSA,
		DigestAlgorithmSHA512.String(): x509.SHA512WithRSA,
	},
	SignatureAlgorithmECDSA.String(): {
		DigestAlgorithmSHA1.String():   x509.ECDSAWithSHA1,
		DigestAlgorithmSHA256.String(): x509.ECDSAWithSHA256,
		DigestAlgorithmSHA384.String(): x509.ECDSAWithSHA384,
		DigestAlgorithmSHA512.String(): x509.ECDSAWithSHA512,
	},
}

// SignatureAlgorithmByOID returns an algorithm by OID
func SignatureAlgorithmByOID(oid string) (*SignatureAlgorithmInfo, error) {
	item := LookupByOID(oid)
	algo, ok := item.(SignatureAlgorithmInfo)
	if !ok {
		return nil, errors.NotFoundf(algNotFoundFmt, oid)
	}
	return &algo, nil
}

// SignatureAlgorithmByName returns an algorithm by name
func SignatureAlgorithmByName(name string) (SignatureAlgorithmInfo, error) {
	item := LookupByName(name)
	algo, ok := item.(SignatureAlgorithmInfo)
	if !ok {
		return SignatureAlgorithmInfo{}, errors.NotFoundf(algNotFoundFmt, name)
	}
	return algo, nil
}

// SignatureAlgorithmByKey returns an algorithm by key
func SignatureAlgorithmByKey(pkey interface{}) (*PublicKeyAlgorithmInfo, error) {
	key, ok := pkey.(crypto.Signer)
	if !ok {
		return nil, errors.NotSupportedf("crypto.Signer")
	}

	pub := key.Public()

	switch pub.(type) {
	case *rsa.PublicKey:
		return &RSA, nil
	case *ecdsa.PublicKey:
		return &ECDSA, nil
	}
	return nil, errors.NotSupportedf("crypto.Signer type %T", pub)
}

// SignatureAlgorithmByKeyAndHash returns an algorithm by key and Hash
func SignatureAlgorithmByKeyAndHash(pkey interface{}, hash crypto.Hash) (*SignatureAlgorithmInfo, error) {
	key, ok := pkey.(crypto.Signer)
	if !ok {
		return nil, errors.NotSupportedf("crypto.Signer")
	}

	pub := key.Public()

	switch pub.(type) {
	case *rsa.PublicKey:
		switch hash {
		case crypto.SHA1:
			return &RSAWithSHA1, nil
		case crypto.SHA256:
			return &RSAWithSHA256, nil
		case crypto.SHA384:
			return &RSAWithSHA384, nil
		case crypto.SHA512:
			return &RSAWithSHA512, nil
		default:
			return nil, errors.NotSupportedf("RSA")
		}
	case *ecdsa.PublicKey:
		switch hash {
		case crypto.SHA1:
			return &ECDSAWithSHA1, nil
		case crypto.SHA256:
			return &ECDSAWithSHA256, nil
		case crypto.SHA384:
			return &ECDSAWithSHA384, nil
		case crypto.SHA512:
			return &ECDSAWithSHA512, nil
		default:
			return nil, errors.NotSupportedf("ECDSA")
		}
	}
	return nil, errors.NotSupportedf("crypto.Signer type %T", pub)
}

// SignatureAlgorithmByX509 returns an algorithm by X509 identifier
func SignatureAlgorithmByX509(sig x509.SignatureAlgorithm) *SignatureAlgorithmInfo {
	switch sig {
	case x509.SHA1WithRSA:
		return &RSAWithSHA1
	case x509.SHA256WithRSA:
		return &RSAWithSHA256
	case x509.SHA384WithRSA:
		return &RSAWithSHA384
	case x509.SHA512WithRSA:
		return &RSAWithSHA512
	case x509.ECDSAWithSHA1:
		return &ECDSAWithSHA1
	case x509.ECDSAWithSHA256:
		return &ECDSAWithSHA256
	case x509.ECDSAWithSHA384:
		return &ECDSAWithSHA384
	case x509.ECDSAWithSHA512:
		return &ECDSAWithSHA512
	default:
		panic("Invalid x509.SignatureAlgorithm") // This shouldn't be possible
	}
}
