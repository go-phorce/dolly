package csrprov_test

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"

	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/go-phorce/dolly/xpki/cryptoprov/testprov"
	"github.com/go-phorce/dolly/xpki/csrprov"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projFolder = "../.."

func loadInmemProvider(t *testing.T) cryptoprov.Provider {
	p, err := testprov.Init()
	assert.NoError(t, err)
	return p
}

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

func Test_CsrGenkeyCA(t *testing.T) {
	csrCA := `
	{
		"CN": "[TEST] Dolly Root CA",
		"key": {
			"algo": "rsa",
			"size": 4096
		},
		"names": [
			{
				"C": "US",
				"L": "CA",
				"O": "ekspand.com",
				"OU": "dolly-dev"
			}
		],
		"ca": {
			"pathlen": 3
		}
	}`

	defprov := loadProvider(t)
	prov := csrprov.New(defprov)

	req := csrprov.CertificateRequest{
		KeyRequest: prov.NewKeyRequest("Test_CsrGenkeyCA", "ECDSA", 256, csrprov.Signing),
	}

	err := json.Unmarshal([]byte(csrCA), &req)
	require.NoError(t, err, "invalid csr")

	var key, csrPEM, certPEM []byte
	certPEM, csrPEM, key, err = prov.NewRoot(&req)
	require.NoError(t, err, "init CA")
	assert.NotNil(t, csrPEM)

	keyStr := string(key)
	assert.Contains(t, keyStr, "pkcs11:")
	assert.Contains(t, keyStr, "manufacturer=")
	assert.Contains(t, keyStr, "model=")
	assert.Contains(t, keyStr, "serial=")
	assert.Contains(t, keyStr, "token=")
	assert.Contains(t, keyStr, "id=")
	assert.Contains(t, keyStr, "type=private")

	pkuri, err := cryptoprov.ParsePrivateKeyURI(keyStr)
	require.NoError(t, err, "pkcs11uri:%q", keyStr)
	assert.NotEmpty(t, pkuri.ID)
	assert.NotEmpty(t, pkuri.TokenSerial)

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(certPEM))
	require.True(t, ok, "failed to parse root certificate")

	block, _ := pem.Decode([]byte(certPEM))
	require.NotEqual(t, block, "failed to parse certificate PEM")

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err, "failed to parse certificate")
	assert.True(t, cert.IsCA)
	assert.Equal(t, "[TEST] Dolly Root CA", cert.Subject.CommonName)
	assert.Equal(t, []string{"dolly-dev"}, cert.Subject.OrganizationalUnit)
	assert.Equal(t, []string{"ekspand.com"}, cert.Subject.Organization)
	assert.Equal(t, cert.KeyUsage, x509.KeyUsageCRLSign|x509.KeyUsageCertSign)
	assert.Equal(t, 3, cert.MaxPathLen)

	opts := x509.VerifyOptions{
		Roots: roots,
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageCodeSigning,
		},
	}

	_, err = cert.Verify(opts)
	require.NoError(t, err, "failed to verify certificate")
}

func Test_Csr(t *testing.T) {
	defprov := loadProvider(t)
	prov := csrprov.New(defprov)

	csr := prov.NewSigningCertificateRequest("label", "RSA", 1024, "localhost", []csrprov.X509Name{
		{
			O:  "org1",
			OU: "unit1",
		},
	}, []string{"127.0.0.1", "localhost"})
	require.NotNil(t, csr)
	require.NotNil(t, csr.KeyRequest)

	_, _, _, err := prov.NewRoot(csr)
	require.Error(t, err)
	assert.Equal(t, "process request: generate key: validate RSA key: RSA key is too weak: 1024", err.Error())

	csr = prov.NewSigningCertificateRequest("label", "RSA", 2048, "localhost", []csrprov.X509Name{
		{
			O:  "org1",
			OU: "unit1",
		},
	}, []string{"127.0.0.1", "localhost"})
	_, _, _, err = prov.NewRoot(csr)
	require.NoError(t, err)

	_, _, _, err = prov.ParseCsrRequest(csr)
	require.NoError(t, err)
}

func Test_ParseCsr(t *testing.T) {
	defprov := loadProvider(t)
	prov := csrprov.New(defprov)

	tt := []struct {
		name   string
		req    *csrprov.CertificateRequest
		experr string
	}{
		{
			name:   "no key",
			req:    &csrprov.CertificateRequest{},
			experr: "invalid key request",
		},
		{
			name: "valid rsa",
			req: prov.NewSigningCertificateRequest("label", "RSA", 2048, "localhost", []csrprov.X509Name{
				{
					O:  "org1",
					OU: "unit1",
				},
			}, []string{"127.0.0.1", "localhost"}),
			experr: "",
		},
		{
			name: "valid rsa",
			req: prov.NewSigningCertificateRequest("label", "ECDSA", 256, "localhost", []csrprov.X509Name{
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
			cr, k, kid, err := prov.ParseCsrRequest(tc.req)
			if tc.experr != "" {
				assert.Nil(t, k)
				require.Error(t, err)
				assert.Equal(t, tc.experr, err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, cr)
				require.NotNil(t, k)
				assert.NotEmpty(t, kid)
			}
		})
	}
}

func Test_ProcessCsrRequest(t *testing.T) {
	defprov := loadProvider(t)
	prov := csrprov.New(defprov)

	tt := []struct {
		name   string
		req    *csrprov.CertificateRequest
		experr string
	}{
		{
			name:   "empty",
			req:    &csrprov.CertificateRequest{},
			experr: "invalid request: missing subject information",
		},
		{
			name:   "no key",
			req:    &csrprov.CertificateRequest{CN: "localhost"},
			experr: "process request: invalid key request",
		},
		{
			name: "valid rsa",
			req: prov.NewSigningCertificateRequest("label", "RSA", 2048, "localhost", []csrprov.X509Name{
				{
					O:  "org1",
					OU: "unit1",
				},
			}, []string{"127.0.0.1", "localhost"}),
			experr: "",
		},
		{
			name: "valid rsa",
			req: prov.NewSigningCertificateRequest("label", "ECDSA", 256, "localhost", []csrprov.X509Name{
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
			cr, k, kid, pub, err := prov.ProcessCsrRequest(tc.req)
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
