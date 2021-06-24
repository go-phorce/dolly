package authority

import (
	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xpki", "authority")

// Authority defines the CA
type Authority struct {
	issuers          map[string]*Issuer // label => Issuer
	issuersByProfile map[string]*Issuer // cert profile => Issuer

	// Crypto holds providers for HSM, SoftHSM, KMS, etc.
	crypto *cryptoprov.Crypto
}

// NewAuthority returns new instance of Authority
func NewAuthority(cfg *Config, crypto *cryptoprov.Crypto) (*Authority, error) {
	if cfg.Authority == nil {
		return nil, errors.New("missing Authority configuration")
	}

	ca := &Authority{
		crypto:           crypto,
		issuers:          make(map[string]*Issuer),
		issuersByProfile: make(map[string]*Issuer),
	}

	ocspNextUpdate := cfg.Authority.DefaultAIA.GetOCSPExpiry()
	crlNextUpdate := cfg.Authority.DefaultAIA.GetCRLExpiry()
	crlRenewal := cfg.Authority.DefaultAIA.GetCRLRenewal()

	for _, isscfg := range cfg.Authority.Issuers {
		if isscfg.GetDisabled() {
			logger.Infof("reason=disabled, issuer=%s", isscfg.Label)
			continue
		}

		ccfg := isscfg.Copy()
		if ccfg.AIA == nil {
			ccfg.AIA = cfg.Authority.DefaultAIA.Copy()
		}
		if ccfg.AIA.CRLRenewal == 0 {
			ccfg.AIA.CRLRenewal = crlRenewal
		}
		if ccfg.AIA.CRLExpiry == 0 {
			ccfg.AIA.CRLExpiry = crlNextUpdate
		}
		if ccfg.AIA.OCSPExpiry == 0 {
			ccfg.AIA.OCSPExpiry = ocspNextUpdate
		}
		if ccfg.AIA.CrlURL == "" {
			ccfg.AIA.CrlURL = cfg.Authority.DefaultAIA.CrlURL
		}
		if ccfg.AIA.OcspURL == "" {
			ccfg.AIA.OcspURL = cfg.Authority.DefaultAIA.OcspURL
		}
		if ccfg.AIA.AiaURL == "" {
			ccfg.AIA.AiaURL = cfg.Authority.DefaultAIA.AiaURL
		}
		issuer, err := NewIssuer(ccfg, crypto)
		if err != nil {
			return nil, errors.Annotatef(err, "unable to create issuer: %q", isscfg.Label)
		}

		ca.issuers[isscfg.Label] = issuer

		for profileName := range isscfg.Profiles {
			/*
				if is := ca.issuersByProfile[profileName]; is != nil {
					return nil, errors.Errorf("profile %q is already registered by %q issuer", profileName, is.Label())
				}
			*/
			ca.issuersByProfile[profileName] = issuer
		}
	}

	return ca, nil
}

// GetIssuerByLabel by label
func (s *Authority) GetIssuerByLabel(label string) (*Issuer, error) {
	issuer, ok := s.issuers[label]
	if ok {
		return issuer, nil
	}
	return nil, errors.Errorf("issuer not found: %s", label)
}

// GetIssuerByProfile by profile
func (s *Authority) GetIssuerByProfile(profile string) (*Issuer, error) {
	issuer, ok := s.issuersByProfile[profile]
	if ok {
		return issuer, nil
	}
	return nil, errors.Errorf("issuer not found for profile: %s", profile)
}

// Issuers returns a list of issuers
func (s *Authority) Issuers() []*Issuer {
	list := make([]*Issuer, 0, len(s.issuers))
	for _, ca := range s.issuers {
		list = append(list, ca)
	}

	return list
}
