package cryptoprov_test

import (
	"crypto/elliptic"
	"os"
	"path/filepath"
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
	wd, err := os.Getwd() // package dir
	require.NoError(t, err, "unable to determine current directory")

	binCfg, err := filepath.Abs(filepath.Join(wd, projFolder))
	require.NoError(t, err)

	p11, err := crypto11.ConfigureFromFile(filepath.Join(binCfg, "etc/dev/softhsm_unittest.json"))
	require.NoError(t, err, "failed to load HSM config in dir: %v", binCfg)

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

	t.Run("RSA", func(t *testing.T) {
		rsaKeyLabel := "rsa" + guid.MustCreate()
		rsa, err := d.GenerateRSAKey(rsaKeyLabel, 1024, 1)
		require.NoError(t, err)

		keyID, keyLabel, err := d.IdentifyKey(rsa)
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

		_, err = cp.LoadSigner([]byte(keyURI))
		require.NoError(t, err)
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

		_, err = cp.LoadSigner([]byte(keyURI))
		require.NoError(t, err)
	})
}
