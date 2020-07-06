package ctl_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-phorce/dolly/ctl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fooActionParams struct {
	fooflag *string
}

type barActionParams struct {
	barflag *string
}

func Test_NewControl(t *testing.T) {
	assert.Panics(t, func() {
		ctl.NewControl(&ctl.ControlDefinition{})
	})
}

func Test_NewApplication(t *testing.T) {
	outusage := &bytes.Buffer{}

	app := ctl.NewApplication("test", "A test command-line tool").
		Terminate(nil).
		UsageWriter(outusage)

	cli := ctl.NewControl(&ctl.ControlDefinition{
		App:       app,
		Output:    os.Stdout,
		ErrOutput: os.Stderr,
	})

	app.Writer(os.Stdout)
	app.ErrorWriter(os.Stdout)
	app.UsageWriter(os.Stdout)
	app.Version("1")

	assert.Equal(t, app, cli.App())
	assert.Equal(t, os.Stdin, cli.Reader())
	assert.Equal(t, os.Stdout, cli.Writer())
	assert.Equal(t, os.Stderr, cli.ErrWriter())

	cli.WithReader(nil)
	assert.Equal(t, os.Stdin, cli.Reader())
	cli.WithReader(os.Stdin)
	assert.Equal(t, os.Stdin, cli.Reader())
	cli.WithWriter(nil)
	assert.Equal(t, os.Stdout, cli.Writer())
	cli.WithErrWriter(nil)
	assert.Equal(t, os.Stderr, cli.ErrWriter())

	app.Flag("V", "verbose").Default("false").String()
	app.Flag("H", "hidden").Default("none").Hidden().String()
	app.Flag("R", "some required flag").Required().Bool()

	sub := app.Command("foo2", "level1 command").
		PreAction(func() error { return nil }).
		Action(cli.RegisterAction(func(c ctl.Control, _ interface{}) error {
			fmt.Fprintf(c.Writer(), "foo2 action executed\n")
			return nil
		}, nil))
	sub.Flag("V1", "another flag").Default("true").Short('v').Bool()
	sub.Flag("R1", "some required flag").Required().String()

	cmdFoo := app.Command("foo", "foo description").
		Alias("bar").
		PreAction(func() error { return nil })

	sub2 := cmdFoo.Command("bar", "level2 command of foo").
		PreAction(func() error { return nil }).
		Action(cli.RegisterAction(func(c ctl.Control, _ interface{}) error {
			fmt.Fprintf(c.Writer(), "bar action executed\n")
			return nil
		}, nil))
	sub2.Flag("V2", "another flag").Default("true").Short('v').Bool()
	sub2.Flag("R2", "some required flag").Required().String()

	app.UsageWriter(outusage)
	outusage.Reset()
	cmd, out := parse(cli, []string{"test", "--R", "help"})
	assert.Equal(t, "help", cmd)
	assert.Equal(t, `usage: test --[no-]R [<flags>] <command> [<args> ...]

A test command-line tool

Flags:
  --help       Show context-sensitive help (also try --help-long and
               --help-man).
  --version    Show application version.
  --V="false"  verbose
  --R          some required flag

Commands:
  help [<command>...]
    Show help.

  foo2 --R1=R1 [<flags>]
    level1 command

  foo bar --R2=R2 [<flags>]
    level2 command of foo


`, string(outusage.Bytes()))

	cmd, out = parse(cli, []string{"test", "--R", "foo2", "--R1", "param1"})
	assert.Equal(t, "foo2", cmd)
	assert.Equal(t, "foo2 action executed\n", out)

	cmd, out = parse(cli, []string{"test", "--R", "foo", "bar", "--R2", "param1"})
	assert.Equal(t, "foo bar", cmd)
	assert.Equal(t, "bar action executed\n", out)
}

