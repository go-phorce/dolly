package tlsconfig_test

import (
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-phorce/dolly/rest/tlsconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkCert(t *testing.T, hours int) (pemCert, pemKey []byte) {
	// rsa key pair
	key, err := rsa.GenerateKey(crand.Reader, 512)
	require.NoError(t, err)

	// certificate
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(math.MaxInt64)),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore: time.Now().UTC().Add(-time.Hour),
		NotAfter:  time.Now().UTC().Add(time.Hour * time.Duration(hours)),
	}
	cert, err := x509.CreateCertificate(crand.Reader, certTemplate, certTemplate, &key.PublicKey, key)
	require.NoError(t, err)

	pemKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pemCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	return
}

func Test_BuildFromFiles(t *testing.T) {
	pemCert, pemKey := mkCert(t, 1)
	require.NotNil(t, pemCert)
	require.NotNil(t, pemKey)

	tmpDir := filepath.Join(os.TempDir(), "tests", "tlsconfig")
	err := os.MkdirAll(tmpDir, os.ModePerm)
	require.NoError(t, err)

	pemFile := filepath.Join(tmpDir, "BuildFromFiles.pem")
	keyFile := filepath.Join(tmpDir, "BuildFromFiles-key.pem")

	err = ioutil.WriteFile(pemFile, pemCert, os.ModePerm)
	require.NoError(t, err)
	err = ioutil.WriteFile(keyFile, pemKey, os.ModePerm)
	require.NoError(t, err)

	cfg, err := tlsconfig.BuildFromFiles(pemFile, keyFile, "", true)
	assert.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.ClientAuth)
}
