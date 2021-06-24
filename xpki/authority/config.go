package authority

import (
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"github.com/go-phorce/dolly/algorithms/slices"
	"github.com/go-phorce/dolly/xpki/csr"
	"github.com/jinzhu/copier"
	"github.com/juju/errors"
	"gopkg.in/yaml.v2"
)

var (
	// DefaultCRLRenewal specifies default duration for CRL renewal
	DefaultCRLRenewal = 7 * 24 * time.Hour // 7 days
	// DefaultCRLExpiry specifies default duration for CRL expiry
	DefaultCRLExpiry = 30 * 24 * time.Hour // 30 days
	// DefaultOCSPExpiry specifies default for OCSP expiry
	DefaultOCSPExpiry = 1 * 24 * time.Hour // 1 day
)

// Config provides configuration for Certification Authority
type Config struct {
	Authority *CAConfig               `json:"authority,omitempty" yaml:"authority,omitempty"`
	Profiles  map[string]*CertProfile `json:"profiles" yaml:"profiles"`
}

// CAConfig contains configuration info for CA
type CAConfig struct {
	// DefaultAIA specifies default AIA configuration
	DefaultAIA *AIAConfig `json:"default_aia,omitempty" yaml:"default_aia,omitempty"`

	// Issuers specifies the list of issuing authorities.
	Issuers []IssuerConfig `json:"issuers,omitempty" yaml:"issuers,omitempty"`

	// PrivateRoots specifies the list of private Root Certs files.
	PrivateRoots []string `json:"private_roots,omitempty" yaml:"private_roots,omitempty"`

	// PublicRoots specifies the list of public Root Certs files.
	PublicRoots []string `json:"public_roots,omitempty" yaml:"public_roots,omitempty"`
}

// IssuerConfig contains configuration info for the issuing certificate
type IssuerConfig struct {
	// Disabled specifies if the certificate disabled to use
	Disabled *bool `json:"disabled,omitempty" yaml:"disabled,omitempty"`

	// Label specifies Issuer's label
	Label string `json:"label,omitempty" yaml:"label,omitempty"`

	// Type specifies type: tls|codesign|timestamp|ocsp|spiffe|trusty
	Type string

	// CertFile specifies location of the cert
	CertFile string `json:"cert,omitempty" yaml:"cert,omitempty"`

	// KeyFile specifies location of the key
	KeyFile string `json:"key,omitempty" yaml:"key,omitempty"`

	// CABundleFile specifies location of the CA bundle file
	CABundleFile string `json:"ca_bundle,omitempty" yaml:"ca_bundle,omitempty"`

	// RootBundleFile specifies location of the Root CA file
	RootBundleFile string `json:"root_bundle,omitempty" yaml:"root_bundle,omitempty"`

	// AIA specifies AIA configuration
	AIA *AIAConfig `json:"aia,omitempty" yaml:"aia,omitempty"`

	// Profiles are populated after loading
	Profiles map[string]*CertProfile `json:"-" yaml:"-"`
}

// AIAConfig contains AIA configuration info
type AIAConfig struct {
	// AiaURL specifies a template for AIA URL.
	// The ${ISSUER_ID} variable will be replaced with a Subject Key Identifier of the issuer.
	AiaURL string `json:"issuer_url" yaml:"issuer_url"`

	// OcspURL specifies a template for OCSP URL.
	// The ${ISSUER_ID} variable will be replaced with a Subject Key Identifier of the issuer.
	OcspURL string `json:"ocsp_url" yaml:"ocsp_url"`

	// DefaultOcspURL specifies a template for CRL URL.
	// The ${ISSUER_ID} variable will be replaced with a Subject Key Identifier of the issuer.
	CrlURL string `json:"crl_url" yaml:"crl_url"`

	// CRLExpiry specifies value in 72h format for duration of CRL next update time
	CRLExpiry time.Duration `json:"crl_expiry,omitempty" yaml:"crl_expiry,omitempty"`

	// OCSPExpiry specifies value in 8h format for duration of OCSP next update time
	OCSPExpiry time.Duration `json:"ocsp_expiry,omitempty" yaml:"ocsp_expiry,omitempty"`

	// CRLRenewal specifies value in 8h format for duration of CRL renewal before next update time
	CRLRenewal time.Duration `json:"crl_renewal,omitempty" yaml:"crl_renewal,omitempty"`
}

