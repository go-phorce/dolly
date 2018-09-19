package oid

import (
	crand "crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HashAlgorithmByOID(t *testing.T) {
	for _, c := range oidTests {
		t.Run(c.name, func(t *testing.T) {
			oi := LookupByOID(c.oid)
			assert.NotNil(t, oi, "LookupByOID %s")

			hi, err := HashAlgorithmByOID(oi.String())
			if oi.Type() == AlgHash {
				assert.NoError(t, err)
				assert.NotNil(t, hi)
			} else {
				assert.Error(t, err)
				assert.Nil(t, hi)
			}
		})
	}
}

func Test_HashAlgorithmForPublicKey(t *testing.T) {
	rsa512, err := rsa.GenerateKey(crand.Reader, 512)
	require.NoError(t, err)
	ha := HashAlgorithmForPublicKey(rsa512)
	assert.Equal(t, SHA1.Name(), ha.Name())

	rsa1024, err := rsa.GenerateKey(crand.Reader, 1024)
	require.NoError(t, err)
	ha = HashAlgorithmForPublicKey(rsa1024)
	assert.Equal(t, SHA1.Name(), ha.Name())

	rsa2048, err := rsa.GenerateKey(crand.Reader, 2048)
	require.NoError(t, err)
	ha = HashAlgorithmForPublicKey(rsa2048)
	assert.Equal(t, SHA256.Name(), ha.Name())

	rsa4096, err := rsa.GenerateKey(crand.Reader, 4096)
	require.NoError(t, err)
	ha = HashAlgorithmForPublicKey(rsa4096)
	assert.Equal(t, SHA512.Name(), ha.Name())
}
