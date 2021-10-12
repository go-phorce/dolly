package csr

import (
	"encoding/json"

	cfsslcli "github.com/cloudflare/cfssl/cli"
	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/csrprov"
	"github.com/pkg/errors"
)

// GenKeyFlags specifies flags for GenKey command
type GenKeyFlags struct {
	// Initca specifies if it's a request for self-signed CA cert
	Initca *bool
	// CsrProfile specifies file name with CSR profile
	CsrProfile *string
	// Label specifies name for generated key
	Label *string
	// Output specifies the optional prefix for output files,
	// if not set, the output will be printed to STDOUT only
	Output *string
}

func ensureGenKeyFlags(f *GenKeyFlags) *GenKeyFlags {
	var (
		emptyString = ""
		falseVal    = false
	)
	if f.Initca == nil {
		f.Initca = &falseVal
	}
	if f.CsrProfile == nil {
		f.CsrProfile = &emptyString
	}
	if f.Label == nil {
		f.Label = &emptyString
	}
	if f.Output == nil {
		f.Output = &emptyString
	}
	return f
}

// GenKey generates a key
func GenKey(c ctl.Control, p interface{}) error {
	flags := ensureGenKeyFlags(p.(*GenKeyFlags))

	cryptoprov := c.(*cli.Cli).CryptoProv()
	if cryptoprov == nil {
		return errors.Errorf("unsupported command for this crypto provider")
	}

	prov := csrprov.New(cryptoprov.Default())

	csrf, err := cli.ReadStdin(*flags.CsrProfile)
	if err != nil {
		return errors.WithMessage(err, "read CSR profile")
	}

	req := csrprov.CertificateRequest{
		// TODO: alg and size from params
		KeyRequest: prov.NewKeyRequest(prefixKeyLabel(*flags.Label), "ECDSA", 256, csrprov.Signing),
	}

	err = json.Unmarshal(csrf, &req)
	if err != nil {
		return errors.WithMessage(err, "invalid CSR")
	}

	if *flags.Initca {
		var key, csrPEM, cert []byte
		cert, csrPEM, key, err = prov.NewRoot(&req)
		if err != nil {
			return errors.WithMessage(err, "init CA")
		}

		if *flags.Output == "" {
			cfsslcli.PrintCert(key, csrPEM, cert)
		} else {
			baseName := *flags.Output

			err = cli.WriteFile(baseName+".pem", cert, 0664)
			if err != nil {
				return errors.WithStack(err)
			}
			err = cli.WriteFile(baseName+".csr", csrPEM, 0664)
			if err != nil {
				return errors.WithStack(err)
			}
			err = cli.WriteFile(baseName+"-key.pem", key, 0600)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	} else {
		if req.CA != nil {
			return errors.New("CA section only permitted with --initca option")
		}

		var key, csrPEM []byte
		csrPEM, key, _, _, err = prov.ProcessCsrRequest(&req)
		if err != nil {
			key = nil
			return errors.WithMessage(err, "ProcessCsrRequest")
		}

		if *flags.Output == "" {
			c.(*cli.Cli).PrintCert(key, csrPEM, nil)
		} else {
			baseName := *flags.Output

			err = cli.WriteFile(baseName+".csr", csrPEM, 0664)
			if err != nil {
				return errors.WithStack(err)
			}
			err = cli.WriteFile(baseName+"-key.pem", key, 0600)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
