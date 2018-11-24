package metrics

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"time"
)

var (
	keyForCertExpiry = []string{"cert", "expiry", "days"}
	keyForCrlExpiry  = []string{"crl", "expiry", "days"}
)

// PublishShortLivedCertExpirationInDays publish cert expiration time in Days for short lived certificates
func PublishShortLivedCertExpirationInDays(c *x509.Certificate, typ string) float32 {
	expiresIn := c.NotAfter.Sub(time.Now().UTC())
	expiresInDays := float32(expiresIn) / float32(time.Hour*24)
	SetGauge(
		keyForCertExpiry,
		expiresInDays,
		Tag{"CN", c.Subject.CommonName},
		Tag{"type", typ},
	)
	return expiresInDays
}

// PublishCertExpirationInDays publish cert expiration time in Days
func PublishCertExpirationInDays(c *x509.Certificate, typ string) float32 {
	expiresIn := c.NotAfter.Sub(time.Now().UTC())
	expiresInDays := float32(expiresIn) / float32(time.Hour*24)
	SetGauge(
		keyForCertExpiry,
		expiresInDays,
		Tag{"CN", c.Subject.CommonName},
		Tag{"type", typ},
		Tag{"Serial", c.SerialNumber.String()},
		Tag{"SKI", hex.EncodeToString(c.SubjectKeyId)},
	)
	return expiresInDays
}

// PublishCRLExpirationInDays publish CRL expiration time in Days
func PublishCRLExpirationInDays(c *pkix.CertificateList, issuer *x509.Certificate) float32 {
	PublishCertExpirationInDays(issuer, "issuer")

	expiresIn := c.TBSCertList.NextUpdate.Sub(time.Now().UTC())
	expiresInDays := float32(expiresIn) / float32(time.Hour*24)
	SetGauge(
		keyForCrlExpiry,
		expiresInDays,
		Tag{"CN", issuer.Subject.CommonName},
		Tag{"Serial", issuer.SerialNumber.String()},
		Tag{"SKI", hex.EncodeToString(issuer.SubjectKeyId)},
	)
	return expiresInDays
}