// Copy returns new copy
func (c *Config) Copy() *Config {
	d := new(Config)
	copier.Copy(d, c)
	return d
}

// Copy returns new copy
func (c *IssuerConfig) Copy() *IssuerConfig {
	d := new(IssuerConfig)
	copier.Copy(d, c)
	return d
}

// Copy returns new copy
func (c *AIAConfig) Copy() *AIAConfig {
	return &AIAConfig{
		c.AiaURL,
		c.OcspURL,
		c.CrlURL,
		c.CRLExpiry,
		c.OCSPExpiry,
		c.CRLRenewal,
	}
}

// GetDisabled specifies if the certificate disabled to use
func (c *IssuerConfig) GetDisabled() bool {
	return c.Disabled != nil && *c.Disabled
}

// GetCRLExpiry specifies value in 72h format for duration of CRL next update time
func (c *AIAConfig) GetCRLExpiry() time.Duration {
	if c != nil && c.CRLExpiry > 0 {
		return c.CRLExpiry
	}
	return DefaultCRLExpiry
}

// GetOCSPExpiry specifies value in 8h format for duration of OCSP next update time
func (c *AIAConfig) GetOCSPExpiry() time.Duration {
	if c != nil && c.OCSPExpiry > 0 {
		return c.OCSPExpiry
	}
	return DefaultOCSPExpiry
}

// GetCRLRenewal specifies value in 8h format for duration of CRL renewal before next update time
func (c *AIAConfig) GetCRLRenewal() time.Duration {
	if c != nil && c.CRLRenewal > 0 {
		return c.CRLRenewal
	}
	return DefaultCRLRenewal
}

// CertProfile provides certificate profile
type CertProfile struct {
	Description string `json:"description" yaml:"description"`

	// Usage provides a list key usages
	Usage []string `json:"usages" yaml:"usages"`

	CAConstraint CAConstraint `json:"ca_constraint" yaml:"ca_constraint"`
	OCSPNoCheck  bool         `json:"ocsp_no_check" yaml:"ocsp_no_check"`

	Expiry   csr.Duration `json:"expiry" yaml:"expiry"`
	Backdate csr.Duration `json:"backdate" yaml:"backdate"`

	AllowedExtensions []csr.OID `json:"allowed_extensions" yaml:"allowed_extensions"`

	// AllowedNames specifies a RegExp to check for allowed names.
	// If not provided, then all values are allowed
	AllowedNames string `json:"allowed_names" yaml:"allowed_names"`

	// AllowedDNS specifies a RegExp to check for allowed DNS.
	// If not provided, then all values are allowed
	AllowedDNS string `json:"allowed_dns" yaml:"allowed_dns"`

	// AllowedEmail specifies a RegExp to check for allowed email.
	// If not provided, then all values are allowed
	AllowedEmail string `json:"allowed_email" yaml:"allowed_email"`

	// AllowedURI specifies a RegExp to check for allowed URI.
	// If not provided, then all values are allowed
	AllowedURI string `json:"allowed_uri" yaml:"allowed_uri"`

	// AllowedFields provides booleans for fields in the CSR.
	// If a AllowedFields is not present in a CertProfile,
	// all of these fields may be copied from the CSR into the signed certificate.
	// If a AllowedFields *is* present in a CertProfile,
	// only those fields with a `true` value in the AllowedFields may
	// be copied from the CSR to the signed certificate.
	// Note that some of these fields, like Subject, can be provided or
	// partially provided through the API.
	// Since API clients are expected to be trusted, but CSRs are not, fields
	// provided through the API are not subject to validation through this
	// mechanism.
	AllowedCSRFields *csr.AllowedFields `json:"allowed_fields" yaml:"allowed_fields"`

	Policies []csr.CertificatePolicy `json:"policies" yaml:"policies"`

	IssuerLabel  string   `json:"issuer_label" yaml:"issuer_label"`
	AllowedRoles []string `json:"allowed_roles" yaml:"allowed_roles"`
	DeniedRoles  []string `json:"denied_roles" yaml:"denied_roles"`

	AllowedNamesRegex *regexp.Regexp `json:"-" yaml:"-"`
	AllowedDNSRegex   *regexp.Regexp `json:"-" yaml:"-"`
	AllowedEmailRegex *regexp.Regexp `json:"-" yaml:"-"`
	AllowedURIRegex   *regexp.Regexp `json:"-" yaml:"-"`
}

