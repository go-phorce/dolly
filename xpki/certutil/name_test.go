package certutil_test

import (
	"crypto/x509/pkix"
	"testing"

	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/stretchr/testify/assert"
)

func Test_NameToString(t *testing.T) {
	tcases := []struct {
		expected string
		name     pkix.Name
	}{
		{
			expected: "C=USA, C=Canada, ST=Prov2, L=Loc1, O=Ekspand, OU=go-phorce, CN=cn1, SERIALNUMBER=123",
			name: pkix.Name{
				Country:            []string{"USA", "Canada"},
				Organization:       []string{"Ekspand"},
				OrganizationalUnit: []string{"go-phorce"},
				Locality:           []string{"Loc1"},
				Province:           []string{"Prov2"},
				StreetAddress:      []string{"str addr"},
				PostalCode:         []string{"98007"},
				CommonName:         "cn1",
				SerialNumber:       "123",
			},
		},
		{
			expected: "O=Ekspand, OU=go-phorce, CN=cn1",
			name: pkix.Name{
				Organization:       []string{"Ekspand"},
				OrganizationalUnit: []string{"go-phorce"},
				CommonName:         "cn1",
			},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, certutil.NameToString(&tc.name))
		})
	}
}
