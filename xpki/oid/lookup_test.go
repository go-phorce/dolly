package oid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type oidInfo struct {
	name string
	oid  string
	typ  AlgType
	info Info
}

var oidTests = []oidInfo{
	{oid: "1.2.840.113549.1.1.1", name: "RSA", typ: AlgPubKey, info: RSA},
	{oid: "1.2.840.10045.2.1", name: "ECDSA", typ: AlgPubKey, info: ECDSA},
	{oid: "1.3.14.3.2.26", name: "SHA1", typ: AlgHash, info: SHA1},
	{oid: "2.16.840.1.101.3.4.2.1", name: "SHA256", typ: AlgHash, info: SHA256},
	{oid: "2.16.840.1.101.3.4.2.2", name: "SHA384", typ: AlgHash, info: SHA384},
	{oid: "2.16.840.1.101.3.4.2.3", name: "SHA512", typ: AlgHash, info: SHA512},
	{oid: "2.16.840.1.101.3.4.2.7", name: "SHA3-224", typ: AlgHash, info: SHA3x224},
	{oid: "2.16.840.1.101.3.4.2.8", name: "SHA3-256", typ: AlgHash, info: SHA3x256},
	{oid: "2.16.840.1.101.3.4.2.9", name: "SHA3-384", typ: AlgHash, info: SHA3x384},
	{oid: "2.16.840.1.101.3.4.2.10", name: "SHA3-512", typ: AlgHash, info: SHA3x512},
	{oid: "2.16.840.1.101.3.4.2.11", name: "SHAKE128", typ: AlgHash, info: SHAKE128},
	{oid: "2.16.840.1.101.3.4.2.12", name: "SHAKE256", typ: AlgHash, info: SHAKE256},
	{oid: "1.2.840.113549.1.1.5", name: "RSA-SHA1", typ: AlgSig, info: RSAWithSHA1},
	{oid: "1.2.840.113549.1.1.11", name: "RSA-SHA256", typ: AlgSig, info: RSAWithSHA256},
	{oid: "1.2.840.113549.1.1.12", name: "RSA-SHA384", typ: AlgSig, info: RSAWithSHA384},
	{oid: "1.2.840.113549.1.1.13", name: "RSA-SHA512", typ: AlgSig, info: RSAWithSHA512},
	{oid: "1.2.840.10045.4.1", name: "ECDSA-SHA1", typ: AlgSig, info: ECDSAWithSHA1},
	{oid: "1.2.840.10045.4.3.2", name: "ECDSA-SHA256", typ: AlgSig, info: RSAWithSHA256},
	{oid: "1.2.840.10045.4.3.3", name: "ECDSA-SHA384", typ: AlgSig, info: RSAWithSHA384},
	{oid: "1.2.840.10045.4.3.4", name: "ECDSA-SHA512", typ: AlgSig, info: RSAWithSHA512},
}

func Test_LookupByOID(t *testing.T) {
	for _, c := range oidTests {
		t.Run(c.name, func(t *testing.T) {
			i1 := LookupByOID(c.oid)
			assert.NotNil(t, i1, "LookupByOID %s")
			assert.Equal(t, c.name, i1.Name())
			assert.Equal(t, c.typ, i1.Type())
			assert.Equal(t, c.oid, i1.String())

			i2 := LookupByName(c.name)
			assert.NotNil(t, i2)
			assert.Equal(t, c.name, i2.Name())
			assert.Equal(t, c.typ, i2.Type())
			assert.Equal(t, c.oid, i2.String())

			oid, err := NewObjectIdentifier(c.oid)
			require.NoError(t, err)
			assert.Equal(t, c.oid, oid.String())
		})
	}
}
