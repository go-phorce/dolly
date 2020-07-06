package cryptoprov_test

import (
	"crypto"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/xpki/crypto11"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/go-phorce/dolly/xpki/cryptoprov/testprov"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadP11Provider(t *testing.T) cryptoprov.Provider {
	f := "/tmp/dolly/softhsm_unittest.json"
	p11, err := crypto11.ConfigureFromFile(f)
	require.NoError(t, err, "failed to load HSM config: %s", f)

	prov, supported := interface{}(p11).(cryptoprov.Provider)
	require.True(t, supported)

	return prov
}

func Test_P11(t *testing.T) {
	prov := loadP11Provider(t)

	inm, err := testprov.Init()
	require.NoError(t, err)

	cp, err := cryptoprov.New(prov, []cryptoprov.Provider{inm})
	require.NoError(t, err)

	err = cp.Add(prov)
	assert.NoError(t, err)
	err = cp.Add(prov)
	require.Error(t, err)
	assert.Equal(t, "duplicate provider specified for manufacturer: SoftHSM", err.Error())

	d := cp.Default()
	assert.NotEmpty(t, d.Manufacturer())
	assert.NotNil(t, d.Model())

	_, err = cp.ByManufacturer("SoftHSM")
	assert.NoError(t, err)
	_, err = cp.ByManufacturer("NetHSM")
	assert.Error(t, err)
	assert.Equal(t, "provider for manufacturer NetHSM not found", err.Error())

	keyURI, keyBytes, err := d.ExportKey("test")
	assert.Error(t, err)
	assert.Empty(t, keyURI)
	assert.Nil(t, keyBytes)

	t.Run("RSA-sign", func(t *testing.T) {
		rsaKeyLabel := "rsa" + guid.MustCreate()
		key, err := d.GenerateRSAKey(rsaKeyLabel, 1024, 1)
		require.NoError(t, err)

		keyID, keyLabel, err := d.IdentifyKey(key)
		require.NoError(t, err)
		assert.NotEmpty(t, keyID)
		assert.Equal(t, rsaKeyLabel, keyLabel)

		keyURI, keyBytes, err := d.ExportKey(keyID)
		assert.NoError(t, err)
		assert.NotEmpty(t, keyURI)
		assert.Nil(t, keyBytes)

		pvkURI, err := cryptoprov.ParsePrivateKeyURI(keyURI)
		require.NoError(t, err)
		assert.Equal(t, "SoftHSM", pvkURI.Manufacturer())
		assert.Equal(t, keyID, pvkURI.ID())

		_, err = cp.LoadGPGPrivateKey(time.Now(), []byte(keyURI))
		require.NoError(t, err)

		_, pvk, err := cp.LoadPrivateKey([]byte(keyURI))
		require.NoError(t, err)

		message := []byte("To Be Signed")
		hashed := sha256.Sum256(message)

		signer, ok := pvk.(crypto.Signer)
		assert.True(t, ok, "crypto.Signer not supported")
		signature, err := signer.Sign(rand.Reader, hashed[:], crypto.SHA256)
		require.NoError(t, err)

		err = rsa.VerifyPKCS1v15(signer.Public().(*rsa.PublicKey), crypto.SHA256, hashed[:], signature)
		require.NoError(t, err)
	})

	t.Run("RSA-encrypt", func(t *testing.T) {
		rsaKeyLabel := "rsa" + guid.MustCreate()
		key, err := d.GenerateRSAKey(rsaKeyLabel, 1024, 2)
		require.NoError(t, err)

		keyID, keyLabel, err := d.IdentifyKey(key)
		require.NoError(t, err)
		assert.NotEmpty(t, keyID)
		assert.Equal(t, rsaKeyLabel, keyLabel)

		keyURI, keyBytes, err := d.ExportKey(keyID)
		assert.NoError(t, err)
		assert.NotEmpty(t, keyURI)
		assert.Nil(t, keyBytes)

		pvkURI, err := cryptoprov.ParsePrivateKeyURI(keyURI)
		require.NoError(t, err)
		assert.Equal(t, "SoftHSM", pvkURI.Manufacturer())
		assert.Equal(t, keyID, pvkURI.ID())

		_, pvk, err := cp.LoadPrivateKey([]byte(keyURI))
		require.NoError(t, err)

		message := []byte("To Be Encrypted")

		decryptor, ok := pvk.(crypto.Decrypter)
		assert.True(t, ok, "crypto.Decrypter not supported")

		encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, decryptor.Public().(*rsa.PublicKey), message)
		require.NoError(t, err)

		decrypted, err := decryptor.Decrypt(rand.Reader, encrypted, nil)
		require.NoError(t, err)
		assert.Equal(t, message, decrypted)
	})

	t.Run("ECDSA", func(t *testing.T) {
		ecdsaKeyLabel := "ecdsa" + guid.MustCreate()
		rsa, err := d.GenerateECDSAKey(ecdsaKeyLabel, elliptic.P256())
		require.NoError(t, err)

		keyID, keyLabel, err := d.IdentifyKey(rsa)
		require.NoError(t, err)
		assert.NotEmpty(t, keyID)
		assert.Equal(t, ecdsaKeyLabel, keyLabel)

		keyURI, keyBytes, err := d.ExportKey(keyID)
		assert.NoError(t, err)
		assert.NotEmpty(t, keyURI)
		assert.Nil(t, keyBytes)

		pvkURI, err := cryptoprov.ParsePrivateKeyURI(keyURI)
		require.NoError(t, err)
		assert.Equal(t, "SoftHSM", pvkURI.Manufacturer())
		assert.Equal(t, keyID, pvkURI.ID())

		_, err = cp.LoadGPGPrivateKey(time.Now(), []byte(keyURI))
		require.NoError(t, err)

		_, _, err = cp.LoadPrivateKey([]byte(keyURI))
		require.NoError(t, err)
	})
}
