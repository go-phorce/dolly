// Package ctl provides common code for building a command line control app that works via https
// calls to a service
package ctl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-phorce/dolly/xhttp/marshal"
	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "ctl")

// ReturnCode is the type that your command returns, these map to standard process return codes
type ReturnCode int

const (
	// RCOkay denotes success
	RCOkay ReturnCode = 0
	// RCFailed denotes a failure in the requested command
	RCFailed ReturnCode = 1
	// RCUsage denotes that the parmaters supplied to the tool were somehow incorrect
	RCUsage ReturnCode = 64
)

// ControlAction is a wrapper over kp.Action
type ControlAction func(c Control, flags interface{}) error

// Control is an interface for CLI
type Control interface {
	App() Application
	// Reader is the source to read from, typically set to os.Stdin
	Reader() io.Reader
	// Writer is the destination for all output from the command, typically set to os.Stdout
	Writer() io.Writer
	// ErrWriter is the destinaton for errors, typically set to os.Stderr
	ErrWriter() io.Writer

	Fail(msg string, err error) error
	Parse(args []string) string
	ReturnCode() ReturnCode
}

// Ctl contains the definition and result from the parsed and initialized data
// It contains all the information needed by pre-action and action to perform the task
type Ctl struct {
	Control

	params *ControlDefinition
	rc     ReturnCode
}

// NewControl creates new Control
func NewControl(d *ControlDefinition) *Ctl {
	if d.App == nil {
		logger.Panic("App variable is not provided in ControlDefinition")
	}
	ctl := &Ctl{
		params: d,
	}
	return ctl
}

// RegisterAction create new Control action
func (ctl *Ctl) RegisterAction(f ControlAction, params interface{}) Action {
	return func() error {
		err := f(ctl, params)
		if err != nil {
			return ctl.Fail("action failed", err)
		}
		return nil
	}
}

// App returns current control App
func (ctl *Ctl) App() Application {
	return ctl.params.App
}

// Reader is the source to read from, typically set to os.Stdin
func (ctl *Ctl) Reader() io.Reader {
	if ctl.params.Stdin == nil {
		return os.Stdin
	}
	return ctl.params.Stdin
}

// WithReader allows to specify a custom reader
func (ctl *Ctl) WithReader(reader io.Reader) *Ctl {
	ctl.params.Stdin = reader
	return ctl
}

// Writer returns a writer for control output
func (ctl *Ctl) Writer() io.Writer {
	if ctl.params.Output == nil {
		return os.Stdout
	}
	return ctl.params.Output
}

// WithWriter allows to specify a custom writer
func (ctl *Ctl) WithWriter(out io.Writer) *Ctl {
	ctl.params.Output = out
	return ctl
}

// ErrWriter returns a writer for control output
func (ctl *Ctl) ErrWriter() io.Writer {
	if ctl.params.ErrOutput != nil {
		return ctl.params.ErrOutput
	}
	return os.Stderr
}

// WithErrWriter allows to specify a custom error writer
func (ctl *Ctl) WithErrWriter(out io.Writer) *Ctl {
	ctl.params.ErrOutput = out
	return ctl
}

// ReturnCode returns execution code
func (ctl *Ctl) ReturnCode() ReturnCode {
	return ctl.rc
}

// Fail the execution and return error
func (ctl *Ctl) Fail(msg string, err error) error {
	ctl.rc = RCFailed
	logger.Errorf("api=Ctl, message=%q, err=[%s]", msg, errors.ErrorStack(err))
	return err
}

// Reset is used mostly in tests to reset the control to initial state
func (ctl *Ctl) Reset(out io.Writer, errout io.Writer) {
	ctl.params.Output = out
	ctl.params.ErrOutput = errout
	ctl.rc = RCOkay
}

// ControlDefinition contains the default settings for control application
type ControlDefinition struct {
	App Application
	// Stdin is the source to read from, typically set to os.Stdin
	Stdin io.Reader
	// Output is the destination for all output from the command, typically set to os.Stdout
	Output io.Writer
	// ErrOutput is the destinaton for errors.
	// If not set, errors will be written to os.StdError
	ErrOutput io.Writer
}

// Parse will parse all the supplied args and
// will perform any pre-actions and actions defined on the command
func (ctl *Ctl) Parse(args []string) string {
	cmd, err := ctl.params.App.Parse(args[1:])
	if err != nil {
		if ctl.rc != RCFailed {
			ctl.rc = RCUsage
		}
		fmt.Fprintf(ctl.ErrWriter(), "ERROR: %s\n", err.Error())
		return ""
	}
	return cmd
}

var newLine = []byte("\n")

// WriteJSON prints response to out
func WriteJSON(out io.Writer, value interface{}) error {
	json, err := marshal.EncodeBytes(marshal.PrettyPrint, value)
	if err != nil {
		return errors.Annotate(err, "failed to encode")
	}
	out.Write(json)

	return nil
}

// AskForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again.
// The function does not return until it gets a valid response from the user or if an error occurs.
func AskForConfirmation(out io.Writer, r io.Reader, s string) (bool, error) {
	if out == nil {
		out = os.Stdout
	}
	if r == nil {
		r = os.Stdin
	}

	reader := bufio.NewReader(r)

	for i := 0; i < 3; i++ {
		fmt.Fprintf(out, "%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			return false, errors.Errorf("ReadString failed: [%v]", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true, nil
		} else if response == "n" || response == "no" {
			break
		}
	}
	return false, nil
}
