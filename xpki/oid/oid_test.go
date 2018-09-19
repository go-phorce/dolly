package oid

import (
	"encoding/asn1"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewObjectIdentifier(t *testing.T) {
	oids := []asn1.ObjectIdentifier{
		Data,
		SignedData,
		TSTInfo,
		AttributeContentType,
		AttributeMessageDigest,
		AttributeSigningTime,
		AttributeTimeStampToken,
		SignatureAlgorithmRSA,
		SignatureAlgorithmECDSA,
		DigestAlgorithmSHA1,
		DigestAlgorithmMD5,
		DigestAlgorithmSHA256,
		DigestAlgorithmSHA384,
		DigestAlgorithmSHA512,
		SubjectKeyIdentifier,
	}

	for _, oid := range oids {
		oidstr := oid.String()
		t.Run(oidstr, func(t *testing.T) {
			oi, err := NewObjectIdentifier(oidstr)
			assert.NoError(t, err)
			assert.Equal(t, oidstr, oi.String())
		})
	}
}
