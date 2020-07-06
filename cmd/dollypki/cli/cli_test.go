package cli_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projFolder = "../../../"

func cmdAction(c ctl.Control, p interface{}) error {
	fmt.Fprintf(c.Writer(), "cmd executed!\n")
	return nil
}

func Test_CLI(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	app := ctl.NewApplication("cliapp", "test")
	app.UsageWriter(out)

	cli := cli.New(&ctl.ControlDefinition{
		App:    app,
		Output: out,
	})

	cmd := app.Command("cmd", "Test command").
		PreAction(cli.PopulateControl).
		PreAction(cli.EnsureCryptoProvider)

	cmd.Command("subcmd", "Test sub command").Action(cli.RegisterAction(cmdAction, nil))

	cfg, err := filepath.Abs("/tmp/dolly/softhsm_unittest.json")
	require.NoError(t, err)

	cryptoprov.Register("SoftHSM", cryptoprov.Crypto11Loader)

	require.Panics(t, func() {
		cli.CryptoProv()
	})

	err = cli.EnsureCryptoProvider()
	assert.Error(t, err)
	assert.Equal(t, "use --hsm-cfg flag to specify config file", err.Error())

	cli.Parse([]string{"cliapp", "--hsm-cfg", cfg, "cmd", "subcmd"})

	err = cli.EnsureCryptoProvider()
	require.NoError(t, err)

	require.NotPanics(t, func() {
		cli.CryptoProv()
	})

	assert.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.Contains(t, out.String(), "cmd executed!")
}

func Test_ReadStdin(t *testing.T) {
	_, err := cli.ReadStdin("")
	require.Error(t, err)
	assert.Equal(t, "empty file name", err.Error())

	b, err := cli.ReadStdin("-")
	assert.NoError(t, err)
	assert.Empty(t, b)
}
