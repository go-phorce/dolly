package ctl_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fooActionParams struct {
	fooflag *string
}

type barActionParams struct {
	barflag *string
}

func Test_ParseCore(t *testing.T) {
	app := ctl.NewApplication("test", "A test command-line tool").Terminate(nil)
	//app.UsageWriter(os.Stderr)

	cli := ctl.NewControl(&ctl.ControlDefinition{
		App:    app,
		Output: os.Stderr,
	})

	app.Command("foo", "foo description").PreAction(cli.PopulateControl)
	app.Command("bar", "bar description").PreAction(cli.PopulateControl)

	assert.Equal(t, app, cli.App())
	assert.Equal(t, os.Stderr, cli.Writer())
	assert.Empty(t, cli.ServerURL())
	assert.Nil(t, cli.ServerURLs())

	foobar := app.Command("foobar", "foobar description").PreAction(cli.PopulateControl)
	foobarflag := foobar.Flag("foobarflag", "foobarflag description").Required().String()

	cmd, _ := parse(cli, []string{"test", "-V", "foo"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.True(t, cli.Verbose())
	assert.Empty(t, cli.ServerURL())
	assert.Nil(t, cli.ServerURLs())

	assert.Equal(t, 0, cli.RetryLimit())
	assert.Equal(t, time.Duration(0), cli.RetryTimeout())

	cmd, _ = parse(cli, []string{"test", "foo"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.False(t, cli.Verbose())

	cmd, _ = parse(cli, []string{"test", "foobar", "--foobarflag", "test"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foobar", cmd)
	assert.Equal(t, "test", *foobarflag)

	cmd, out := parse(cli, []string{"test", "--bogus", "foo"})
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

func Test_ParseCoreWithServer(t *testing.T) {
	outw := &bytes.Buffer{}

	app := ctl.NewApplication("test", "A test command-line tool with Server").Terminate(nil)
	//app.UsageWriter(os.Stderr)
	cli := ctl.NewControl(&ctl.ControlDefinition{
		App:        app,
		Output:     outw,
		WithServer: true,
	})

	cli.Print("1-1")
	cli.Printf("2-%d", 2)
	cli.Println("3")
	cli.PrintJSON(struct{}{})
	assert.Equal(t, "1-12-23\n{}\n", string(outw.Bytes()))

	app.Command("foo", "foo description").PreAction(cli.PopulateControl)
	app.Command("bar", "bar description").PreAction(cli.PopulateControl)

	foobar := app.Command("foobar", "foobar description").PreAction(cli.PopulateControl)
	foobarflag := foobar.Flag("foobarflag", "foobarflag description").Required().String()

	cmd, _ := parse(cli, []string{"test", "--json", "bar"})
	assert.Equal(t, cli.ReturnCode(), ctl.RCOkay)
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "bar", cmd)
	assert.False(t, cli.Verbose())
	assert.Equal(t, header.ApplicationJSON, cli.ContentType())
	assert.NotEmpty(t, cli.ServerURL())
	assert.Equal(t, 1, len(cli.ServerURLs()))
	assert.Equal(t, 0, cli.RetryLimit())
	assert.Equal(t, time.Duration(6*time.Second), cli.RetryTimeout())

	hn, err := os.Hostname()
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("https://%s", hn), cli.ServerURL())

	var res string
	cmd, res = parse(cli, []string{"test", "--server", "foo:80", "--server", "bar:8080", "foo"})
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())
	assert.Equal(t, "ERROR: unsupported URL scheme \"foo\", use http:// or https://\n", res)

	cmd, res = parse(cli, []string{"test", "--server", "http://foo:80", "--server", "bar:8080", "foo"})
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())
	assert.Equal(t, "ERROR: unsupported URL scheme \"bar\", use http:// or https://\n", res)

	cmd, _ = parse(cli, []string{"test", "--server", "http://foo:80", "--server", "http://bar:8080", "foo"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.Equal(t, "http://foo:80", cli.ServerURL())
	require.Equal(t, 2, len(cli.ServerURLs()))
	assert.Equal(t, "http://foo:80", cli.ServerURLs()[0].String())
	assert.Equal(t, "http://bar:8080", cli.ServerURLs()[1].String())

	cmd, _ = parse(cli, []string{"test", "--server", "https://foo:9999", "foo"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.Equal(t, "https://foo:9999", cli.ServerURL())
	assert.Equal(t, header.TextPlain, cli.ContentType())
	assert.Equal(t, 1, len(cli.ServerURLs()))

	cmd, _ = parse(cli, []string{"test", "--server", "https://foo:9999", "-s", "http://bar:9999", "foo"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.Equal(t, header.TextPlain, cli.ContentType())
	assert.Equal(t, "https://foo:9999", cli.ServerURL())
	require.Equal(t, 2, len(cli.ServerURLs()))
	assert.Equal(t, "https://foo:9999", cli.ServerURLs()[0].String())
	assert.Equal(t, "http://bar:9999", cli.ServerURLs()[1].String())

	cmd, _ = parse(cli, []string{"test", "--ct", "text/html", "foo"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.False(t, cli.Verbose())
	assert.Equal(t, "text/html", cli.ContentType())

	cmd, _ = parse(cli, []string{"test", "-V", "--json", "foo"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.True(t, cli.Verbose())
	assert.Equal(t, header.ApplicationJSON, cli.ContentType())

	cmd, _ = parse(cli, []string{"test", "--server", "https://foo:9999", "foobar", "--foobarflag", "test"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foobar", cmd)
	assert.Equal(t, "https://foo:9999", cli.ServerURL())
	assert.Equal(t, "test", *foobarflag)

	cmd, _ = parse(cli, []string{"test", "-s", "ekspander", "foo"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.Equal(t, "https://ekspander", cli.ServerURL())

	cmd, out := parse(cli, []string{"test", "--bogus", "foo"})
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())
	assert.Empty(t, cmd)
	assert.Equal(t, "ERROR: unknown long flag '--bogus'\n", out)

	cmd, out = parse(cli, []string{"test", "bob"})
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())
	assert.Empty(t, cmd)
	assert.Equal(t, "ERROR: expected command but got \"bob\"\n", out)

	cmd, out = parse(cli, []string{"test"})
	assert.Empty(t, cmd)
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())

	cli = ctl.NewControl(&ctl.ControlDefinition{
		App:              app,
		Output:           outw,
		WithServer:       true,
		DefaultServerURL: "https://ekspand.com:80",
	})
	cmd, _ = parse(cli, []string{"test"})
	assert.Empty(t, cmd)
	assert.Equal(t, "https://ekspand.com:80", cli.ServerURL())
}

func Test_Action(t *testing.T) {
	app := ctl.NewApplication("test", "A test command-line tool").Terminate(nil)
	//app.UsageWriter(os.Stderr)

	cli := ctl.NewControl(&ctl.ControlDefinition{
		App:        app,
		Output:     os.Stderr,
		WithServer: true,
	})

	fooFlags := new(fooActionParams)
	fooCmd := app.Command("foo", "testing Success Action").PreAction(cli.PopulateControl).Action(cli.RegisterAction(successAction, fooFlags))
	fooFlags.fooflag = fooCmd.Flag("fooflag", "fooflag description").Required().String()

	barFlags := new(barActionParams)
	barCmd := app.Command("bar", "testing Failed Action").PreAction(cli.PopulateControl).Action(cli.RegisterAction(failedAction, barFlags))
	barFlags.barflag = barCmd.Flag("barflag", "barflag description").Required().String()

	// if app.Terminate(nil)  is set, then --help without command shall fail
	cmd, out := parse(cli, []string{"test", "--help"})
	require.Equal(t, ctl.RCUsage, cli.ReturnCode())
	assert.Empty(t, cmd)
	assert.Contains(t, out, "command not specified")

	cmd, out = parse(cli, []string{"test", "-V", "foo", "--fooflag", "1"})
	require.Equal(t, ctl.RCOkay, cli.ReturnCode())
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "foo", cmd)
	assert.True(t, cli.Verbose())
	assert.Equal(t, "Verbose output: 1\nSuccessAction output\n", out)

	cmd, out = parse(cli, []string{"test", "bar", "--barflag", "2"})
	require.Equal(t, cli.ReturnCode(), ctl.RCFailed)
	assert.Empty(t, cmd)
	assert.False(t, cli.Verbose())
	assert.Equal(t, "ERROR: FailedAction\n", out)
}

func Test_AskForConfirmation(t *testing.T) {
	app := ctl.NewApplication("test", "A test command-line tool").Terminate(nil)
	cli := ctl.NewControl(&ctl.ControlDefinition{
		App:        app,
		Output:     os.Stderr,
		WithServer: true,
	})

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
			ok, err := cli.AskForConfirmation(strings.NewReader(tc.response), tc.prompt)
			if tc.err != "" {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.res, ok)
			}
		})
	}
}

func successAction(c ctl.Control, f interface{}) error {
	fooFlags := f.(*fooActionParams)
	if c.Verbose() {
		c.Printf("Verbose output: %s\n", *fooFlags.fooflag)
	}
	c.Println("SuccessAction output")
	return nil
}

func failedAction(c ctl.Control, f interface{}) error {
	barFlags := f.(*barActionParams)
	if c.Verbose() {
		c.Printf("Verbose output: %s\n", *barFlags.barflag)
	}

	return errors.New("FailedAction")
}

func parse(cli *ctl.Ctl, args []string) (string, string) {
	outw := &bytes.Buffer{}
	cli.Reset(outw)
	cmd := cli.Parse(args)
	return cmd, outw.String()
}
