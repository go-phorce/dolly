package csr_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/cmd/dollypki/csr"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/stretchr/testify/suite"
)

const projFolder = "../../../"

type testSuite struct {
	suite.Suite
	out     bytes.Buffer
	tempDir string
	cli     *cli.Cli
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

func (s *testSuite) hasTextInFile(file string, texts ...string) {
	f, err := ioutil.ReadFile(file)
	s.Require().NoError(err, "Unable to read: %s", file)

	str := string(f)
	for _, t := range texts {
		s.True(strings.Index(str, t) >= 0, "Expecting to find text %q in file %q", t, file)
	}
}

func (s *testSuite) hasNoTextInFile(file string, texts ...string) {
	f, err := ioutil.ReadFile(file)
	s.Require().NoError(err, "Unable to read: %s", file)

	str := string(f)
	for _, t := range texts {
		s.False(strings.Index(str, t) >= 0, "NOT expecting to find text %q in file %q", t, file)
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

	s.tempDir = filepath.Join(os.TempDir(), "csrtest-"+guid.MustCreate())
	err = os.MkdirAll(s.tempDir, 0777)
	s.Require().NoError(err)
}

func (s *testSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *testSuite) Test_GenKey() {
	err := s.run(csr.GenKey, &csr.GenKeyFlags{})
	s.Require().Error(err)
	s.Equal(`read CSR profile: open : no such file or directory`, err.Error())

	label := guid.MustCreate()
	missingcsrprofile := "testdata/missing.json"
	err = s.run(csr.GenKey, &csr.GenKeyFlags{
		CsrProfile: &missingcsrprofile,
		Label:      &label,
	})
	s.Require().Error(err)
	s.Equal(`read CSR profile: open testdata/missing.json: no such file or directory`, err.Error())

	csrprofile := "testdata/test_dolly_client-csr.json"
	err = s.run(csr.GenKey, &csr.GenKeyFlags{
		CsrProfile: &csrprofile,
		Label:      &label,
	})
	s.Require().NoError(err)
	s.hasText(`-----BEGIN CERTIFICATE REQUEST-----`)

	initca := true
	err = s.run(csr.GenKey, &csr.GenKeyFlags{
		CsrProfile: &csrprofile,
		Label:      &label,
		Initca:     &initca,
	})
	s.Require().NoError(err)

	output := filepath.Join(s.tempDir, "genkey")
	err = s.run(csr.GenKey, &csr.GenKeyFlags{
		CsrProfile: &csrprofile,
		Label:      &label,
		Output:     &output,
	})
	s.Require().NoError(err)
	s.hasTextInFile(output+".csr", `-----BEGIN CERTIFICATE REQUEST-----`)
	s.hasTextInFile(output+"-key.pem", `pkcs11:`)

	csrprofile = "testdata/test_dolly_root-csr.json"
	output = filepath.Join(s.tempDir, "genkeyca")

	err = s.run(csr.GenKey, &csr.GenKeyFlags{
		CsrProfile: &csrprofile,
		Label:      &label,
		Output:     &output,
	})
	s.Require().Error(err)
	s.Equal("CA section only permitted with --initca option", err.Error())

	err = s.run(csr.GenKey, &csr.GenKeyFlags{
		Initca:     &initca,
		CsrProfile: &csrprofile,
		Label:      &label,
		Output:     &output,
	})
	s.Require().NoError(err)
	s.hasTextInFile(output+".csr", `-----BEGIN CERTIFICATE REQUEST-----`)
	s.hasTextInFile(output+"-key.pem", `pkcs11:`)
}

func (s *testSuite) Test_GenCert() {
	err := s.run(csr.GenCert, &csr.GenCertFlags{})
	s.Require().Error(err)
	s.Equal(`CA certificate and key are required`, err.Error())

	caFile := projFolder + "etc/dev/certs/rootca/test_dolly_root_CA.pem"
	caKeyFile := projFolder + "etc/dev/certs/rootca/test_dolly_root_CA-key.pem"

	err = s.run(csr.GenCert, &csr.GenCertFlags{
		CA:    &caFile,
		CAKey: &caKeyFile,
	})
	s.Require().Error(err)
	s.Equal(`read CSR profile: open : no such file or directory`, err.Error())

	label := "with_ts_*"
	missingcsrprofile := "testdata/missing.json"
	err = s.run(csr.GenCert, &csr.GenCertFlags{
		CsrProfile: &missingcsrprofile,
		Label:      &label,
		CA:         &caFile,
		CAKey:      &caKeyFile,
	})
	s.Require().Error(err)
	s.Equal(`read CSR profile: open testdata/missing.json: no such file or directory`, err.Error())

	csrprofile := "testdata/test_dolly_client-csr.json"
	err = s.run(csr.GenCert, &csr.GenCertFlags{
		CsrProfile: &csrprofile,
		Label:      &label,
		CA:         &caFile,
		CAKey:      &caKeyFile,
	})
	s.Require().Error(err)
	s.Equal(`ca-config: {"code":5200,"message":"invalid path"}`, err.Error())

	cacfg := "testdata/ca-config.dev.json"
	err = s.run(csr.GenCert, &csr.GenCertFlags{
		CsrProfile: &csrprofile,
		Label:      &label,
		CAConfig:   &cacfg,
		CA:         &caFile,
		CAKey:      &caKeyFile,
	})
	s.Require().NoError(err)
	s.hasText(`-----BEGIN CERTIFICATE-----`)

	output := filepath.Join(s.tempDir, "genkeyca")
	csrprofile = "testdata/test_dolly_root-csr.json"
	err = s.run(csr.GenCert, &csr.GenCertFlags{
		CsrProfile: &csrprofile,
		Label:      &label,
		Output:     &output,
		CAConfig:   &cacfg,
		CA:         &caFile,
		CAKey:      &caKeyFile,
	})
	s.Require().Error(err)
	s.Equal("CA section only permitted with --initca option, use genkey comand instead", err.Error())

	csrprofile = "testdata/test_dolly_client-csr.json"
	err = s.run(csr.GenCert, &csr.GenCertFlags{
		CsrProfile: &csrprofile,
		Label:      &label,
		Output:     &output,
		CAConfig:   &cacfg,
		CA:         &caFile,
		CAKey:      &caKeyFile,
	})
	s.Require().NoError(err)
	s.hasTextInFile(output+".csr", `-----BEGIN CERTIFICATE REQUEST-----`)
	s.hasTextInFile(output+"-key.pem", `pkcs11:`)
}

func (s *testSuite) Test_SignCert() {
	err := s.run(csr.SignCert, &csr.SignCertFlags{})
	s.Require().Error(err)
	s.Equal(`CA certificate and key are required`, err.Error())

	caFile := projFolder + "etc/dev/certs/rootca/test_dolly_root_CA.pem"
	caKeyFile := projFolder + "etc/dev/certs/rootca/test_dolly_root_CA-key.pem"

	err = s.run(csr.SignCert, &csr.SignCertFlags{
		CA:    &caFile,
		CAKey: &caKeyFile,
	})
	s.Require().Error(err)
	s.Equal(`read CSR profile: open : no such file or directory`, err.Error())

	missingcsr := "testdata/missing.json"
	err = s.run(csr.SignCert, &csr.SignCertFlags{
		Csr:   &missingcsr,
		CA:    &caFile,
		CAKey: &caKeyFile,
	})
	s.Require().Error(err)
	s.Equal(`read CSR profile: open testdata/missing.json: no such file or directory`, err.Error())

	//
	// Generate key and CSR
	//
	csrprofile := "testdata/test_dolly_client-csr.json"
	label := "signcert*"
	output := filepath.Join(s.tempDir, "signcertcsr")
	err = s.run(csr.GenKey, &csr.GenKeyFlags{
		CsrProfile: &csrprofile,
		Label:      &label,
		Output:     &output,
	})
	s.Require().NoError(err)
	s.hasTextInFile(output+".csr", `-----BEGIN CERTIFICATE REQUEST-----`)
	s.hasTextInFile(output+"-key.pem", `pkcs11:`)

	csrfile := output + ".csr"
	err = s.run(csr.SignCert, &csr.SignCertFlags{
		Csr:   &csrfile,
		CA:    &caFile,
		CAKey: &caKeyFile,
	})
	s.Require().Error(err)
	s.Equal(`ca-config: {"code":5200,"message":"invalid path"}`, err.Error())

	cacfg := "testdata/ca-config.dev.json"
	err = s.run(csr.SignCert, &csr.SignCertFlags{
		Csr:      &csrfile,
		CA:       &caFile,
		CAKey:    &caKeyFile,
		CAConfig: &cacfg,
	})
	s.Require().NoError(err)
	s.hasText(`-----BEGIN CERTIFICATE-----`)

	output = filepath.Join(s.tempDir, "signcert")
	err = s.run(csr.SignCert, &csr.SignCertFlags{
		Csr:      &csrfile,
		CA:       &caFile,
		CAKey:    &caKeyFile,
		CAConfig: &cacfg,
		Output:   &output,
	})
	s.Require().NoError(err)
	s.hasTextInFile(output+".pem", `-----BEGIN CERTIFICATE-----`)
	s.hasTextInFile(output+".csr", `-----BEGIN CERTIFICATE REQUEST-----`)
}
