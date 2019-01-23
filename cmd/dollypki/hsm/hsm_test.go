package hsm_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-phorce/dolly/algorithms/guid"
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
func (s *testSuite) hasText(texts ...string) {
	outStr := s.out.String()
	for _, t := range texts {
		s.True(strings.Index(outStr, t) >= 0, "Expecting to find text %q in value %q", t, outStr)
	}
}

func (s *testSuite) hasNoText(texts ...string) {
	outStr := s.out.String()
	for _, t := range texts {
		s.True(strings.Index(outStr, t) < 0, "Expecting to NOT find text %q in value %q", t, outStr)
	}
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

	id := guid.MustCreate()
	err = s.run(hsm.RmKey, &hsm.RmKeyFlags{
		Prefix: &id,
	})
	s.Require().NoError(err)
	s.hasText("no keys found with prefix: " + id)

	err = s.run(hsm.RmKey, &hsm.RmKeyFlags{
		ID: &id,
	})
	s.Require().NoError(err)
	s.Empty(s.out.String())
}

func (s *testSuite) Test_HsmGenKey() {
	err := s.run(hsm.GenKey, &hsm.GenKeyFlags{})
	s.Error(err)
	s.Equal(`unsupported purpose: ""`, err.Error())

	encrypt := "e"
	label := "TestHsmGenKey"
	algo := "algo"
	rsa := "rsa"
	size1024 := 1024
	size2048 := 2048
	yes := true

	err = s.run(hsm.GenKey, &hsm.GenKeyFlags{
		Purpose: &encrypt,
		Label:   &label,
	})
	s.Error(err)
	s.Equal(`invalid algorithm: `, err.Error())

	err = s.run(hsm.GenKey, &hsm.GenKeyFlags{
		Algo:    &algo,
		Purpose: &encrypt,
		Label:   &label,
	})
	s.Error(err)
	s.Equal(`invalid algorithm: algo`, err.Error())

	err = s.run(hsm.GenKey, &hsm.GenKeyFlags{
		Size:    &size1024,
		Algo:    &rsa,
		Purpose: &encrypt,
		Label:   &label,
	})
	s.Error(err)
	s.Equal(`validate RSA key: RSA key is too weak: 1024`, err.Error())

	err = s.run(hsm.GenKey, &hsm.GenKeyFlags{
		Size:    &size2048,
		Algo:    &rsa,
		Purpose: &encrypt,
		Label:   &label,
		Check:   &yes,
	})
	s.NoError(err)
}
