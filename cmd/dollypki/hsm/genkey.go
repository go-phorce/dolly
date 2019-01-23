package hsm

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-phorce/dolly/fileutil"

	cfsslcli "github.com/cloudflare/cfssl/cli"
	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/csrprov"
	"github.com/juju/errors"
)

// GenKeyFlags specifies flags for GenKey command
type GenKeyFlags struct {
	// Algo specifies algorithm
	Algo *string
	// Size specifies key size in bits
	Size *int
	// Purpose
	Purpose *string
	// Label specifies name for generated key
	Label *string
	// Output specifies the prefix for generated key
	// if not set, the output will be printed to STDOUT only
	Output *string
	// Force to override key file if exists
	Force *bool
	// Check if file exists, and exit without error
	Check *bool
}

func ensureGenKeyFlags(f *GenKeyFlags) *GenKeyFlags {
	var (
		emptyString = ""
		intVal      = 0
		falseVal    = false
	)
	if f.Size == nil {
		f.Size = &intVal
	}
	if f.Algo == nil {
		f.Algo = &emptyString
	}
	if f.Label == nil {
		f.Label = &emptyString
	}
	if f.Purpose == nil {
		f.Purpose = &emptyString
	}
	if f.Output == nil {
		f.Output = &emptyString
	}
	if f.Force == nil {
		f.Force = &falseVal
	}
	if f.Check == nil {
		f.Check = &falseVal
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

	if *flags.Check && *flags.Output != "" && fileutil.FileExists(*flags.Output) == nil {
		c.Printf("%q file exists, specify --force flag to override\n", *flags.Output)
		return nil
	}

	if !*flags.Force && *flags.Output != "" && fileutil.FileExists(*flags.Output) == nil {
		return errors.Errorf("%q file exists, specify --force flag to override", *flags.Output)
	}

	crypto := cryptoprov.Default()
	csr := csrprov.New(cryptoprov.Default())

	purpose := csrprov.Signing
	switch *flags.Purpose {
	case "s", "sign", "signing":
		purpose = csrprov.Signing
	case "e", "encrypt", "encryption":
		purpose = csrprov.Encryption
	default:
		return errors.Errorf("unsupported purpose: %q", *flags.Purpose)
	}

	req := csr.NewKeyRequest(prefixKeyLabel(*flags.Label), *flags.Algo, *flags.Size, purpose)
	prv, err := req.Generate()
	if err != nil {
		return errors.Trace(err)
	}

	keyID, _, err := crypto.IdentifyKey(prv)
	if err != nil {
		return errors.Trace(err)
	}

	uri, key, err := crypto.ExportKey(keyID)
	if err != nil {
		return errors.Trace(err)
	}

	if key == nil {
		key = []byte(uri)
	}

	if *flags.Output == "" {
		cfsslcli.PrintCert(key, nil, nil)
	} else {
		err = cli.WriteFile(*flags.Output, key, 0600)
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
