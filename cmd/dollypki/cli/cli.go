// Package cli provides common code for building a command line control for the service
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/pkg/errors"
)

// ReturnCode is the type that your command returns, these map to standard process return codes
type ReturnCode ctl.ReturnCode

// ReadFileOrStdinFn allows to read from file or Stdin if the name is "-"
type ReadFileOrStdinFn func(filename string) ([]byte, error)

// Cli is a project specific wrapper to the ctl.Cli struct
type Cli struct {
	*ctl.Ctl
	ReadFileOrStdin ReadFileOrStdinFn

	flags struct {
		// hsmConfig specifies HSM configuration file
		hsmConfig *string

		debug   *bool
		verbose *bool
	}

	crypto *cryptoprov.Crypto
}

// New creates an instance of CLI
func New(d *ctl.ControlDefinition) *Cli {
	cli := &Cli{
		Ctl:             ctl.NewControl(d),
		ReadFileOrStdin: ReadStdin,
	}

	cli.flags.hsmConfig = d.App.Flag("hsm-cfg", "HSM provider configuration file").Required().String()
	cli.flags.verbose = d.App.Flag("verbose", "Verbose output").Short('V').Bool()
	cli.flags.debug = d.App.Flag("debug", "Redirect logs to stderr").Short('d').Bool()

	return cli
}

// Verbose specifies if verbose output is enabled
func (cli *Cli) Verbose() bool {
	return *cli.flags.verbose
}

// CryptoProv returns crypto provider
func (cli *Cli) CryptoProv() *cryptoprov.Crypto {
	if cli == nil || cli.crypto == nil {
		panic("use EnsureCryptoProvider() in App settings")
	}
	return cli.crypto
}

// RegisterAction create new Control action
func (cli *Cli) RegisterAction(f func(c ctl.Control, flags interface{}) error, params interface{}) ctl.Action {
	return func() error {
		err := f(cli, params)
		if err != nil {
			return cli.Fail("action failed", err)
		}
		return nil
	}
}

// EnsureCryptoProvider is pre-action to load Crypto provider
func (cli *Cli) EnsureCryptoProvider() error {
	if cli.crypto != nil {
		return nil
	}

	if *cli.flags.hsmConfig == "" {
		return errors.New("use --hsm-cfg flag to specify config file")
	}

	var err error
	cli.crypto, err = cryptoprov.Load(*cli.flags.hsmConfig, nil)
	if err != nil {
		return errors.WithMessage(err, "unable to initialize crypto providers")
	}

	return nil
}

// WithCryptoProvider sets custom Crypto Provider
func (cli *Cli) WithCryptoProvider(crypto *cryptoprov.Crypto) {
	cli.crypto = crypto
}

// ReadStdin reads from stdin if the file is "-"
func ReadStdin(filename string) ([]byte, error) {
	if filename == "" {
		return nil, errors.New("empty file name")
	}
	if filename == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	return ioutil.ReadFile(filename)
}

// WriteFile creates and writes to a file
func WriteFile(filespec string, contents []byte, perms os.FileMode) error {
	return ioutil.WriteFile(filespec, contents, perms)
}

// PrintCert outputs a cert, key and csr to stdout
func (cli *Cli) PrintCert(key, csrBytes, cert []byte) {
	out := map[string]string{}
	if cert != nil {
		out["cert"] = string(cert)
	}

	if key != nil {
		out["key"] = string(key)
	}

	if csrBytes != nil {
		out["csr"] = string(csrBytes)
	}

	jsonOut, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(cli.ErrWriter(), "unable to encode output: %s", err.Error())
		return
	}
	fmt.Fprintf(cli.Writer(), "%s\n", jsonOut)
}

// PopulateControl is a pre-action for kingpin library to populate the
// control object after all the flags are parsed
func (cli *Cli) PopulateControl() error {
	isDebug := *cli.flags.debug
	var sink io.Writer
	if isDebug {
		sink = cli.ErrWriter()
		xlog.SetFormatter(xlog.NewColorFormatter(sink, true))
		xlog.SetGlobalLogLevel(xlog.DEBUG)
	} else {
		xlog.SetGlobalLogLevel(xlog.CRITICAL)
	}

	return nil
}