func Test_ParseCore(t *testing.T) {
	app := ctl.NewApplication("test", "A test command-line tool").Terminate(nil)
	//app.UsageWriter(os.Stderr)

	cli := ctl.NewControl(&ctl.ControlDefinition{
		App:       app,
		Output:    os.Stdout,
		ErrOutput: os.Stderr,
	})

	app.Command("foo", "foo description")
	app.Command("bar", "bar description")

	assert.Equal(t, app, cli.App())
	assert.Equal(t, os.Stdout, cli.Writer())
	assert.Equal(t, os.Stderr, cli.ErrWriter())

	foobar := app.Command("foobar", "foobar description")
	foobarflag := foobar.Flag("foobarflag", "foobarflag description").Required().String()

	cmd, out := parse(cli, []string{"test", "-V", "foo"})
	assert.Empty(t, cmd)
	assert.Equal(t, "ERROR: unknown short flag '-V'\n", out)

	cmd, _ = parse(cli, []string{"test", "foo"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)

	cmd, _ = parse(cli, []string{"test", "foobar", "--foobarflag", "test"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foobar", cmd)
	assert.Equal(t, "test", *foobarflag)

	cmd, out = parse(cli, []string{"test", "--bogus", "foo"})
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())
	assert.Empty(t, cmd)
	assert.Equal(t, "ERROR: unknown long flag '--bogus'\n", out)

	cmd, out = parse(cli, []string{"test", "bob"})
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())
	assert.Empty(t, cmd)
	assert.Equal(t, "ERROR: expected command but got \"bob\"\n", out)

	cmd, _ = parse(cli, []string{"test"})
	assert.Empty(t, cmd)
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())
}

func Test_Action(t *testing.T) {
	app := ctl.NewApplication("test", "A test command-line tool").Terminate(nil)
	//app.UsageWriter(os.Stderr)

	cli := ctl.NewControl(&ctl.ControlDefinition{
		App:       app,
		Output:    os.Stdout,
		ErrOutput: nil,
	})
	assert.Equal(t, os.Stdout, cli.Writer())
	assert.Equal(t, os.Stderr, cli.ErrWriter())

	fooFlags := new(fooActionParams)
	fooCmd := app.Command("foo", "testing Success Action").Action(cli.RegisterAction(successAction, fooFlags))
	fooFlags.fooflag = fooCmd.Flag("fooflag", "fooflag description").Required().String()

	barFlags := new(barActionParams)
	barCmd := app.Command("bar", "testing Failed Action").Action(cli.RegisterAction(failedAction, barFlags))
	barFlags.barflag = barCmd.Flag("barflag", "barflag description").Required().String()

	// if app.Terminate(nil)  is set, then --help without command shall fail
	cmd, out := parse(cli, []string{"test", "--help"})
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())
	assert.Empty(t, cmd)
	assert.Contains(t, out, "command not specified")

	cmd, out = parse(cli, []string{"test", "foo", "--fooflag", "1"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.Equal(t, "SuccessAction output\n", out)

	cmd, out = parse(cli, []string{"test", "bar", "--barflag", "2"})
	require.Equal(t, cli.ReturnCode(), ctl.RCFailed)
	assert.Empty(t, cmd)
	assert.Equal(t, "ERROR: FailedAction\n", out)
}

func Test_WriteJSON(t *testing.T) {
	out := bytes.NewBuffer([]byte{})
	val := struct{ Key, Value string }{Key: "n", Value: "v"}

	err := ctl.WriteJSON(out, val)
	require.NoError(t, err)
	assert.Equal(t, `{
	"Key": "n",
	"Value": "v"
}`, string(out.Bytes()))
}

func Test_AskForConfirmation(t *testing.T) {
	tcases := []struct {
		prompt   string
		response string
		res      bool
		err      string
	}{
		{"Answer_y", "y\n", true, ""},
		{"Answer_yes", "yes\n", true, ""},
		{"Answer_no", "no\n", false, ""},
		{"Answer_n", "n\n", false, ""},
		{"Answer_nn", "nn", false, "error"},
	}

	for _, tc := range tcases {
		t.Run(tc.prompt, func(t *testing.T) {
			ok, err := ctl.AskForConfirmation(os.Stdout, strings.NewReader(tc.response), tc.prompt)
			if tc.err != "" {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.res, ok)
			}
		})
	}

	_, err := ctl.AskForConfirmation(nil, strings.NewReader("bogus"), "Answer Y/N")
	assert.Error(t, err)
}

func successAction(c ctl.Control, f interface{}) error {
	fmt.Fprintf(c.Writer(), "SuccessAction output\n")
	return nil
}

func failedAction(c ctl.Control, f interface{}) error {
	return errors.New("FailedAction")
}

func parse(cli *ctl.Ctl, args []string) (string, string) {
	outw := &bytes.Buffer{}
	cli.Reset(outw, outw)
	cmd := cli.Parse(args)
	return cmd, outw.String()
}
