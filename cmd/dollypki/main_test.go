package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-phorce/dolly/ctl"
	"github.com/stretchr/testify/suite"
)

const projFolder = "../../"

type testSuite struct {
	suite.Suite
	baseArgs []string
	out      bytes.Buffer
}

func (s *testSuite) run(additionalFlags ...string) ctl.ReturnCode {
	rc := realMain(append(s.baseArgs, additionalFlags...), &s.out)
	return rc
}

// hasText is a helper method to assert that the out stream contains the supplied
// text somewhere
func (s *testSuite) hasText(t string) {
	s.True(strings.Index(s.out.String(), t) >= 0, "Expecting to find text %q in value %q", t, s.out.String())
}

func Test_RaphtyCtlSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupTest() {
	cfg, err := filepath.Abs(projFolder + "etc/dev/softhsm_unittest.json")
	s.Require().NoError(err)

	s.baseArgs = []string{"dollypki", "--hsm-cfg", cfg}
	s.out.Reset()
}

func (s *testSuite) TearDownTest() {
}

func (s *testSuite) Test_HsmSlots() {
	s.Equal(ctl.RCOkay, s.run("hsm", "slots"))
	s.hasText(`Slot:`)
}

func (s *testSuite) Test_HsmKeys() {
	s.Equal(ctl.RCOkay, s.run("hsm", "lskey"))
	s.hasText(`Slot:`)
}

func (s *testSuite) Test_HsmKeyInfo() {
	s.Equal(ctl.RCOkay, s.run("hsm", "keyinfo", "--id", "12345"))
	s.hasText(`failed to get key info on slot`)
}

func (s *testSuite) Test_HsmKeyDel() {
	s.Equal(ctl.RCOkay, s.run("hsm", "rmkey", "--id", "12345"))
}
