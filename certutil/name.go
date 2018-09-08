package certutil

import (
	"crypto/x509/pkix"
	"fmt"
	"strings"
)

const certTimeFormat = "Jan _2 15:04:05 2006 GMT"

// NameToString converts Name to string,
// compatable with openssl output
func NameToString(name pkix.Name) string {
	parts := []string{}
	for _, c := range name.Country {
		parts = append(parts, fmt.Sprintf("C=%s", c))
	}
	for _, c := range name.Province {
		parts = append(parts, fmt.Sprintf("ST=%s", c))
	}
	for _, c := range name.Locality {
		parts = append(parts, fmt.Sprintf("L=%s", c))
	}
	for _, c := range name.Organization {
		parts = append(parts, fmt.Sprintf("O=%s", c))
	}
	for _, c := range name.OrganizationalUnit {
		parts = append(parts, fmt.Sprintf("OU=%s", c))
	}
	if name.CommonName != "" {
		parts = append(parts, fmt.Sprintf("CN=%s", name.CommonName))
	}
	if name.SerialNumber != "" {
		parts = append(parts, fmt.Sprintf("SERIALNUMBER=%s", name.SerialNumber))
	}
	return strings.Join(parts, ", ")
}