// CAConstraint specifies various CA constraints on the signed certificate.
// CAConstraint would verify against (and override) the CA
// extensions in the given CSR.
type CAConstraint struct {
	IsCA       bool `json:"is_ca" yaml:"is_ca"`
	MaxPathLen int  `json:"max_path_len" yaml:"max_path_len"`
}

// Copy returns new copy
func (p *CertProfile) Copy() *CertProfile {
	d := new(CertProfile)
	copier.Copy(d, p)
	return d
}

// AllowedExtensionsStrings returns slice of strings
func (p *CertProfile) AllowedExtensionsStrings() []string {
	list := make([]string, len(p.AllowedExtensions))
	for i, o := range p.AllowedExtensions {
		list[i] = o.String()
	}
	return list
}

// IsAllowed returns true, if a role is allowed to request this profile
func (p *CertProfile) IsAllowed(role string) bool {
	if len(p.DeniedRoles) > 0 && (slices.ContainsString(p.DeniedRoles, role) || slices.ContainsString(p.DeniedRoles, "*")) {
		return false
	}
	if len(p.AllowedRoles) > 0 && (slices.ContainsString(p.AllowedRoles, role) || slices.ContainsString(p.AllowedRoles, "*")) {
		return true
	}
	return true
}

// DefaultCertProfile returns a default configuration
// for a certificate profile, specifying basic key
// usage and a 1 year expiration time.
// The key usages chosen are:
//   signing, key encipherment, client auth and server auth.
func DefaultCertProfile() *CertProfile {
	return &CertProfile{
		Description: "default profile with Server and Client auth",
		Usage:       []string{"signing", "key encipherment", "server auth", "client auth"},
		Expiry:      csr.Duration(8760 * time.Hour),
		Backdate:    csr.Duration(10 * time.Minute),
	}
}

// LoadConfig loads the configuration file stored at the path
// and returns the configuration.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("invalid path")
	}

	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Annotate(err, "unable to read configuration file")
	}

	var cfg = new(Config)
	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(body, cfg)
	} else {
		err = yaml.Unmarshal(body, cfg)
	}

	if err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal configuration")
	}

	if len(cfg.Profiles) == 0 {
		return nil, errors.New("no \"profiles\" configuration present")
	}

	if cfg.Profiles["default"] == nil {
		logger.Infof("reason=no_default_profile")
		cfg.Profiles["default"] = DefaultCertProfile()
	}

	if cfg.Authority != nil && cfg.Authority.DefaultAIA != nil {
		for i := range cfg.Authority.Issuers {
			iss := &cfg.Authority.Issuers[i]
			if iss.AIA == nil {
				iss.AIA = cfg.Authority.DefaultAIA.Copy()
			} else {
				if iss.AIA.AiaURL == "" {
					iss.AIA.AiaURL = cfg.Authority.DefaultAIA.AiaURL
				}
				if iss.AIA.CrlURL == "" {
					iss.AIA.CrlURL = cfg.Authority.DefaultAIA.CrlURL
				}
				if iss.AIA.OcspURL == "" {
					iss.AIA.OcspURL = cfg.Authority.DefaultAIA.OcspURL
				}
				if iss.AIA.CRLExpiry == 0 {
					iss.AIA.CRLExpiry = cfg.Authority.DefaultAIA.GetCRLExpiry()
				}
				if iss.AIA.CRLRenewal == 0 {
					iss.AIA.CRLRenewal = cfg.Authority.DefaultAIA.GetCRLRenewal()
				}
				if iss.AIA.OCSPExpiry == 0 {
					iss.AIA.OCSPExpiry = cfg.Authority.DefaultAIA.GetOCSPExpiry()
				}
			}

			iss.Profiles = make(map[string]*CertProfile)
			for name, profile := range cfg.Profiles {
				if profile.IssuerLabel == iss.Label ||
					(profile.IssuerLabel == "" && len(cfg.Authority.Issuers) == 1) {
					iss.Profiles[name] = profile
				}
			}
		}
	}

	if err = cfg.Validate(); err != nil {
		return nil, errors.Annotate(err, "invalid configuration")
	}

	return cfg, nil
}

