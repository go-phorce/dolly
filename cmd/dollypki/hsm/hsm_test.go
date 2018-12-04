package hsm_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/cmd/dollypki/hsm"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/stretchr/testify/suite"
)

const projFolder = "../../../"

type testSuite struct {
	suite.Suite
	out bytes.Buffer
	cli *cli.Cli
}

// hasText is a helper method to assert that the out stream contains the supplied
// text somewhere
func (s *testSuite) hasText(t string) {
	s.True(strings.Index(s.out.String(), t) >= 0, "Expecting to find text %q in value %q", t, s.out.String())
}

func (s *testSuite) run(action ctl.ControlAction, p interface{}) error {
	s.out.Reset()
	return action(s.cli, p)
}

func Test_CtlSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupTest() {
	cryptoprov.Register("SoftHSM", cryptoprov.Crypto11Loader)

	cfg, err := filepath.Abs(projFolder + "etc/dev/softhsm_unittest.json")
	s.Require().NoError(err)
	s.out.Reset()

	app := ctl.NewApplication("cliapp", "test")
	app.UsageWriter(&s.out)

	s.cli = cli.New(&ctl.ControlDefinition{
		App:        app,
		Output:     &s.out,
		WithServer: false,
	})

	s.cli.Parse([]string{"cliapp", "--hsm-cfg", cfg})

	err = s.cli.EnsureCryptoProvider()
	s.Require().NoError(err)

	s.Require().NotPanics(func() {
		s.cli.CryptoProv()
	})
}

func (s *testSuite) TearDownTest() {
}

func (s *testSuite) Test_HsmSlots() {
	err := s.run(hsm.Slots, nil)
	s.NoError(err)
	s.hasText(`Slot:`)
}

func (s *testSuite) Test_HsmKeys() {
	err := s.run(hsm.Keys, &hsm.LsKeyFlags{})
	s.NoError(err)
	s.hasText(`Slot:`)
}

func (s *testSuite) Test_HsmKeyInfo() {
	err := s.run(hsm.KeyInfo, &hsm.KeyInfoFlags{})
	s.NoError(err)
	s.hasText(`Slot: `)

	id := "123"
	err = s.run(hsm.KeyInfo, &hsm.KeyInfoFlags{
		ID: &id,
	})
	s.NoError(err)
	s.hasText(`failed to get key info on slot`)
}

func (s *testSuite) Test_HsmKeyDel() {
	err := s.run(hsm.RmKey, &hsm.RmKeyFlags{})
	s.Error(err)
	s.Equal("either of --prefix and --id must be specified", err.Error())

	id := "123"
	err = s.run(hsm.RmKey, &hsm.RmKeyFlags{
		Prefix: &id,
	})
	s.Require().NoError(err)
	s.hasText("no keys found with prefix: 123")

	err = s.run(hsm.RmKey, &hsm.RmKeyFlags{
		ID: &id,
	})
	s.Require().NoError(err)
	s.Empty(s.out.String())
}
