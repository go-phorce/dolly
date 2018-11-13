package oid

import (
	"encoding/asn1"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewObjectIdentifierFromOID(t *testing.T) {
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

func Test_NewObjectIdentifier2(t *testing.T) {
	t.Run("should fails", func(t *testing.T) {
		_, err := NewObjectIdentifier("")
		assert.Error(t, err)
		var s string
		_, err = NewObjectIdentifier(s)
		assert.Error(t, err)
		_, err = NewObjectIdentifier("1.2.")
		assert.Error(t, err)
		_, err = NewObjectIdentifier("1.2.a.3")
		assert.Error(t, err)
		_, err = NewObjectIdentifier("{iso(1) member-body(2) us(840) rsadsi(113549) pkcs(1) pkcs-9(9) messageDigest(4}")
		assert.Error(t, err)
	})

	t.Run("asn1 notation", func(t *testing.T) {
		oid1, err := NewObjectIdentifier("{iso(1) member-body(2) us(840) rsadsi(113549) pkcs(1) pkcs-9(9) messageDigest(4)}")
		require.NoError(t, err)

		oid2, err := NewObjectIdentifier("1.2.840.113549.1.9.4")
		require.NoError(t, err)

		assert.Equal(t, oid2, oid1)
	})
}
