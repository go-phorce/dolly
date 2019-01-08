package csrprov

import (
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"io/ioutil"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/initca"
	"github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/juju/errors"
)

// A CertificateRequest encapsulates the API interface to the
// certificate request functionality.
type CertificateRequest struct {
	CN           string
	Names        []X509Name `json:"names" yaml:"names"`
	Hosts        []string   `json:"hosts" yaml:"hosts"`
	KeyRequest   KeyRequest `json:"key,omitempty" yaml:"key,omitempty"`
	CA           *CAConfig  `json:"ca,omitempty" yaml:"ca,omitempty"`
	SerialNumber string     `json:"serialnumber,omitempty" yaml:"serialnumber,omitempty"`
}

// X509Name contains the SubjectInfo fields.
type X509Name struct {
	C            string // Country
	ST           string // State
	L            string // Locality
	O            string // OrganisationName
	OU           string // OrganisationalUnitName
	SerialNumber string
}

// Name returns the PKIX name for the request.
func (cr *CertificateRequest) Name() pkix.Name {
	var name pkix.Name
	name.CommonName = cr.CN

	for _, n := range cr.Names {
		appendIf(n.C, &name.Country)
		appendIf(n.ST, &name.Province)
		appendIf(n.L, &name.Locality)
		appendIf(n.O, &name.Organization)
		appendIf(n.OU, &name.OrganizationalUnit)
	}
	name.SerialNumber = cr.SerialNumber
	return name
}

// appendIf appends to a if s is not an empty string.
func appendIf(s string, a *[]string) {
	if s != "" {
		*a = append(*a, s)
	}
}

// ConvertToCFSSL converts to CFSSL type
func (c *X509Name) ConvertToCFSSL() *csr.Name {
	return &csr.Name{
		C:  c.C,
		ST: c.ST,
		L:  c.L,
		O:  c.O,
		OU: c.OU,
	}
}

// CopyToCFSSL copies to CFSSL type
func (c *X509Name) CopyToCFSSL(d *csr.Name) {
	d.C = c.C
	d.ST = c.ST
	d.L = c.L
	d.O = c.O
	d.OU = c.OU
}

// CAConfig is a section used in the requests initialising a new CA.
type CAConfig struct {
	PathLength  int    `json:"pathlen" yaml:"pathlen"`
	PathLenZero bool   `json:"pathlenzero" yaml:"pathlenzero"`
	Expiry      string `json:"expiry" yaml:"expiry"`
	Backdate    string `json:"backdate" yaml:"backdate"`
}

// ConvertToCFSSL converts to CFSSL type
func (c *CAConfig) ConvertToCFSSL() *csr.CAConfig {
	return &csr.CAConfig{
		PathLength:  c.PathLength,
		PathLenZero: c.PathLenZero,
		Expiry:      c.Expiry,
		Backdate:    c.Backdate,
	}
}

// CopyToCFSSL copies to CFSSL type
func (c *CAConfig) CopyToCFSSL(d *csr.CAConfig) {
	d.PathLength = c.PathLength
	d.PathLenZero = c.PathLenZero
	d.Expiry = c.Expiry
	d.Backdate = c.Backdate
}

// ValidateCSR contains the default validation logic for certificate
// authority certificates. The only requirement here is that the
// certificate have a non-empty subject field.
func ValidateCSR(req *CertificateRequest) error {
	if req.CN != "" {
		return nil
	}

	if len(req.Names) == 0 {
		return errors.New("missing subject information")
	}

	for _, n := range req.Names {
		if isNameEmpty(n) {
			return errors.New("empty name")
		}
	}

	return nil
}

// isNameEmpty returns true if the name has no identifying information in it.
func isNameEmpty(n X509Name) bool {
	empty := func(s string) bool { return strings.TrimSpace(s) == "" }

	if empty(n.C) && empty(n.ST) && empty(n.L) && empty(n.O) && empty(n.OU) {
		return true
	}
	return false
}

// MakeCAPolicy make CA policy from the given certificate request
func MakeCAPolicy(req *CertificateRequest) (*config.Signing, error) {
	var err error
	policy := initca.CAPolicy()
	if req.CA == nil {
		return policy, nil
	}

	if req.CA.Expiry != "" {
		policy.Default.ExpiryString = req.CA.Expiry
		policy.Default.Expiry, err = time.ParseDuration(req.CA.Expiry)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	if req.CA.Backdate != "" {
		policy.Default.Backdate, err = time.ParseDuration(req.CA.Backdate)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	policy.Default.CAConstraint.MaxPathLen = req.CA.PathLength
	if req.CA.PathLength != 0 && req.CA.PathLenZero {
		logger.Infof("ignore invalid 'pathlenzero' value")
	} else {
		policy.Default.CAConstraint.MaxPathLenZero = req.CA.PathLenZero
	}
	return policy, nil
}

// ParseCaFiles parses CA files
func ParseCaFiles(caFile, caKeyFile string) (cakey []byte, parsedCa *x509.Certificate, err error) {
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		err = errors.Annotatef(err, "load ca file")
		return
	}

	cakey, err = ioutil.ReadFile(caKeyFile)
	if err != nil {
		err = errors.Annotatef(err, "load ca-key file")
		return
	}

	parsedCa, err = helpers.ParseCertificatePEM(ca)
	if err != nil {
		err = errors.Annotatef(err, "parse ca file")
		return
	}

	return
}

// NewLocalCASignerFromFile generates a new local signer from a caFile
// and a caKey file, both PEM encoded or caKey contains PKCS#11 Uri
func NewLocalCASignerFromFile(c *cryptoprov.Crypto, caFile, caKeyFile string, policy *config.Signing) (*local.Signer, crypto.Signer, error) {
	ca, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "load ca file")
	}
	cakey, err := ioutil.ReadFile(caKeyFile)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "load ca-key file")
	}

	return NewLocalCASignerFromPEM(c, ca, cakey, policy)
}

// NewLocalCASignerFromPEM generates a new local signer from PEM encoded blocks,
// or caKey contains PKCS#11 Uri
func NewLocalCASignerFromPEM(c *cryptoprov.Crypto, ca, caKey []byte, policy *config.Signing) (*local.Signer, crypto.Signer, error) {
	if policy == nil {
		return nil, nil, errors.New("invalid parameter: policy")
	}

	_, pvk, err := c.LoadPrivateKey(caKey)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	sign, supported := pvk.(crypto.Signer)
	if !supported {
		return nil, nil, errors.Errorf("loaded key of %T type does not support crypto.Signer", pvk)
	}

	parsedCa, err := helpers.ParseCertificatePEM(ca)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	signPolicy := config.Signing(*policy)
	localSigner, err := local.NewSigner(sign, parsedCa, signer.DefaultSigAlgo(sign), &signPolicy)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	return localSigner, sign, nil
}
