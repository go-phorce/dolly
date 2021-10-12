package oid

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SignatureAlgorithmByOID(t *testing.T) {
	for _, c := range oidTests {
		t.Run(c.name, func(t *testing.T) {
			oi := LookupByOID(c.oid)
			assert.NotNil(t, oi, "LookupByOID %s")

			hi, err := SignatureAlgorithmByOID(oi.String())
			if oi.Type() == AlgSig {
				assert.NoError(t, err)
				assert.NotNil(t, hi)
			} else {
				assert.Error(t, err)
				assert.Nil(t, hi)

			}
		})
	}
}

func Test_SignatureAlgorithmInfo(t *testing.T) {
	s := RSAWithSHA1
	assert.Equal(t, crypto.SHA1, s.HashFunc())
	assert.Equal(t, asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 5}, s.OID())
	assert.Equal(t, "{iso(1) member-body(2) us(840) rsadsi(113549) pkcs(1) pkcs-1(1) sha1-with-rsa-signature(5)}", s.Registration())
}

func Test_SignatureAlgorithmByName(t *testing.T) {
	s, err := SignatureAlgorithmByName(RSAWithSHA1.Name())
	require.NoError(t, err)
	assert.Equal(t, RSAWithSHA1, *s)

	_, err = SignatureAlgorithmByName("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func Test_SignatureAlgorithmByKey(t *testing.T) {
	_, err := SignatureAlgorithmByKey(0)
	assert.Error(t, err)

	t.Run("RSA", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 512)
		require.NoError(t, err)

		pa, err := SignatureAlgorithmByKey(key)
		require.NoError(t, err)
		assert.Equal(t, &RSA, pa)

		sa, err := SignatureAlgorithmByKeyAndHash(key, SHA1.hash)
		require.NoError(t, err)
		assert.Equal(t, &RSAWithSHA1, sa)

	})
	t.Run("ECDSA", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		pa, err := SignatureAlgorithmByKey(key)
		require.NoError(t, err)
		assert.Equal(t, &ECDSA, pa)

		_, err = SignatureAlgorithmByKey(0)
		assert.Error(t, err)

		_, err = SignatureAlgorithmByKeyAndHash(0, SHA1.hash)
		assert.Error(t, err)

		sa, err := SignatureAlgorithmByKeyAndHash(key, SHA1.hash)
		require.NoError(t, err)
		assert.Equal(t, &ECDSAWithSHA1, sa)

		sa, err = SignatureAlgorithmByKeyAndHash(key, SHA256.hash)
		require.NoError(t, err)
		assert.Equal(t, &ECDSAWithSHA256, sa)

		sa, err = SignatureAlgorithmByKeyAndHash(key, SHA384.hash)
		require.NoError(t, err)
		assert.Equal(t, &ECDSAWithSHA384, sa)

		sa, err = SignatureAlgorithmByKeyAndHash(key, SHA512.hash)
		require.NoError(t, err)
		assert.Equal(t, &ECDSAWithSHA512, sa)
	})
}

func Test_SignatureAlgorithmByX509(t *testing.T) {
	assert.Equal(t, &RSAWithSHA1, SignatureAlgorithmByX509(x509.SHA1WithRSA))
	assert.Equal(t, &RSAWithSHA256, SignatureAlgorithmByX509(x509.SHA256WithRSA))
	assert.Equal(t, &RSAWithSHA384, SignatureAlgorithmByX509(x509.SHA384WithRSA))
	assert.Equal(t, &RSAWithSHA512, SignatureAlgorithmByX509(x509.SHA512WithRSA))
	assert.Equal(t, &ECDSAWithSHA1, SignatureAlgorithmByX509(x509.ECDSAWithSHA1))
	assert.Equal(t, &ECDSAWithSHA256, SignatureAlgorithmByX509(x509.ECDSAWithSHA256))
	assert.Equal(t, &ECDSAWithSHA384, SignatureAlgorithmByX509(x509.ECDSAWithSHA384))
	assert.Equal(t, &ECDSAWithSHA512, SignatureAlgorithmByX509(x509.ECDSAWithSHA512))

	assert.Panics(t, func() {
		SignatureAlgorithmByX509(x509.DSAWithSHA1)
	})
}
