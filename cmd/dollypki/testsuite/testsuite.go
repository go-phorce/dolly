package testsuite

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/stretchr/testify/suite"
)

const projFolder = "../../"

// Suite defines helper test suite
type Suite struct {
	suite.Suite
	// Out is the outpub buffer
	Out bytes.Buffer
	// Cli is the current CLI
	Cli *cli.Cli

	appFlags []string
}

// WithAppFlags specifies application flags, default: -V -d
func (s *Suite) WithAppFlags(appFlags []string) {
	s.appFlags = appFlags
}

// HasText is a helper method to assert that the out stream contains the supplied
// text somewhere
func (s *Suite) HasText(texts ...string) {
	outStr := s.Output()
	for _, t := range texts {
		s.True(strings.Index(outStr, t) >= 0, "Expecting to find text %q in value %q", t, outStr)
	}
}

// HasNoText is a helper method to assert that the out stream does contains the supplied
// text somewhere
func (s *Suite) HasNoText(texts ...string) {
	outStr := s.Output()
	for _, t := range texts {
		s.True(strings.Index(outStr, t) < 0, "Expecting to NOT find text %q in value %q", t, outStr)
	}
}

// HasFile is a helper method to assert that file exists
func (s *Suite) HasFile(file string) {
	stat, err := os.Stat(file)
	s.Require().NoError(err, "File must exist: %s", file)
	s.Require().False(stat.IsDir())
}

// HasTextInFile is a helper method to assert that file contains the supplied text
func (s *Suite) HasTextInFile(file string, texts ...string) {
	f, err := ioutil.ReadFile(file)
	s.Require().NoError(err, "Unable to read: %s", file)
	outStr := string(f)
	for _, t := range texts {
		s.True(strings.Index(outStr, t) >= 0, "Expecting to find text %q in file %q", t, file)
	}
}

// Run is a helper to run a CLI commnd
func (s *Suite) Run(action ctl.ControlAction, p interface{}) error {
	s.Out.Reset()
	return action(s.Cli, p)
}

// Output returns the current CLI output
func (s *Suite) Output() string {
	return s.Out.String()
}

// SetupSuite to set up the tests
func (s *Suite) SetupSuite() {
	app := ctl.NewApplication("cliapp", "test")
	app.UsageWriter(&s.Out)

	flags := []string{"cliapp", "-V"}
	if len(s.appFlags) > 0 {
		flags = append(flags, s.appFlags...)
	}

	s.Cli = cli.New(&ctl.ControlDefinition{
		App:       app,
		Output:    &s.Out,
		ErrOutput: &s.Out,
	})

	s.Cli.Parse(flags)
	s.Cli.PopulateControl()
}

// TearDownSuite to clean up resources
func (s *Suite) TearDownSuite() {
}
