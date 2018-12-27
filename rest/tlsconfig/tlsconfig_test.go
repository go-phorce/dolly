package tlsconfig_test

import (
	"crypto/tls"
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

func Test_BuildFromFiles(t *testing.T) {
	pemCert, pemKey, err := testify.MakeSelfCertRSAPem(1)
	require.NoError(t, err)
	require.NotNil(t, pemCert)
	require.NotNil(t, pemKey)

	tmpDir := filepath.Join(os.TempDir(), "tests", "tlsconfig")
	err = os.MkdirAll(tmpDir, os.ModePerm)
	require.NoError(t, err)

	pemFile := filepath.Join(tmpDir, "BuildFromFiles.pem")
	keyFile := filepath.Join(tmpDir, "BuildFromFiles-key.pem")

	err = ioutil.WriteFile(pemFile, pemCert, os.ModePerm)
	require.NoError(t, err)
	err = ioutil.WriteFile(keyFile, pemKey, os.ModePerm)
	require.NoError(t, err)

	cfg, err := tlsconfig.NewServerTLSFromFiles(pemFile, keyFile, "", tls.RequireAndVerifyClientCert)
	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.ClientAuth)

	cfg, err = tlsconfig.NewServerTLSFromFiles(pemFile, keyFile, pemFile, tls.RequireAndVerifyClientCert)
	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.ClientAuth)

	cfg, reloader, err := tlsconfig.NewClientTLSWithReloader(pemFile, keyFile, pemFile, 5*time.Second)
	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, tls.NoClientCert, cfg.ClientAuth)
	require.NotNil(t, reloader)
	assert.NotNil(t, reloader.Keypair())
	reloader.Close()

	c, k := reloader.CertAndKeyFiles()
	assert.Equal(t, pemFile, c)
	assert.Equal(t, keyFile, k)
}
