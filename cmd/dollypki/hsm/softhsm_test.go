package hsm_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/cmd/dollypki/hsm"
	"github.com/go-phorce/dolly/cmd/dollypki/testsuite"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type softHsmSuite struct {
	testsuite.Suite

	tmpdir string
}

func Test_SoftHsmSuite(t *testing.T) {
	cryptoprov.Register("SoftHSM", cryptoprov.Crypto11Loader)

	s := new(softHsmSuite)

	s.tmpdir = filepath.Join(os.TempDir(), "/tests/dolly", "softhsm")
	err := os.MkdirAll(s.tmpdir, 0777)
	require.NoError(t, err)
	defer os.RemoveAll(s.tmpdir)

	s.WithAppFlags([]string{"--hsm-cfg", "/tmp/dolly/softhsm_unittest.json"})
	suite.Run(t, s)
}

func (s *softHsmSuite) SetupSuite() {
	s.Suite.SetupSuite()
	err := s.Cli.EnsureCryptoProvider()
	s.Require().NoError(err)
}

func (s *softHsmSuite) Test_Slots() {
	err := s.Run(hsm.Slots, nil)
	s.NoError(err)
	s.HasText(`Slot:`)
}

func (s *softHsmSuite) Test_KeyInfo() {
	token := ""
	serial := ""
	id := ""
	withPub := true

	flags := hsm.KeyInfoFlags{
		Token:  &token,
		Serial: &serial,
		ID:     &id,
		Public: &withPub,
	}

	// 1. with default
	err := s.Run(hsm.KeyInfo, &flags)
	s.NoError(err)
	s.HasText(`Version: `)

	// 2. with non-existing Key ID
	id = "123456vlwbeoivbwerfvbwefvwfev"
	flags.ID = &id
	err = s.Run(hsm.KeyInfo, &flags)
	s.NoError(err)
	s.HasText(`failed to get key info on slot`)

	// 3. with non-existing Serial
	serial = "123456vlwbeoivbwe46747"
	id = ""
	flags.ID = &id
	flags.Serial = &serial
	err = s.Run(hsm.KeyInfo, &flags)
	s.NoError(err)
	s.HasText(`no slots found with serial`)
}

func (s *softHsmSuite) Test_LsKeyFlags() {
	token := ""
	serial := ""
	prefix := ""

	flags := hsm.LsKeyFlags{
		Token:  &token,
		Serial: &serial,
		Prefix: &prefix,
	}

	// 1. with default
	err := s.Run(hsm.Keys, &flags)
	s.NoError(err)
	s.HasText(`Version:`)

	// 2. with prefix
	prefix = "123456vlwbeo6959579wefvwfev"
	flags.Prefix = &prefix
	err = s.Run(hsm.Keys, &flags)
	s.NoError(err)
	s.HasText(`no keys found with prefix: 123456vlwbeo6959579wefvwfev`)
}

func (s *softHsmSuite) Test_RmKey() {
	token := ""
	serial := ""
	id := ""
	prefix := ""
	force := false

	flags := hsm.RmKeyFlags{
		Token:  &token,
		Serial: &serial,
		ID:     &id,
		Prefix: &prefix,
		Force:  &force,
	}

	// with default
	err := s.Run(hsm.RmKey, &flags)
	s.Require().Error(err)
	s.Equal("either of --prefix and --id must be specified", err.Error())

	// with mutual exclusive flags
	id = "123456vlwbeoivbwerfvbwefvwfev"
	prefix = "123456vlwbeoivbwerfvbwefvwfev"
	flags.ID = &id
	flags.Prefix = &prefix

	err = s.Run(hsm.RmKey, &flags)
	s.Require().Error(err)
	s.Equal("--prefix and --id should not be specified together", err.Error())

	// with ID
	id = "123456vlwbeoivbwerfvbwefvwfev"
	prefix = ""
	flags.ID = &id
	flags.Prefix = &prefix

	err = s.Run(hsm.RmKey, &flags)
	s.Require().NoError(err)
	s.HasText(`destroyed key: 123456vlwbeoivbwerfvbwefvwfev`)

	// with prefix
	id = ""
	prefix = "58576857856785678567" // non existing
	flags.ID = &id
	flags.Prefix = &prefix

	err = s.Run(hsm.RmKey, &flags)
	s.Require().NoError(err)
	s.HasText("no keys found with prefix: 58576857856785678567\n")
	s.HasNoText(` Type 'yes' to continue or 'no' to cancel. [y/n]:`)
	s.HasNoText(`destroyed key:`)
}

func (s *softHsmSuite) Test_GenKey() {
	err := s.Run(hsm.GenKey, &hsm.GenKeyFlags{})
	s.Require().Error(err)
	s.Equal(`unsupported purpose: ""`, err.Error())

	output := filepath.Join(s.tmpdir, guid.MustCreate())
	err = ioutil.WriteFile(output, []byte{1, 2, 3, 4}, 0664)

	err = s.Run(hsm.GenKey, &hsm.GenKeyFlags{
		Output: &output,
	})
	s.Require().Error(err)
	s.Equal(fmt.Sprintf("%q file exists, specify --force flag to override", output), err.Error())

	encrypt := "e"
	label := "TestHsmGenKey"
	algo := "algo"
	rsa := "rsa"
	size1024 := 1024
	size2048 := 2048

	err = s.Run(hsm.GenKey, &hsm.GenKeyFlags{
		Purpose: &encrypt,
		Label:   &label,
	})
	s.Require().Error(err)
	s.Equal(`invalid algorithm: `, err.Error())

	err = s.Run(hsm.GenKey, &hsm.GenKeyFlags{
		Algo:    &algo,
		Purpose: &encrypt,
		Label:   &label,
	})
	s.Require().Error(err)
	s.Equal(`invalid algorithm: algo`, err.Error())

	err = s.Run(hsm.GenKey, &hsm.GenKeyFlags{
		Size:    &size1024,
		Algo:    &rsa,
		Purpose: &encrypt,
		Label:   &label,
	})
	s.Error(err)
	s.Equal(`validate RSA key: RSA key is too weak: 1024`, err.Error())

	label = "TestHsmGenKey*"
	err = s.Run(hsm.GenKey, &hsm.GenKeyFlags{
		Size:    &size2048,
		Algo:    &rsa,
		Purpose: &encrypt,
		Label:   &label,
	})
	s.NoError(err)

	force := true
	err = s.Run(hsm.GenKey, &hsm.GenKeyFlags{
		Size:    &size2048,
		Algo:    &rsa,
		Purpose: &encrypt,
		Label:   &label,
		Force:   &force,
		Output:  &output,
	})
	s.NoError(err)

}
