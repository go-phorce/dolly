package pkg_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-phorce/dolly/cmd/dollypki/pkg"
	"github.com/go-phorce/dolly/ctl"
	"github.com/stretchr/testify/suite"
)

const projFolder = "../../../"

type testSuite struct {
	suite.Suite
	baseArgs []string
	out      bytes.Buffer
}

func (s *testSuite) run(additionalFlags ...string) ctl.ReturnCode {
	rc := pkg.ParseAndRun("dollypki", append(s.baseArgs, additionalFlags...), &s.out)
	return rc
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

func Test_CtlSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupTest() {
	s.baseArgs = []string{"dollypki", "--hsm-cfg", "/tmp/dolly/softhsm_unittest.json"}
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

func (s *testSuite) TestCsr_genkey() {
	s.Equal(ctl.RCUsage, s.run("csr", "genkey"))
	s.hasText(`ERROR: required flag`)
}

func (s *testSuite) TestCsr_Gencert() {
	s.Equal(ctl.RCUsage, s.run("csr", "gencert"))
	s.hasText(`ERROR: required flag`)
}

func (s *testSuite) TestCsr_signcert() {
	s.Equal(ctl.RCUsage, s.run("csr", "signcert"))
	s.hasText(`ERROR: required flag`)
}
