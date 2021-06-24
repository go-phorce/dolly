package csr_test

import (
	"crypto/x509"
	"testing"

	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/go-phorce/dolly/xpki/csr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCertificateRequestValidate(t *testing.T) {
	tcases := []struct {
		r   *csr.CertificateRequest
		err string
	}{
		{
			r:   &csr.CertificateRequest{CommonName: "ekspand.com"},
			err: "",
		},
		{
			r: &csr.CertificateRequest{
				Names: []csr.X509Name{{O: "ekspand"}},
			},
			err: "",
		},
		{
			r:   &csr.CertificateRequest{Names: []csr.X509Name{{}}},
			err: "empty name",
		},
	}

	for _, tc := range tcases {
		err := tc.r.Validate()
		if tc.err != "" {
			require.Error(t, err)
			assert.Equal(t, tc.err, err.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestCertificateRequestName(t *testing.T) {
	r := &csr.CertificateRequest{
		CommonName:   "ekspand.com",
		SerialNumber: "1234",
		Names: []csr.X509Name{
			{
				O:  "ekspand",
				ST: "WA",
				C:  "US",
			},
		},
	}

	n := r.Name()
	assert.Equal(t, "SERIALNUMBER=1234,CN=ekspand.com,O=ekspand,ST=WA,C=US", n.String())
}

func TestX509SubjectName(t *testing.T) {
	r := &csr.X509Subject{
		CommonName:   "ekspand.com",
		SerialNumber: "1234",
		Names: []csr.X509Name{
			{
				O:  "ekspand",
				ST: "WA",
				C:  "US",
			},
		},
	}

	n := r.Name()
	assert.Equal(t, "SERIALNUMBER=1234,CN=ekspand.com,O=ekspand,ST=WA,C=US", n.String())
}

func TestPopulateName(t *testing.T) {
	req := &csr.CertificateRequest{
		CommonName:   "ekspand.com",
		SerialNumber: "1234",
		Names: []csr.X509Name{
			{
				O:  "ekspand",
				ST: "CA",
				C:  "USA",
			},
		},
	}
	n := req.Name()

	subj := &csr.X509Subject{
		Names: []csr.X509Name{
			{
				O:  "ekspand.com",
				ST: "WA",
				C:  "US",
			},
		},
	}
	n2 := csr.PopulateName(nil, n)
	assert.Equal(t, "SERIALNUMBER=1234,CN=ekspand.com,O=ekspand,ST=CA,C=USA", n2.String())

	n2 = csr.PopulateName(subj, n)
	assert.Equal(t, "SERIALNUMBER=1234,CN=ekspand.com,O=ekspand.com,ST=WA,C=US", n2.String())
}

func TestParsePEM(t *testing.T) {
	pem := `-----BEGIN CERTIFICATE REQUEST-----
MIIBSjCB0QIBADBSMQswCQYDVQQGEwJVUzELMAkGA1UEBxMCV0ExEzARBgNVBAoT
CnRydXN0eS5jb20xITAfBgNVBAMMGFtURVNUXSBUcnVzdHkgTGV2ZWwgMSBDQTB2
MBAGByqGSM49AgEGBSuBBAAiA2IABITXg6XB0tSqS+8gLJ8iPEErcIkiXzA2VFuo
Y/joGvOXaq2GXQyOLXPXDLf0LlTNcQww6McTQUBRjocT7USwhR0EdTS4tfdgQi53
lE9lpMy4V5Gbg9x0t08PQ4EpXM+2KaAAMAoGCCqGSM49BAMDA2gAMGUCMQCut6W1
r6sX2RQbFtUPYEjg2EJdwo8KP0KMzDQEzdh0TzkFaTSxBvMjSR9L2HuntIYCMCuZ
18vhP1NmhNWaLmAPbbukNMhlrDgsezJXzN+/RFv3LCzzOLzHR4V90x6sb2jhmQ==
-----END CERTIFICATE REQUEST-----
`
	crt, err := csr.ParsePEM([]byte(pem))
	require.NoError(t, err)
	assert.Equal(t, "C=US, L=WA, O=trusty.com, CN=[TEST] Trusty Level 1 CA", certutil.NameToString(&crt.Subject))

	pem = `-----BEGIN CERTIFICATE REQUEST-----
	MIICiDCCAXACAQAwQzELMAkGA1UEBhMCVVMxCzAJBgNVBAcTAldBMRMwEQYDVQQK
	Ewp0cnVzdHkuY29tMRIwEAYDVQQDEwlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEB
	/ZgtJhZdT3bQjaXopUfn4faiL1aCYWlLr8BEJQ==
	-----END CERTIFICATE REQUEST-----
	`
	_, err = csr.ParsePEM([]byte(pem))
	require.Error(t, err)
	assert.Equal(t, "unable to parse PEM", err.Error())

	pem = `-----BEGIN CERTIFICATE-----
MIICiDCCAXACAQAwQzELMAkGA1UEBhMCVVMxCzAJBgNVBAcTAldBMRMwEQYDVQQK
Ewp0cnVzdHkuY29tMRIwEAYDVQQDEwlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEB
-----END CERTIFICATE-----
	`
	_, err = csr.ParsePEM([]byte(pem))
	require.Error(t, err)
	assert.Equal(t, "unsupported type in PEM: CERTIFICATE", err.Error())
}

func TestSetSAN(t *testing.T) {
	template := x509.Certificate{}

	csr.SetSAN(&template, []string{
		"ekspand.com",
		"localhost",
		"127.0.0.1",
		"::0",
		"ca@trusty.com",
	})
	assert.Len(t, template.DNSNames, 2)
	assert.Len(t, template.EmailAddresses, 1)
	assert.Len(t, template.IPAddresses, 2)
}
