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
	return (*CmdClause)(kpc.Alias(name))
}

// Command adds a new sub-command.
func (c *CmdClause) Command(name, help string) *CmdClause {
	kpc := (*kp.CmdClause)(c)
	return (*CmdClause)(kpc.Command(name, help))
}

// Action adds a action for this command.
func (c *CmdClause) Action(action Action) *CmdClause {
	kpc := (*kp.CmdClause)(c)
	kpa := func(*kp.ParseContext) error {
		return action()
	}
	return (*CmdClause)(kpc.Action(kpa))
}

// PreAction adds a pre-action for this command.
func (c *CmdClause) PreAction(action Action) *CmdClause {
	kpc := (*kp.CmdClause)(c)
	kpa := func(*kp.ParseContext) error {
		return action()
	}
	return (*CmdClause)(kpc.PreAction(kpa))
}

func (a *proxyapp) Flag(name, help string) *FlagClause {
	f := a.Application.Flag(name, help)
	return (*FlagClause)(f)
}

// Default values for this flag. They *must* be parseable by the value of the flag.
func (f *FlagClause) Default(values ...string) *FlagClause {
	kpf := (*kp.FlagClause)(f)
	return (*FlagClause)(kpf.Default(values...))
}

// Hidden hides a flag from usage but still allows it to be used.
func (f *FlagClause) Hidden() *FlagClause {
	kpf := (*kp.FlagClause)(f)
	return (*FlagClause)(kpf.Hidden())
}

// Required makes the flag required. You can not provide a Default() value to a Required() flag.
func (f *FlagClause) Required() *FlagClause {
	kpf := (*kp.FlagClause)(f)
	return (*FlagClause)(kpf.Required())
}

// Short sets the short flag name.
func (f *FlagClause) Short(name rune) *FlagClause {
	kpf := (*kp.FlagClause)(f)
	return (*FlagClause)(kpf.Short(name))
}

// Bool makes this flag a boolean flag.
func (f *FlagClause) Bool() (target *bool) {
	kpf := (*kp.FlagClause)(f)
	return kpf.Bool()
}
