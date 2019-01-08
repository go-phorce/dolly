package csrprov_test

import (
	"testing"

	"github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/go-phorce/dolly/xpki/csrprov"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CsrName(t *testing.T) {
	defprov := loadProvider(t)
	prov := csrprov.New(defprov)

	cr := prov.NewSigningCertificateRequest("label", "RSA", 1024, "localhost", []csrprov.X509Name{
		{
			O:  "org1",
			OU: "unit1",
		},
	}, []string{"127.0.0.1", "localhost"})
	require.NotNil(t, cr)
	require.NotNil(t, cr.KeyRequest)

	assert.Equal(t, "CN=localhost,OU=unit1,O=org1", cr.Name().String())

	name := &csrprov.X509Name{
		O:  "org1",
		OU: "unit1",
	}
	cname := name.ConvertToCFSSL()
	assert.Equal(t, name.C, cname.C)
	assert.Equal(t, name.ST, cname.ST)
	assert.Equal(t, name.L, cname.L)
	assert.Equal(t, name.OU, cname.OU)
	assert.Equal(t, name.O, cname.O)
	assert.Equal(t, name.SerialNumber, cname.SerialNumber)

	cname = &csr.Name{}
	name.CopyToCFSSL(cname)
	assert.Equal(t, name.C, cname.C)
	assert.Equal(t, name.ST, cname.ST)
	assert.Equal(t, name.L, cname.L)
	assert.Equal(t, name.OU, cname.OU)
	assert.Equal(t, name.O, cname.O)
	assert.Equal(t, name.SerialNumber, cname.SerialNumber)

	cfg := &csrprov.CAConfig{
		PathLength:  1,
		PathLenZero: true,
		Expiry:      "Expiry",
		Backdate:    "Backdate",
	}
	ccfg := cfg.ConvertToCFSSL()
	assert.Equal(t, cfg.PathLength, ccfg.PathLength)
	assert.Equal(t, cfg.PathLenZero, ccfg.PathLenZero)
	assert.Equal(t, cfg.Expiry, ccfg.Expiry)
	assert.Equal(t, cfg.Backdate, ccfg.Backdate)

	ccfg = &csr.CAConfig{}
	cfg.CopyToCFSSL(ccfg)
	assert.Equal(t, cfg.PathLength, ccfg.PathLength)
	assert.Equal(t, cfg.PathLenZero, ccfg.PathLenZero)
	assert.Equal(t, cfg.Expiry, ccfg.Expiry)
	assert.Equal(t, cfg.Backdate, ccfg.Backdate)
}

func Test_ValidateCSR(t *testing.T) {
	tt := []struct {
		req    csrprov.CertificateRequest
		experr string
	}{
		{
			req:    csrprov.CertificateRequest{CN: "already generated"},
			experr: "",
		},
		{
			req:    csrprov.CertificateRequest{},
			experr: "missing subject information",
		},
		{
			req:    csrprov.CertificateRequest{Names: []csrprov.X509Name{}},
			experr: "missing subject information",
		},
		{
			req:    csrprov.CertificateRequest{Names: make([]csrprov.X509Name, 2)},
			experr: "empty name",
		},
		{
			req:    csrprov.CertificateRequest{Names: []csrprov.X509Name{{O: "org1"}}},
			experr: "",
		},
	}

	for _, tc := range tt {
		err := csrprov.ValidateCSR(&tc.req)
		if tc.experr != "" {
			require.Error(t, err)
			assert.Equal(t, tc.experr, err.Error())
		} else {
			require.NoError(t, err)
		}
	}
}

func Test_MakeCAPolicy(t *testing.T) {
	tt := []struct {
		name   string
		req    csrprov.CertificateRequest
		experr string
	}{
		{
			name:   "No CA",
			req:    csrprov.CertificateRequest{CN: "already generated", CA: nil},
			experr: "",
		},
		{
			name: "Expiry 1h",
			req: csrprov.CertificateRequest{CA: &csrprov.CAConfig{
				Expiry: "1h",
			}},
			experr: "",
		},
		{
			name: "bad expiry",
			req: csrprov.CertificateRequest{CA: &csrprov.CAConfig{
				Expiry: "~hh",
			}},
			experr: "time: invalid duration ~hh",
		},
		{
			name: "Backdate 1h",
			req: csrprov.CertificateRequest{CA: &csrprov.CAConfig{
				Backdate: "1h",
			}},
			experr: "",
		},
		{
			name: "bad backdate",
			req: csrprov.CertificateRequest{CA: &csrprov.CAConfig{
				Backdate: "~hh",
			}},
			experr: "time: invalid duration ~hh",
		},
		{
			name: "Expiry 1h, Backdate 1h",
			req: csrprov.CertificateRequest{CA: &csrprov.CAConfig{
				Expiry:   "1h",
				Backdate: "1h",
			}},
			experr: "",
		},
		{
			name: "ignore invalid 'pathlenzero' value",
			req: csrprov.CertificateRequest{CA: &csrprov.CAConfig{
				PathLength:  2,
				PathLenZero: true,
				Expiry:      "1h",
				Backdate:    "1h",
			}},
			experr: "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			p, err := csrprov.MakeCAPolicy(&tc.req)
			if tc.experr != "" {
				assert.Nil(t, p)
				require.Error(t, err)
				assert.Equal(t, tc.experr, err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, p)
			}
		})
	}
}

