package csr

import (
	cfsslconfig "github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/signer"
	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/csrprov"
	"github.com/juju/errors"
)

// SignCertFlags specifies flags for SignCert command
type SignCertFlags struct {
	// CA specifies file name with CA cert
	CA *string
	// CAKey specifies file name with CA key
	CAKey *string
	// CAConfig specifies file name with ca-config
	CAConfig *string
	// Csr specifies file name with pem-encoded CSR
	Csr *string
	// Hostname specifies Host name for generated cert
	Hostname *string
	// Profile specifies the profile name from ca-config
	Profile *string
	// Output specifies the optional prefix for output files,
	// if not set, the output will be printed to STDOUT only
	Output *string
}

func ensureSignCertFlags(f *SignCertFlags) *SignCertFlags {
	var (
		emptyString = ""
	)
	if f.CA == nil {
		f.CA = &emptyString
	}
	if f.CAKey == nil {
		f.CAKey = &emptyString
	}
	if f.CAConfig == nil {
		f.CAConfig = &emptyString
	}
	if f.Csr == nil {
		f.Csr = &emptyString
	}
	if f.Hostname == nil {
		f.Hostname = &emptyString
	}
	if f.Profile == nil {
		f.Profile = &emptyString
	}
	if f.Output == nil {
		f.Output = &emptyString
	}
	return f
}

// SignCert signs a cert
func SignCert(c ctl.Control, p interface{}) error {
	flags := ensureSignCertFlags(p.(*SignCertFlags))

	if *flags.CA == "" || *flags.CAKey == "" {
		return errors.Errorf("CA certificate and key are required")
	}

	// Load CSR
	csrPEM, err := cli.ReadStdin(*flags.Csr)
	if err != nil {
		return errors.Annotate(err, "read CSR")
	}

	// Load ca-config
	cacfg, err := cfsslconfig.LoadFile(*flags.CAConfig)
	if err != nil {
		return errors.Annotate(err, "ca-config")
	}
	if cacfg.Signing == nil {
		return errors.New("missing signing policy in ca-config")
	}
	if !cacfg.Signing.Valid() {
		return errors.New("invalid signing policy in ca-config")
	}

	cryptoprov := c.(*cli.Cli).CryptoProv()
	if cryptoprov == nil {
		return errors.Errorf("unsupported command for this crypto provider")
	}

	s, _, err := csrprov.NewLocalCASignerFromFile(cryptoprov, *flags.CA, *flags.CAKey, cacfg.Signing)
	if err != nil {
		return errors.Annotate(err, "create signer")
	}

	signReq := signer.SignRequest{
		Hosts:   signer.SplitHosts(*flags.Hostname),
		Request: string(csrPEM),
		Profile: *flags.Profile,
	}
	cert, err := s.Sign(signReq)

	if *flags.Output == "" {
		c.(*cli.Cli).PrintCert(nil, nil, cert)
	} else {
		baseName := *flags.Output

		err = cli.WriteFile(baseName+".pem", cert, 0664)
		if err != nil {
			return errors.Trace(err)
		}
		err = cli.WriteFile(baseName+".csr", csrPEM, 0664)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
