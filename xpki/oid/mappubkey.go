package oid

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"

	"github.com/juju/errors"
)

// PublicKeyAlgorithmInfo provides OID info for Public Key algorithms
type PublicKeyAlgorithmInfo struct {
	name         string
	oid          asn1.ObjectIdentifier
	oidstr       string
	registration string
	publey       x509.PublicKeyAlgorithm
}

// Algorithm returns x509.PublicKeyAlgorithm
func (h PublicKeyAlgorithmInfo) Algorithm() x509.PublicKeyAlgorithm {
	return h.publey
}

// Name is friendly name of the OID: SHA1, etc
func (h PublicKeyAlgorithmInfo) Name() string {
	return h.name
}

// OID is ASN1 ObjectIdentifier
func (h PublicKeyAlgorithmInfo) OID() asn1.ObjectIdentifier {
	return h.oid
}

// Registration returns official registration info in
// "{iso(1) identified-organization(3) oiw(14) secsig(3) algorithm(2) 26}" format
func (h PublicKeyAlgorithmInfo) Registration() string {
	return h.registration
}

// String returns string representation of OID: "1.2.840.113549.1"
func (h PublicKeyAlgorithmInfo) String() string {
	if h.oidstr == "" {
		h.oidstr = h.oid.String()
	}
	return h.oidstr
}

// Type specifies OID algorithm type for PubKey
func (h PublicKeyAlgorithmInfo) Type() AlgType {
	return AlgPubKey
}

//
// Public Key Algorithms
//

// RSA specifies RFC 3279, 2.3 Public Key Algorithm info
var RSA = PublicKeyAlgorithmInfo{
	name:         x509.RSA.String(),
	publey:       x509.RSA,
	oid:          SignatureAlgorithmRSA,
	registration: "{iso(1) member-body(2) us(840) rsadsi(113549) pkcs(1) 1}",
}

// ECDSA specifies RFC 3279, Algorithms and Identifiers for the Internet X.509 Public Key Infrastructure
var ECDSA = PublicKeyAlgorithmInfo{
	name:         x509.ECDSA.String(),
	publey:       x509.ECDSA,
	oid:          SignatureAlgorithmECDSA,
	registration: "{iso(1) member-body(2) us(840) ansi-X9-62(10045) keyType(2) 1}",
}

// PublicKeyAlgorithmToSignatureAlgorithm maps certificate public key
// algorithms to CMS signature algorithms.
var PublicKeyAlgorithmToSignatureAlgorithm = map[x509.PublicKeyAlgorithm]pkix.AlgorithmIdentifier{
	x509.RSA:   {Algorithm: SignatureAlgorithmRSA},
	x509.ECDSA: {Algorithm: SignatureAlgorithmECDSA},
}

// PublicKeyAlgorithmByOID returns an algorithm by OID
func PublicKeyAlgorithmByOID(oid string) (*PublicKeyAlgorithmInfo, error) {
	item := LookupByOID(oid)
	algo, ok := item.(PublicKeyAlgorithmInfo)
	if !ok {
		return nil, errors.NotFoundf(algNotFoundFmt, oid)
	}
	return &algo, nil
}

// PublicKeyAlgorithmByName returns an algorithm by name
func PublicKeyAlgorithmByName(name string) (*PublicKeyAlgorithmInfo, error) {
	item := LookupByName(name)
	algo, ok := item.(*PublicKeyAlgorithmInfo)
	if !ok {
		return nil, errors.NotFoundf(algNotFoundFmt, name)
	}
	return algo, nil
}