func Test_ParseCaFiles(t *testing.T) {
	tt := []struct {
		name   string
		ca     string
		key    string
		experr string
	}{
		{
			name:   "No CA",
			experr: "load ca file: open : no such file or directory",
		},
		{
			name:   "no key",
			ca:     "testdata/test_dolly_root_CA.pem",
			experr: "load ca-key file: open : no such file or directory",
		},
		{
			name:   "both",
			ca:     "testdata/test_dolly_root_CA.pem",
			key:    "testdata/test_dolly_root_CA-key.pem",
			experr: "",
		},
		{
			name:   "invalid",
			key:    "testdata/test_dolly_root_CA.pem",
			ca:     "testdata/test_dolly_root_CA-key.pem",
			experr: `parse ca file: {"code":1003,"message":"Failed to parse certificate"}`,
		},
	}

	defprov := loadProvider(t)
	crypto, err := cryptoprov.New(defprov, nil)
	require.NoError(t, err)

	policy := initca.CAPolicy()

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			k, c, err := csrprov.ParseCaFiles(tc.ca, tc.key)
			if tc.experr != "" {
				assert.Nil(t, c)
				require.Error(t, err)
				assert.Equal(t, tc.experr, err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, k)
				require.NotNil(t, c)

				ls, cs, err := csrprov.NewLocalCASignerFromFile(crypto, tc.ca, tc.key, policy)
				require.NoError(t, err)
				require.NotNil(t, ls)
				require.NotNil(t, cs)
			}
		})
	}
}

func Test_NewLocalCASignerFromFile(t *testing.T) {
	tt := []struct {
		name   string
		ca     string
		key    string
		experr string
	}{
		{
			name:   "No CA",
			experr: "load ca file: open : no such file or directory",
		},
		{
			name:   "no key",
			ca:     "testdata/test_dolly_root_CA.pem",
			experr: "load ca-key file: open : no such file or directory",
		},
		{
			name:   "both",
			ca:     "testdata/test_dolly_root_CA.pem",
			key:    "testdata/test_dolly_root_CA-key.pem",
			experr: "",
		},
		{
			name:   "invalid key",
			ca:     "testdata/test_dolly_root_CA-key.pem",
			key:    "testdata/test_dolly_root_CA.pem",
			experr: `failed to parse key`,
		},
		{
			name:   "invalid cert",
			ca:     "testdata/test_dolly_root_CA-key.pem",
			key:    "testdata/test_dolly_root_CA-key.pem",
			experr: `{"code":1003,"message":"Failed to parse certificate"}`,
		},
	}

	defprov := loadProvider(t)
	crypto, err := cryptoprov.New(defprov, nil)
	require.NoError(t, err)

	policy := initca.CAPolicy()

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ls, cs, err := csrprov.NewLocalCASignerFromFile(crypto, tc.ca, tc.key, policy)
			if tc.experr != "" {
				assert.Nil(t, cs)
				require.Error(t, err)
				assert.Equal(t, tc.experr, err.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, ls)
				require.NotNil(t, cs)

				_, _, err = csrprov.NewLocalCASignerFromFile(crypto, tc.ca, tc.key, nil)
				require.Error(t, err)
				assert.Equal(t, "invalid parameter: policy", err.Error())

				_, _, err = csrprov.NewLocalCASignerFromFile(crypto, tc.ca, tc.key, &config.Signing{})
				require.Error(t, err)
				assert.Equal(t, `{"code":5200,"message":"Invalid or unknown policy"}`, err.Error())
			}
		})
	}
}
