// Command dollypki is a command-line utility for managing HSM keys and creating certificates
package main

import (
	"io"
	"os"

	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/cmd/dollypki/hsm"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/cmd", "dollypki")

func main() {
	// Logs are set to os.Stderr, while output to os.Stdout
	rc := realMain(os.Args, os.Stdout)
	os.Exit(int(rc))
}

func realMain(args []string, out io.Writer) ctl.ReturnCode {
	app := ctl.NewApplication("dollypki", " command-line utility for managing HSM keys and creating certificates")
	app.UsageWriter(out)

	cli := cli.New(&ctl.ControlDefinition{
		App:        app,
		Output:     out,
		WithServer: false,
	})

	//app.HelpFlag.Short('h')
	//app.VersionFlag.Short('v')

	// hsm slots|lskey|rmkey
	cmdHsm := app.Command("hsm", "Perform HSM operations").
		PreAction(cli.PopulateControl).
		PreAction(cli.EnsureCryptoProvider)

	cmdHsm.Command("slots", "Show available slots list").Action(cli.RegisterAction(hsm.Slots, nil))

	hsmLsKeyFlags := new(hsm.LsKeyFlags)
	cmdHsmKeys := cmdHsm.Command("lskey", "Show keys list").Action(cli.RegisterAction(hsm.Keys, hsmLsKeyFlags))
	hsmLsKeyFlags.Token = cmdHsmKeys.Flag("token", "slot token").String()
	hsmLsKeyFlags.Serial = cmdHsmKeys.Flag("serial", "slot serial").String()
	hsmLsKeyFlags.Prefix = cmdHsmKeys.Flag("prefix", "key label prefix").String()

	hsmRmKeyFlags := new(hsm.RmKeyFlags)
	cmdRmKey := cmdHsm.Command("rmkey", "Destroy key").Action(cli.RegisterAction(hsm.RmKey, hsmRmKeyFlags))
	hsmRmKeyFlags.Token = cmdRmKey.Flag("token", "slot token").String()
	hsmRmKeyFlags.Serial = cmdRmKey.Flag("serial", "slot serial").String()
	hsmRmKeyFlags.ID = cmdRmKey.Flag("id", "key ID").String()
	hsmRmKeyFlags.Prefix = cmdRmKey.Flag("prefix", "remove keys based on the specified label prefix").String()
	hsmRmKeyFlags.Force = cmdRmKey.Flag("force", "do not ask for confirmation to remove keys").Bool()

	hsmKeyInfoFlags := new(hsm.KeyInfoFlags)
	cmdKeyInfo := cmdHsm.Command("keyinfo", "Get key info").Action(cli.RegisterAction(hsm.KeyInfo, hsmKeyInfoFlags))
	hsmKeyInfoFlags.Token = cmdKeyInfo.Flag("token", "slot token").String()
	hsmKeyInfoFlags.Serial = cmdKeyInfo.Flag("serial", "slot serial").String()
	hsmKeyInfoFlags.ID = cmdKeyInfo.Flag("id", "key ID").Required().String()
	hsmKeyInfoFlags.Public = cmdKeyInfo.Flag("public", "include public key").Bool()

	/* TODO:
	// PKI related command
	genkeyFlags := new(csr.GenKeyFlags)
	cmdGenkey := app.Command("genkey", "Generate key and CSR profile").
		PreAction(cli.PopulateControl).
		PreAction(cli.EnsureCryptoProvider).
		Action(cli.RegisterAction(csr.GenKey, genkeyFlags))
	genkeyFlags.Initca = cmdGenkey.Flag("initca", "Generate self-signed CA").Bool()
	genkeyFlags.CsrProfile = cmdGenkey.Flag("csr-profile", "CSR profile file").Required().String()
	genkeyFlags.Label = cmdGenkey.Flag("label", "Label for generated key").String()
	genkeyFlags.Output = cmdGenkey.Flag("output", "Optional prefix for output files").String()

	gencertFlags := new(csr.GenCertFlags)
	cmdGencert := app.Command("gencert", "Generate a new key and cert from CSR profile").
		PreAction(cli.PopulateControl).
		PreAction(cli.EnsureCryptoProvider).
		Action(cli.RegisterAction(csr.GenCert, gencertFlags))
	gencertFlags.CAConfig = cmdGencert.Flag("ca-config", "CA configuration file").Required().String()
	gencertFlags.CsrProfile = cmdGencert.Flag("csr-profile", "CSR profile file").Required().String()
	gencertFlags.Profile = cmdGencert.Flag("profile", "The profile name from ca-config").Required().String()
	gencertFlags.Label = cmdGencert.Flag("label", "Label for generated key").String()
	gencertFlags.Hostname = cmdGencert.Flag("hostname", "Coma-separated list of Host names for generated cert").String()
	gencertFlags.CA = cmdGencert.Flag("ca", "File name with CA cert").Required().String()
	gencertFlags.CAKey = cmdGencert.Flag("ca-key", "File name with CA key").Required().String()
	gencertFlags.Output = cmdGencert.Flag("output", "Optional prefix for output files").String()

	signcertFlags := new(csr.SignCertFlags)
	cmdSigncert := app.Command("signcert", "Sign cert from CSR request").
		PreAction(cli.PopulateControl).
		PreAction(cli.EnsureCryptoProvider).
		Action(cli.RegisterAction(csr.SignCert, signcertFlags))
	signcertFlags.CAConfig = cmdSigncert.Flag("ca-config", "CA configuration file").Required().String()
	signcertFlags.Csr = cmdSigncert.Flag("csr", "PEM-encoded CSR file").Required().String()
	signcertFlags.Profile = cmdSigncert.Flag("profile", "the profile name from ca-config").Required().String()
	signcertFlags.Hostname = cmdSigncert.Flag("hostname", "coma-separated list of Host names for generated cert").String()
	signcertFlags.CA = cmdSigncert.Flag("ca", "file name with CA cert").Required().String()
	signcertFlags.CAKey = cmdSigncert.Flag("ca-key", "file name with CA key").Required().String()
	signcertFlags.Output = cmdSigncert.Flag("output", "optional prefix for output files").String()
	*/

	cryptoprov.Register("SoftHSM", cryptoprov.Crypto11Loader)

	cli.Parse(args)
	return cli.ReturnCode()
}
