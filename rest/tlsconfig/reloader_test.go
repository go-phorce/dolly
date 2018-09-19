package tlsconfig_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-phorce/dolly/rest/tlsconfig"
	"github.com/go-phorce/dolly/testify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_KeypairReloader(t *testing.T) {
	// try 3 times to ensure notifications
	for i := 0; i < 3; i++ {
		keypairReloader(t)
	}
}

func keypairReloader(t *testing.T) {
	now := time.Now().UTC()
	pemCert, pemKey, err := testify.MakeSelfCertRSAPem(1)
	require.NoError(t, err)
	require.NotNil(t, pemCert)
	require.NotNil(t, pemKey)

	pemFile := filepath.Join(os.TempDir(), "test-BuildTLSConfig.pem")
	keyFile := filepath.Join(os.TempDir(), "test-BuildTLSConfig-key.pem")

	err = ioutil.WriteFile(pemFile, pemCert, os.ModePerm)
	require.NoError(t, err)
	err = ioutil.WriteFile(keyFile, pemKey, os.ModePerm)
	require.NoError(t, err)

	time.Sleep(100)

	k, err := tlsconfig.NewKeypairReloader(pemFile, keyFile, 1*time.Second)
	require.NoError(t, err)
	require.NotNil(t, k)
	defer k.Close()

	// time.Sleep(time.Second * 1)

	loadedAt := k.LoadedAt()
	assert.True(t, loadedAt.After(now), "loaded time must be after test start time")
	assert.Equal(t, uint32(1), k.LoadedCount())

	err = ioutil.WriteFile(pemFile, pemCert, os.ModePerm)
	require.NoError(t, err)
	err = ioutil.WriteFile(keyFile, pemKey, os.ModePerm)
	require.NoError(t, err)
	err = ioutil.WriteFile(pemFile, pemCert, os.ModePerm)
	require.NoError(t, err)

	time.Sleep(time.Second * 1)

	loadedAt2 := k.LoadedAt()
	count := int(k.LoadedCount())
	assert.True(t, count >= 2 && count <= 4, "must be loaded at start, whithin period and after, loaded: %d", k.LoadedCount())
	assert.True(t, loadedAt2.After(loadedAt), "re-loaded time must be after last loaded time")

	err = ioutil.WriteFile(pemFile, pemCert, os.ModePerm)
	require.NoError(t, err)
	err = ioutil.WriteFile(keyFile, pemKey, os.ModePerm)
	require.NoError(t, err)
	err = ioutil.WriteFile(pemFile, pemCert, os.ModePerm)
	require.NoError(t, err)
	err = ioutil.WriteFile(keyFile, pemKey, os.ModePerm)
	require.NoError(t, err)

	time.Sleep(time.Second * 1)

	loadedAt3 := k.LoadedAt()
	count = int(k.LoadedCount())
	assert.True(t, count >= 3 && count <= 5, "must be loaded at start, whithin period and after, loaded: %d", k.LoadedCount())
	assert.True(t, loadedAt3.After(loadedAt2), "re-loaded time must be after last loaded time")
}
