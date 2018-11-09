// Package ctl provides common code for building a command line control app that works via https
// calls to a service
package ctl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-phorce/dolly/xhttp/header"
	"github.com/go-phorce/dolly/xhttp/marshal"
	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xlog/logrotate"
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
	Writer() io.Writer
	Verbose() bool
	ServerURL() string
	ContentType() string

	Print(value string)
	Println(value string)
	Printf(format string, args ...interface{})
	PrintJSON(value interface{})
	Fail(msg string, err error) error
	AskForConfirmation(r io.Reader, s string) (bool, error)
	Parse(args []string) string
	ReturnCode() ReturnCode

	// Client returns application specific HTTP Client
	Client() interface{}
	// Config returns application specific Config
	Config() interface{}
}

// Ctl contains the definition and result from the parsed and initialized data
// It contains all the information needed by pre-action and action to perform the task
type Ctl struct {
	Control

	params     *ControlDefinition
	rc         ReturnCode
	logRotator io.Closer

	serverURL   string
	contentType string

	flags struct {
		// debug flag specifies to output logs to stderr
		debug *bool
		// verbose flag allows for vebose output
		verbose *bool

		// server URL
		url *string
		// specify Content-Type in Accepts header, by default "text/plain"
		contentType *string
		// shortcut for --ct=application/json
		ctJSON *bool

		// Retry settings
		retries *int
		timeout *int
	}
}

