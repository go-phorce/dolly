package certutil

import (
	"crypto/x509"
	"encoding/hex"
)

// GetThumbprintStr returns hex-encoded SHA1 of the certificate
func GetThumbprintStr(c *x509.Certificate) (string, error) {
	return SHA1Hex(c.Raw)
}

// GetSubjectKeyID returns Subject Key Identifier
func GetSubjectKeyID(c *x509.Certificate) string {
	return hex.EncodeToString(c.SubjectKeyId)
}

// GetAuthorityKeyID returns Authority Key Identifier
func GetAuthorityKeyID(c *x509.Certificate) string {
	return hex.EncodeToString(c.AuthorityKeyId)
}

// GetSubjectID returns ID of the cert.
// If present, it uses Subject Key Identifier,
// otherwise SHA1 of the Subject name
func GetSubjectID(c *x509.Certificate) string {
	if len(c.SubjectKeyId) > 0 {
		return hex.EncodeToString(c.SubjectKeyId)
	}
	s, _ := SHA1Hex(c.RawSubject)
	return s
}

// GetIssuerID returns ID of the issuer.
// If present, it uses Authority Key Identifier,
// otherwise SHA1 of the Issuer name
func GetIssuerID(c *x509.Certificate) string {
	if len(c.AuthorityKeyId) > 0 {
		return hex.EncodeToString(c.AuthorityKeyId)
	}
	s, _ := SHA1Hex(c.RawIssuer)
	return s
}
