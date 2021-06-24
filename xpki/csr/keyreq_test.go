package csr_test

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"testing"

	"github.com/go-phorce/dolly/xpki/cryptoprov/inmemcrypto"
	"github.com/go-phorce/dolly/xpki/csr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyRequest(t *testing.T) {
	defprov := inmemcrypto.NewProvider()
	tt := []struct {
		algo   string
		size   int
		expalg x509.SignatureAlgorithm
		experr string
	}{
		{"rsa", 512, x509.SHA1WithRSA, "validate RSA key: RSA key is too weak: 512"},
		{"RSA", 1024, x509.SHA1WithRSA, "validate RSA key: RSA key is too weak: 1024"},
		{"RSA", 2048, x509.SHA256WithRSA, ""},
		{"RSA", 3072, x509.SHA384WithRSA, ""},
		{"rsa", 4096, x509.SHA512WithRSA, ""},
		{"rsa", 8192, x509.SHA512WithRSA, "validate RSA key: RSA key size too large: 8192"},
		{"rsa", 168192, x509.SHA512WithRSA, "validate RSA key: RSA key size too large: 168192"},
		{"ecdsa", 521, x509.ECDSAWithSHA512, ""},
		{"ECDSA", 384, x509.ECDSAWithSHA384, ""},
		{"ECDSA", 256, x509.ECDSAWithSHA256, ""},
		{"ECDSA", 128, x509.ECDSAWithSHA1, "validate ECDSA key: invalid curve size: 128"},
		{"DSA", 256, x509.UnknownSignatureAlgorithm, "invalid algorithm: DSA"},
	}

	for _, tc := range tt {
		label := fmt.Sprintf("%s_%d", tc.algo, tc.size)
		t.Run(label, func(t *testing.T) {
			assert.Equal(t, tc.expalg, csr.SigAlgo(tc.algo, tc.size))

			kr := csr.NewKeyRequest(defprov, label, tc.algo, tc.size, csr.SigningKey)
			priv, err := kr.Generate()
			if tc.experr != "" {
				require.Error(t, err)
				assert.Equal(t, tc.experr, err.Error())
			} else {
				require.NoError(t, err)
				signer := priv.(crypto.Signer)
				assert.Equal(t, tc.expalg, csr.DefaultSigAlgo(signer))
			}
		})
	}
}
