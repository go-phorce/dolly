package util

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"time"

	"github.com/go-phorce/dolly/metrics"
)

var (
	keyForCertExpiry = []string{"cert", "expiry", "days"}
	keyForCrlExpiry  = []string{"crl", "expiry", "days"}
)

// PublishShortLivedCertExpirationInDays publish cert expiration time in Days for short lived certificates
func PublishShortLivedCertExpirationInDays(c *x509.Certificate, typ string) float32 {
	expiresIn := c.NotAfter.Sub(time.Now().UTC())
	expiresInDays := float32(expiresIn) / float32(time.Hour*24)
	metrics.SetGauge(
		keyForCertExpiry,
		expiresInDays,
		metrics.Tag{Name: "CN", Value: c.Subject.CommonName},
		metrics.Tag{Name: "type", Value: typ},
	)
	return expiresInDays
}

// PublishCertExpirationInDays publish cert expiration time in Days
func PublishCertExpirationInDays(c *x509.Certificate, typ string) float32 {
	expiresIn := c.NotAfter.Sub(time.Now().UTC())
	expiresInDays := float32(expiresIn) / float32(time.Hour*24)
	metrics.SetGauge(
		keyForCertExpiry,
		expiresInDays,
		metrics.Tag{Name: "CN", Value: c.Subject.CommonName},
		metrics.Tag{Name: "type", Value: typ},
		metrics.Tag{Name: "Serial", Value: c.SerialNumber.String()},
		metrics.Tag{Name: "SKI", Value: hex.EncodeToString(c.SubjectKeyId)},
	)
	return expiresInDays
}

// PublishCRLExpirationInDays publish CRL expiration time in Days
func PublishCRLExpirationInDays(c *pkix.CertificateList, issuer *x509.Certificate) float32 {
	PublishCertExpirationInDays(issuer, "issuer")

	expiresIn := c.TBSCertList.NextUpdate.Sub(time.Now().UTC())
	expiresInDays := float32(expiresIn) / float32(time.Hour*24)
	metrics.SetGauge(
		keyForCrlExpiry,
		expiresInDays,
		metrics.Tag{Name: "CN", Value: issuer.Subject.CommonName},
		metrics.Tag{Name: "Serial", Value: issuer.SerialNumber.String()},
		metrics.Tag{Name: "SKI", Value: hex.EncodeToString(issuer.SubjectKeyId)},
	)
	return expiresInDays
}
