package testprov

import (
	"crypto"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_InitProvider(t *testing.T) {
	p, err := Init()
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.inMemProv)
	assert.NotNil(t, p.rsaKeyGenerator)
	assert.NotNil(t, p.ecdsaKeyGenerator)
}

func Test_GenerateKeys(t *testing.T) {
	p, err := Init()
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.inMemProv)
	assert.NotNil(t, p.rsaKeyGenerator)
	assert.NotNil(t, p.ecdsaKeyGenerator)

	priv, err := p.GenerateECDSAKey("inmemoryECDSA", elliptic.P256())
	require.NoError(t, err)
	assert.NotNil(t, priv)

	keyID1, label, err := p.IdentifyKey(priv)
	require.NoError(t, err)
	assert.Equal(t, "inmemoryECDSA", label)

	l := len(p.inMemProv.keyIDToPvk)
	assert.Equal(t, 1, l)
	keyURI, keyBytes, err := p.ExportKey(keyID1)
	require.NoError(t, err)
	assert.Nil(t, keyBytes)
	expectedURI := fmt.Sprintf("pkcs11:manufacturer=testprov;model=inmem;serial=20764350726;token=inmemoryECDSA;id=%s;type=private", keyID1)
	assert.Equal(t, expectedURI, keyURI)

	s, err := p.GetKey(keyID1)
	require.NoError(t, err)
	si, ok := s.(*provImpl)
	assert.True(t, ok)
	assert.NotNil(t, si.Public())

	pvk, err := p.GetKey(keyID1)
	require.NoError(t, err)
	assert.NotNil(t, pvk)

	priv, err = p.GenerateRSAKey("inmemoryRSA", 4096, 1)
	require.NoError(t, err)
	assert.NotNil(t, priv)

	keyID2, label, err := p.IdentifyKey(priv)
	require.NoError(t, err)
	assert.Equal(t, "inmemoryRSA", label)

	l = len(p.inMemProv.keyIDToPvk)
	assert.Equal(t, 2, l)
	keyURI, _, err = p.ExportKey(keyID2)
	require.NoError(t, err)
	expectedURI = fmt.Sprintf("pkcs11:manufacturer=testprov;model=inmem;serial=20764350726;token=inmemoryRSA;id=%s;type=private", keyID2)
	assert.Equal(t, expectedURI, keyURI)

	s, err = p.GetKey(keyID2)
	require.NoError(t, err)
	si, ok = s.(*provImpl)
	assert.True(t, ok)
	assert.NotNil(t, si.Public())

	pvk, err = p.GetKey(keyID2)
	require.NoError(t, err)
	assert.NotNil(t, pvk)
}

func Test_SignECDSA(t *testing.T) {
	p, err := Init()
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.inMemProv)
	assert.NotNil(t, p.rsaKeyGenerator)
	assert.NotNil(t, p.ecdsaKeyGenerator)

	priv, err := p.GenerateECDSAKey("inmemoryECDSA", elliptic.P256())
	require.NoError(t, err)
	assert.NotNil(t, priv)

	keyID, label, err := p.IdentifyKey(priv)
	require.NoError(t, err)
	assert.Equal(t, "inmemoryECDSA", label)

	l := len(p.inMemProv.keyIDToPvk)
	assert.Equal(t, 1, l)
	keyURI, _, err := p.ExportKey(keyID)
	require.NoError(t, err)

	expectedURI := fmt.Sprintf("pkcs11:manufacturer=testprov;model=inmem;serial=20764350726;token=inmemoryECDSA;id=%s;type=private", keyID)
	assert.Equal(t, expectedURI, keyURI)

	signer, ok := priv.(crypto.Signer)
	require.True(t, ok)

	digest := certutil.SHA1([]byte(keyID))
	_, err = signer.Sign(rand.Reader, digest, crypto.SHA1)
	require.NoError(t, err)
}

func Test_SignRSA(t *testing.T) {
	p, err := Init()
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.inMemProv)
	assert.NotNil(t, p.rsaKeyGenerator)
	assert.NotNil(t, p.ecdsaKeyGenerator)

	priv, err := p.GenerateRSAKey("inmemoryRSA", 1024, 1)
	require.NoError(t, err)
	assert.NotNil(t, priv)

	keyID, label, err := p.IdentifyKey(priv)
	require.NoError(t, err)
	assert.Equal(t, "inmemoryRSA", label)

	l := len(p.inMemProv.keyIDToPvk)
	assert.Equal(t, 1, l)
	keyURI, _, err := p.ExportKey(keyID)
	require.NoError(t, err)

	expectedURI := fmt.Sprintf("pkcs11:manufacturer=testprov;model=inmem;serial=20764350726;token=inmemoryRSA;id=%s;type=private", keyID)
	assert.Equal(t, expectedURI, keyURI)

	signer, ok := priv.(crypto.Signer)
	require.True(t, ok)

	digest := certutil.SHA1([]byte(keyID))
	_, err = signer.Sign(rand.Reader, digest, crypto.SHA1)
	require.NoError(t, err)
}