// DefaultCertProfile returns default CertProfile
func (c *Config) DefaultCertProfile() *CertProfile {
	return c.Profiles["default"]
}

// Validate returns an error if the profile is invalid
func (p *CertProfile) Validate() error {
	if p.Expiry == 0 {
		return errors.New("no expiry set")
	}

	if len(p.Usage) == 0 {
		return errors.New("no usages specified")
	} else if _, _, unk := p.Usages(); len(unk) > 0 {
		return errors.Errorf("unknown usage: %s", strings.Join(unk, ","))
	}

	for _, policy := range p.Policies {
		for _, qualifier := range policy.Qualifiers {
			if qualifier.Type != "" &&
				qualifier.Type != csr.UserNoticeQualifierType &&
				qualifier.Type != csr.CpsQualifierType {
				return errors.New("invalid policy qualifier type: " + qualifier.Type)
			}
		}
	}

	if p.AllowedNames != "" && p.AllowedNamesRegex == nil {
		rule, err := regexp.Compile(p.AllowedNames)
		if err != nil {
			return errors.Annotate(err, "failed to compile AllowedNames")
		}
		p.AllowedNamesRegex = rule
	}
	if p.AllowedDNS != "" && p.AllowedDNSRegex == nil {
		rule, err := regexp.Compile(p.AllowedDNS)
		if err != nil {
			return errors.Annotate(err, "failed to compile AllowedDNS")
		}
		p.AllowedDNSRegex = rule
	}
	if p.AllowedEmail != "" && p.AllowedEmailRegex == nil {
		rule, err := regexp.Compile(p.AllowedEmail)
		if err != nil {
			return errors.Annotate(err, "failed to compile AllowedEmail")
		}
		p.AllowedEmailRegex = rule
	}
	if p.AllowedURI != "" && p.AllowedURIRegex == nil {
		rule, err := regexp.Compile(p.AllowedURI)
		if err != nil {
			return errors.Annotate(err, "failed to compile AllowedURI")
		}
		p.AllowedURIRegex = rule
	}

	return nil
}

// IsAllowedExtention returns true of the extension is allowed
func (p *CertProfile) IsAllowedExtention(oid csr.OID) bool {
	for _, allowed := range p.AllowedExtensions {
		if allowed.Equal(oid) {
			return true
		}
	}
	return false
}

// Validate returns an error if the configuration is invalid
func (c *Config) Validate() error {
	var err error

	issuers := map[string]bool{}
	if c.Authority != nil {
		for i := range c.Authority.Issuers {
			iss := &c.Authority.Issuers[i]
			issuers[iss.Label] = true
		}
	}

	for name, profile := range c.Profiles {
		err = profile.Validate()
		if err != nil {
			return errors.Annotatef(err, "invalid %s profile", name)
		}
		if profile.IssuerLabel != "" {
			if !issuers[profile.IssuerLabel] {
				return errors.Annotatef(err, "%s issuer not found for %s profile", profile.IssuerLabel, name)
			}
		}
	}

	return nil
}

// Usages parses the list of key uses in the profile, translating them
// to a list of X.509 key usages and extended key usages.
// The unknown uses are collected into a slice that is also returned.
func (p *CertProfile) Usages() (ku x509.KeyUsage, eku []x509.ExtKeyUsage, unk []string) {
	for _, keyUse := range p.Usage {
		if kuse, ok := csr.KeyUsage[keyUse]; ok {
			ku |= kuse
		} else if ekuse, ok := csr.ExtKeyUsage[keyUse]; ok {
			eku = append(eku, ekuse)
		} else {
			unk = append(unk, keyUse)
		}
	}
	return
}
