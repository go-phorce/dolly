package csr

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	cfsslconfig "github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/signer"
	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/csrprov"
	"github.com/juju/errors"
)

// GenCertFlags specifies flags for GenCert command
type GenCertFlags struct {
	// CA specifies file name with CA cert
	CA *string
	// CAKey specifies file name with CA key
	CAKey *string
	// CAConfig specifies file name with ca-config
	CAConfig *string
	// CsrProfile specifies file name with CSR profile
	CsrProfile *string
	// Label specifies name for generated key
	Label *string
	// Hostname specifies Host name for generated cert
	Hostname *string
	// Profile specifies the profile name from ca-config
	Profile *string
	// Output specifies the optional prefix for output files,
	// if not set, the output will be printed to STDOUT only
	Output *string
}

func ensureGenCertFlags(f *GenCertFlags) *GenCertFlags {
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
	if f.CsrProfile == nil {
		f.CsrProfile = &emptyString
	}
	if f.Label == nil {
		f.Label = &emptyString
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

// GenCert generates a cert
func GenCert(c ctl.Control, p interface{}) error {
	flags := ensureGenCertFlags(p.(*GenCertFlags))

	cryptoprov := c.(*cli.Cli).CryptoProv()
	if cryptoprov == nil {
		return errors.Errorf("unsupported command for this crypto provider")
	}

	prov := csrprov.New(cryptoprov.Default())

	if *flags.CA == "" || *flags.CAKey == "" {
		return errors.Errorf("CA certificate and key are required")
	}

	// Load CSR
	csrf, err := cli.ReadStdin(*flags.CsrProfile)
	if err != nil {
		return errors.Annotate(err, "read CSR profile")
	}

	req := csrprov.CertificateRequest{
		// TODO: alg and size from params
		KeyRequest: prov.NewKeyRequest(prefixKeyLabel(*flags.Label), "ECDSA", 256, csrprov.Signing),
	}

	err = json.Unmarshal(csrf, &req)
	if err != nil {
		return errors.Annotate(err, "invalid CSR")
	}

	if req.CA != nil {
		return errors.New("CA section only permitted with --initca option, use genkey comand instead")
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

	// ensure that signer can be created before the key is generated
	s, _, err := csrprov.NewLocalCASignerFromFile(cryptoprov, *flags.CA, *flags.CAKey, cacfg.Signing)
	if err != nil {
		return errors.Annotate(err, "create signer")
	}

	var key, csrPEM []byte
	csrPEM, key, _, _, err = prov.ProcessCsrRequest(&req)
	if err != nil {
		return errors.Annotate(err, "ProcessRequest")
	}

	signReq := signer.SignRequest{
		Hosts:   signer.SplitHosts(*flags.Hostname),
		Request: string(csrPEM),
		Profile: *flags.Profile,
	}
	cert, err := s.Sign(signReq)

	if *flags.Output == "" {
		c.(*cli.Cli).PrintCert(key, csrPEM, cert)
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
		err = cli.WriteFile(baseName+"-key.pem", key, 0600)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// prefixKeyLabel adds a date prefix to label for a key
func prefixKeyLabel(label string) string {
	if strings.HasSuffix(label, "*") {
		g := guid.MustCreate()
		t := time.Now().UTC()
		label = strings.TrimSuffix(label, "*") +
			fmt.Sprintf("_%04d%02d%02d%02d%02d%02d_%x", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), g[:4])
	}

	return label
}
