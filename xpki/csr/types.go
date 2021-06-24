package csr

import (
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/juju/errors"
)

const (
	// UserNoticeQualifierType defines id-qt-unotice
	UserNoticeQualifierType = "id-qt-unotice"
	// CpsQualifierType defines id-qt-cps
	CpsQualifierType = "id-qt-cps"

	// OneYear duration
	OneYear = Duration(8760 * time.Hour)
)

// BasicConstraintsOID specifies OID for BasicConstraints
var BasicConstraintsOID = asn1.ObjectIdentifier{2, 5, 29, 19}

// BasicConstraints CSR information RFC 5280, 4.2.1.9
type BasicConstraints struct {
	IsCA       bool `asn1:"optional"`
	MaxPathLen int  `asn1:"optional,default:-1"`
}

// KeyUsage contains a mapping of string names to key usages.
var KeyUsage = map[string]x509.KeyUsage{
	"signing":            x509.KeyUsageDigitalSignature,
	"digital signature":  x509.KeyUsageDigitalSignature,
	"content commitment": x509.KeyUsageContentCommitment,
	"key encipherment":   x509.KeyUsageKeyEncipherment,
	"key agreement":      x509.KeyUsageKeyAgreement,
	"data encipherment":  x509.KeyUsageDataEncipherment,
	"cert sign":          x509.KeyUsageCertSign,
	"crl sign":           x509.KeyUsageCRLSign,
	"encipher only":      x509.KeyUsageEncipherOnly,
	"decipher only":      x509.KeyUsageDecipherOnly,
}

// ExtKeyUsage contains a mapping of string names to extended key
// usages.
var ExtKeyUsage = map[string]x509.ExtKeyUsage{
	"any":              x509.ExtKeyUsageAny,
	"server auth":      x509.ExtKeyUsageServerAuth,
	"client auth":      x509.ExtKeyUsageClientAuth,
	"code signing":     x509.ExtKeyUsageCodeSigning,
	"email protection": x509.ExtKeyUsageEmailProtection,
	"s/mime":           x509.ExtKeyUsageEmailProtection,
	"ipsec end system": x509.ExtKeyUsageIPSECEndSystem,
	"ipsec tunnel":     x509.ExtKeyUsageIPSECTunnel,
	"ipsec user":       x509.ExtKeyUsageIPSECUser,
	"timestamping":     x509.ExtKeyUsageTimeStamping,
	"ocsp signing":     x509.ExtKeyUsageOCSPSigning,
	"microsoft sgc":    x509.ExtKeyUsageMicrosoftServerGatedCrypto,
	"netscape sgc":     x509.ExtKeyUsageNetscapeServerGatedCrypto,
}

// OID is the asn1's ObjectIdentifier, provide a custom
// JSON marshal / unmarshal.
type OID asn1.ObjectIdentifier

// Equal reports whether oi and other represent the same identifier.
func (oid OID) Equal(other OID) bool {
	return asn1.ObjectIdentifier(oid).Equal(asn1.ObjectIdentifier(other))
}

func (oid OID) String() string {
	return asn1.ObjectIdentifier(oid).String()
}

// UnmarshalJSON unmarshals a JSON string into an OID.
func (oid *OID) UnmarshalJSON(data []byte) (err error) {
	last := len(data) - 1
	if data[0] != '"' || data[last] != '"' {
		return errors.New("OID JSON string not wrapped in quotes: " + string(data))
	}
	parsedOid, err := parseObjectIdentifier(string(data[1:last]))
	if err != nil {
		return err
	}
	*oid = OID(parsedOid)
	return
}

// UnmarshalYAML unmarshals a YAML string into an OID.
func (oid *OID) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	err := unmarshal(&buf)
	if err != nil {
		return err
	}

	parsedOid, err := parseObjectIdentifier(buf)
	if err != nil {
		return err
	}
	*oid = OID(parsedOid)
	return err
}

// MarshalJSON marshals an oid into a JSON string.
func (oid OID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%v"`, asn1.ObjectIdentifier(oid))), nil
}

func parseObjectIdentifier(oidString string) (oid asn1.ObjectIdentifier, err error) {
	validOID, err := regexp.MatchString("\\d(\\.\\d+)*", oidString)
	if err != nil {
		return
	}
	if !validOID {
		err = errors.Errorf("invalid OID: %q", oidString)
		return
	}

	segments := strings.Split(oidString, ".")
	oid = make(asn1.ObjectIdentifier, len(segments))
	for i, intString := range segments {
		oid[i], err = strconv.Atoi(intString)
		if err != nil {
			err = errors.Annotatef(err, "invalid OID")
			return
		}
	}
	return
}

// Duration represents a period of time, its the same as time.Duration
// but supports better marshalling from json
type Duration time.Duration

// UnmarshalJSON handles decoding our custom json serialization for Durations
// json values that are numbers are treated as seconds
// json values that are strings, can use the standard time.Duration units indicators
// e.g. this can decode val:100 as well as val:"10m"
func (d *Duration) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		dir, err := time.ParseDuration(string(b[1 : len(b)-1]))
		*d = Duration(dir)
		return err
	}
	i, err := json.Number(string(b)).Int64()
	*d = Duration(time.Duration(i) * time.Second)
	return err
}

// UnmarshalYAML handles decoding our custom json serialization for Durations
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	err := unmarshal(&buf)
	if err != nil {
		return err
	}

	dir, err := time.ParseDuration(buf)
	*d = Duration(dir)
	return err
}

// MarshalJSON encodes our custom Duration value as a quoted version of its underlying value's String() output
// this means you get a duration with a trailing units indicator, e.g. "10m0s"
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

// String returns a string formatted version of the duration in a valueUnits format, e.g. 5m0s for 5 minutes
func (d Duration) String() string {
	return time.Duration(d).String()
}

// TimeDuration returns this duration in a time.Duration type
func (d Duration) TimeDuration() time.Duration {
	return time.Duration(d)
}
