package ctl

import (
	"io"

	kp "gopkg.in/alecthomas/kingpin.v2"
)

// Action is a CTL action
type Action func() error

// CmdClause is a proxy interface to kingpin's *CmdClause
type CmdClause kp.CmdClause

// FlagClause is a proxy interface to kingpin's *FlagClause
type FlagClause kp.FlagClause

// Application is a proxy interface to kingpin's *Application
type Application interface {
	Version(version string) Application
	Parse(args []string) (command string, err error)
	Command(name, help string) *CmdClause
	Flag(name, help string) *FlagClause
	Terminate(terminate func(int)) Application
	Writer(w io.Writer) Application
	ErrorWriter(w io.Writer) Application
	UsageWriter(w io.Writer) Application
}

type proxyapp struct {
	*kp.Application
}

// NewApplication creates a new application instance
func NewApplication(name, help string) Application {
	return &proxyapp{
		Application: kp.New(name, help),
	}
}

func (a *proxyapp) Terminate(terminate func(int)) Application {
	a.Application.Terminate(terminate)
	return a
}

func (a *proxyapp) Parse(args []string) (command string, err error) {
	return a.Application.Parse(args)
}

// Writer sets the io.Writer to use for output.
func (a *proxyapp) Writer(w io.Writer) Application {
	a.Application.Writer(w)
	return a
}

// ErrorWriter sets the io.Writer to use for errors.
func (a *proxyapp) ErrorWriter(w io.Writer) Application {
	a.Application.ErrorWriter(w)
	return a
}

// UsageWriter sets the io.Writer to use for errors.
func (a *proxyapp) UsageWriter(w io.Writer) Application {
	a.Application.UsageWriter(w)
	return a
}

// Version adds a --version flag for displaying the application version.
func (a *proxyapp) Version(version string) Application {
	a.Application.Version(version)
	return a
}

// Command adds a new top-level command.
func (a *proxyapp) Command(name, help string) *CmdClause {
	return (*CmdClause)(a.Application.Command(name, help))
}

// Alias adds an alias for this command.
func (c *CmdClause) Alias(name string) *CmdClause {
	kpc := (*kp.CmdClause)(c)
	kpc.Alias(name)
	return c
}

// Command adds a new sub-command.
func (c *CmdClause) Command(name, help string) *CmdClause {
	kpc := (*kp.CmdClause)(c)
	return (*CmdClause)(kpc.Command(name, help))
}

// Action adds an action for this command.
func (c *CmdClause) Action(action Action) *CmdClause {
	kpc := (*kp.CmdClause)(c)
	kpa := func(*kp.ParseContext) error {
		return action()
	}
	kpc.Action(kpa)
	return c
}

// PreAction adds a pre-action for this command.
func (c *CmdClause) PreAction(action Action) *CmdClause {
	kpc := (*kp.CmdClause)(c)
	kpa := func(*kp.ParseContext) error {
		return action()
	}
	kpc.PreAction(kpa)
	return c
}

// Flag defines a new flag with the given long name and help.
func (c *CmdClause) Flag(name, help string) *FlagClause {
	kpc := (*kp.CmdClause)(c)
	f := kpc.Flag(name, help)
	return (*FlagClause)(f)
}

func (a *proxyapp) Flag(name, help string) *FlagClause {
	f := a.Application.Flag(name, help)
	return (*FlagClause)(f)
}

// Default values for this flag. They *must* be parseable by the value of the flag.
func (f *FlagClause) Default(values ...string) *FlagClause {
	kpf := (*kp.FlagClause)(f)
	kpf.Default(values...)
	return f
}

// Hidden hides a flag from usage but still allows it to be used.
func (f *FlagClause) Hidden() *FlagClause {
	kpf := (*kp.FlagClause)(f)
	kpf.Hidden()
	return f
}

// Required makes the flag required. You can not provide a Default() value to a Required() flag.
func (f *FlagClause) Required() *FlagClause {
	kpf := (*kp.FlagClause)(f)
	kpf.Required()
	return f
}

// Short sets the short flag name.
func (f *FlagClause) Short(name rune) *FlagClause {
	kpf := (*kp.FlagClause)(f)
	kpf.Short(name)
	return f
}

// Bool makes this flag a boolean flag.
func (f *FlagClause) Bool() (target *bool) {
	kpf := (*kp.FlagClause)(f)
	return kpf.Bool()
}

// Settings interface
type Settings kp.Settings

// GetFilesList retruns list from command arguments
func GetFilesList(s Settings) *FilesList {
	target := (*FilesList)(new([]string))
	s.SetValue(target)
	return target
}