// NewControl creates new Control
func NewControl(d *ControlDefinition) *Ctl {
	if d.App == nil {
		logger.Panic("application is not specified in ControlDefinition")
	}
	ctl := &Ctl{
		params: d,
	}
	ctl.initGlobalFlags()
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

// Writer returns a writer for control output
func (ctl *Ctl) Writer() io.Writer {
	return ctl.params.Output
}

// Verbose specifies if verbose output is enabled
func (ctl *Ctl) Verbose() bool {
	return *ctl.flags.verbose
}

// ServerURL is the URL for the server to control
func (ctl *Ctl) ServerURL() string {
	return *ctl.flags.url
}

// RetryLimit returns retries limit
func (ctl *Ctl) RetryLimit() int {
	return *ctl.flags.retries
}

// RetryTimeout returns retries timeout
func (ctl *Ctl) RetryTimeout() time.Duration {
	if ctl.flags.timeout != nil && *ctl.flags.timeout > 0 {
		return time.Second * time.Duration(*ctl.flags.timeout)
	}
	return 0
}

// ContentType is content-type for the server commands
func (ctl *Ctl) ContentType() string {
	return *ctl.flags.contentType
}

// ReturnCode returns execution code
func (ctl *Ctl) ReturnCode() ReturnCode {
	return ctl.rc
}

// Print prints string
func (ctl *Ctl) Print(value string) {
	ctl.params.Output.Write([]byte(value))
}

var newLine = []byte("\n")

// Println  prints new line
func (ctl *Ctl) Println(value string) {
	ctl.params.Output.Write([]byte(value))
	ctl.params.Output.Write(newLine)
}

// Printf prints formatted string
func (ctl *Ctl) Printf(format string, args ...interface{}) {
	fmt.Fprintf(ctl.params.Output, format, args...)
}

// PrintJSON prints JSON response to Output
func (ctl *Ctl) PrintJSON(value interface{}) {
	WriteJSON(ctl.params.Output, value)
}

// Fail the execution and return error
func (ctl *Ctl) Fail(msg string, err error) error {
	ctl.rc = RCFailed
	logger.Errorf("api=Ctl, message='%s', err=[%s]", msg, errors.ErrorStack(err))
	return err
}

// Reset is used mostly in tests to reset the control to initial state
func (ctl *Ctl) Reset(w io.Writer) {
	// reset ctl
	if ctl.params.WithServer {
		*ctl.flags.ctJSON = false
		*ctl.flags.contentType = ""
		*ctl.flags.url = ""
	}

	*ctl.flags.debug = false
	*ctl.flags.verbose = false

	ctl.params.Output = w
	ctl.rc = RCOkay
}

// AskForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user or if an error occurs.
func (ctl *Ctl) AskForConfirmation(r io.Reader, s string) (bool, error) {
	if r == nil {
		r = os.Stdin
	}

	reader := bufio.NewReader(r)

	for i := 0; i < 3; i++ {
		ctl.Printf("%s [y/n]: ", s)

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

// ControlDefinition contains the default settings for control application
type ControlDefinition struct {
	App Application
	// Output is the destination for all output from the command, typically set to os.Stdout
	Output io.Writer

	// WithServer specifies if the CTL operates with a remote server
	WithServer bool
	// DefaultServerURL is the default URL that the service to be controlled is accessed on
	DefaultServerURL string
	// DefaultContentType specifies Content-Type in Accepts header: "text/plain"
	DefaultContentType string
}

// InitGlobalFlags builds a FlagSet configured with the default set of flags & usage formatting.
// Once created, the caller can add additional flags if needed.
func (ctl *Ctl) initGlobalFlags() {
	app := ctl.params.App

	ctl.flags.verbose = app.Flag("verbose", "Verbose output").Short('V').Bool()
	ctl.flags.debug = app.Flag("debug", "Redirect logs to stderr").Short('d').Bool()

	if ctl.params.WithServer {
		defURL := ctl.params.DefaultServerURL
		if defURL == "" {
			hn, _ := os.Hostname()
			defURL = fmt.Sprintf("https://%s", hn)
		}
		ctl.flags.url = app.Flag("server", "Url to server to connect to").Default(defURL).Short('s').String()
		ctl.flags.contentType = app.Flag("ct", "Content-Type in Accepts header, by default 'text/plain'").String()
		ctl.flags.ctJSON = app.Flag("json", "Use JSON Content-Type in Accepts header and printed response").Bool()

		ctl.flags.retries = app.Flag("retries", "Number of retries for connect failures").Default("0").Int()
		ctl.flags.timeout = app.Flag("timeout", "Timeout in seconds").Default("6").Int()
	}
}

// Parse will parse all the supplied args and
// will perform any pre-actions and actions defined on the command
func (ctl *Ctl) Parse(args []string) string {
	cmd, err := ctl.params.App.Parse(args[1:])
	if ctl.logRotator != nil {
		defer ctl.logRotator.Close()
	}
	if err != nil {
		if ctl.rc != RCFailed {
			ctl.rc = RCUsage
		}
		fmt.Fprintf(ctl.params.Output, "ERROR: %s\n", err.Error())
		return ""
	}
	return cmd
}

// PopulateControl is a pre-action for kingpin library to populate the
// control object after all the flags are parsed
func (ctl *Ctl) PopulateControl() error {
	var err error

	isDebug := *ctl.flags.debug
	appName := filepath.Base(os.Args[0])
	logFolder := filepath.Join(os.TempDir(), appName, "logs")
	var sink io.Writer
	if isDebug {
		sink = os.Stderr
		xlog.SetFormatter(xlog.NewColorFormatter(sink, true))
	}
	logRotator, err := logrotate.Initialize(logFolder, appName, 7, 10, false, sink)
	if err != nil {
		logger.Error("unable to initialize log file '%s/%s.log': %s", logFolder, appName, err)
	} else {
		logger.Infof("log file: %s/%s.log", logFolder, appName)
	}
	xlog.SetGlobalLogLevel(xlog.TRACE)

	ctl.logRotator = logRotator

	if ctl.params.WithServer {
		if *ctl.flags.ctJSON {
			if *ctl.flags.contentType != "" && *ctl.flags.contentType != header.ApplicationJSON {
				return errors.Errorf("conflict with --ct flag value: %s", *ctl.flags.contentType)
			}
			*ctl.flags.contentType = header.ApplicationJSON
		}

		if *ctl.flags.contentType == "" {
			if ctl.params.DefaultContentType != "" {
				*ctl.flags.contentType = ctl.params.DefaultContentType
			} else {
				*ctl.flags.contentType = header.TextPlain
			}
		}

		url := *ctl.flags.url
		if url == "" {
			url = ctl.params.DefaultServerURL
		}
		if !strings.HasPrefix(url, "http") {
			url = fmt.Sprintf("https://%s", url)
		}
		*ctl.flags.url = url
	}

	return nil
}

// WriteJSON prints response to out
func WriteJSON(out io.Writer, value interface{}) error {
	json, err := marshal.EncodeBytes(marshal.PrettyPrint, value)
	if err != nil {
		return errors.Annotate(err, "failed to encode respose")
	}
	out.Write([]byte(json))
	out.Write(newLine)

	return nil
}
