package authority_test

import (
	"crypto"
	"fmt"

	"github.com/go-phorce/dolly/xpki/authority"
)

func (s *testSuite) TestNewIssuer() {

	cfg, err := authority.LoadConfig("./testdata/ca-config.dev.yaml")
	s.Require().NoError(err)
	s.Require().NotNil(cfg.Authority)
	s.NotNil(cfg.Authority.DefaultAIA)

	for _, isscfg := range cfg.Authority.Issuers {
		if isscfg.GetDisabled() {
			continue
		}

		issuer, err := authority.NewIssuer(&isscfg, s.crypto)
		s.Require().NoError(err)

		s.NotNil(issuer.Bundle())
		s.NotNil(issuer.Signer())
		s.NotEmpty(issuer.PEM())
		s.NotEmpty(issuer.OcspURL())
		s.NotEmpty(issuer.Label())
		s.NotEmpty(issuer.KeyHash(crypto.SHA1))
		s.NotNil(issuer.Profile("client"))
		s.Nil(issuer.Profile("notfound"))

		s.Equal(fmt.Sprintf("http://localhost:7880/v1/crl/%s.crl", issuer.SubjectKID()), issuer.CrlURL())
		s.Equal(fmt.Sprintf("http://localhost:7880/v1/certs/%s.crt", issuer.SubjectKID()), issuer.AiaURL())
		//s.NotNil(issuer.AIAExtension("server"))
		//s.Nil(issuer.AIAExtension("not_supported"))
	}
}

func (s *testSuite) TestNewIssuerErrors() {

	aia := &authority.AIAConfig{
		AiaURL:  "https://localhost/v1/certs/${ISSUER_ID}.crt",
		OcspURL: "https://localhost/v1/ocsp",
		CrlURL:  "https://localhost/v1/crl/${ISSUER_ID}.crl",
	}
	cfg := &authority.IssuerConfig{
		KeyFile: "not_found",
		AIA:     aia,
	}
	_, err := authority.NewIssuer(cfg, s.crypto)
	s.Require().Error(err)
	s.Equal("unable to create signer: load key file: open not_found: no such file or directory", err.Error())

	cfg = &authority.IssuerConfig{
		KeyFile:  ca2KeyFile,
		CertFile: "not_found",
	}
	_, err = authority.NewIssuer(cfg, s.crypto)
	s.Require().Error(err)
	s.Equal("failed to load cert: open not_found: no such file or directory", err.Error())

	cfg = &authority.IssuerConfig{
		CertFile:       ca2CertFile,
		KeyFile:        ca2KeyFile,
		CABundleFile:   caBundleFile,
		RootBundleFile: "not_found",
	}
	_, err = authority.NewIssuer(cfg, s.crypto)
	s.Require().Error(err)
	s.Equal("failed to load root-bundle: open not_found: no such file or directory", err.Error())

	cfg = &authority.IssuerConfig{
		CertFile:       ca2CertFile,
		KeyFile:        ca2KeyFile,
		CABundleFile:   "not_found",
		RootBundleFile: rootBundleFile,
	}
	_, err = authority.NewIssuer(cfg, s.crypto)
	s.Require().Error(err)
	s.Equal("failed to load ca-bundle: open not_found: no such file or directory", err.Error())
}
