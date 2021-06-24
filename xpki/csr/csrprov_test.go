package csr_test

import (
	"crypto"
	"testing"

	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/go-phorce/dolly/xpki/csr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadProvider(t *testing.T) cryptoprov.Provider {
	cfgfile := "/tmp/dolly/softhsm_unittest.json"

	err := cryptoprov.Register("SoftHSM", cryptoprov.Crypto11Loader)
	assert.NoError(t, err)
	defer cryptoprov.Unregister("SoftHSM")

	p, err := cryptoprov.LoadProvider(cfgfile)
	require.NoError(t, err)

	assert.Equal(t, "SoftHSM", p.Manufacturer())

	return p
}

func TestGenerateKeyAndRequest(t *testing.T) {
	defprov := loadProvider(t)
	prov := csr.NewProvider(defprov)

	tt := []struct {
		name   string
		req    *csr.CertificateRequest
		experr string
	}{
		{
			name:   "no key",
			req:    &csr.CertificateRequest{},
			experr: "invalid key request",
		},
		{
			name: "valid rsa",
			req: prov.NewSigningCertificateRequest("label", "RSA", 2048, "localhost", []csr.X509Name{
				{
					O:  "org1",
					OU: "unit1",
				},
			}, []string{"127.0.0.1", "localhost"}),
			experr: "",
		},
		{
			name: "valid rsa",
			req: prov.NewSigningCertificateRequest("label", "ECDSA", 256, "localhost", []csr.X509Name{
				{
					O:  "org1",
					OU: "unit1",
				},
			}, []string{"127.0.0.1", "localhost"}),
			experr: "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cr, k, kid, err := prov.GenerateKeyAndRequest(tc.req)
			if tc.experr != "" {
				assert.Nil(t, k)
				require.Error(t, err)
				assert.Equal(t, tc.experr, err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, cr)
				require.NotNil(t, k)
				assert.NotEmpty(t, kid)

				signer := k.(crypto.Signer)
				assert.Equal(t, tc.req.KeyRequest.SigAlgo(), csr.DefaultSigAlgo(signer))
			}
		})
	}
}

func TestCreateRequestAndExportKey(t *testing.T) {
	defprov := loadProvider(t)
	prov := csr.NewProvider(defprov)

	tt := []struct {
		name   string
		req    *csr.CertificateRequest
		experr string
	}{
		{
			name:   "empty",
			req:    &csr.CertificateRequest{},
			experr: "process request: invalid key request",
		},
		{
			name:   "no key",
			req:    &csr.CertificateRequest{CommonName: "localhost"},
			experr: "process request: invalid key request",
		},
		{
			name: "valid rsa",
			req: prov.NewSigningCertificateRequest("label", "RSA", 2048, "localhost", []csr.X509Name{
				{
					O:  "org1",
					OU: "unit1",
				},
			}, []string{"127.0.0.1", "localhost"}),
			experr: "",
		},
		{
			name: "valid rsa",
			req: prov.NewSigningCertificateRequest("label", "ECDSA", 256, "localhost", []csr.X509Name{
				{
					O:  "org1",
					OU: "unit1",
				},
			}, []string{"127.0.0.1", "localhost"}),
			experr: "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cr, k, kid, pub, err := prov.CreateRequestAndExportKey(tc.req)
			if tc.experr != "" {
				assert.Nil(t, k)
				require.Error(t, err)
				assert.Equal(t, tc.experr, err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, cr)
				require.NotNil(t, k)
				require.NotNil(t, pub)
				assert.NotEmpty(t, kid)
			}
		})
	}
}
