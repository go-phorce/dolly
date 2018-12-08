package cryptoprov_test

import (
	"path/filepath"
	"testing"

	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/go-phorce/dolly/xpki/cryptoprov/testprov"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inmemloader(_ cryptoprov.TokenConfig) (cryptoprov.Provider, error) {
	p, err := testprov.Init()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return p, nil
}

func Test_LoadProvider(t *testing.T) {
	_, _ = cryptoprov.Unregister("SoftHSM")

	cfgfile := filepath.Join(projFolder, "etc/dev/softhsm_unittest.json")
	_, err := cryptoprov.LoadProvider(cfgfile)
	assert.Error(t, err)

	err = cryptoprov.Register("SoftHSM", cryptoprov.Crypto11Loader)
	assert.NoError(t, err)
	defer cryptoprov.Unregister("SoftHSM")

	p, err := cryptoprov.LoadProvider(cfgfile)
	require.NoError(t, err)

	assert.Equal(t, "SoftHSM", p.Manufacturer())
}

func Test_Load(t *testing.T) {
	_ = cryptoprov.Register("SoftHSM", cryptoprov.Crypto11Loader)
	defer cryptoprov.Unregister("SoftHSM")
	_ = cryptoprov.Register("inmem", inmemloader)
	defer cryptoprov.Unregister("inmem")

	cp, err := cryptoprov.Load(
		filepath.Join(projFolder, "etc/dev/softhsm_unittest.json"),
		[]string{filepath.Join(projFolder, "xpki/cryptoprov/testdata/inmem_testprov.json")})
	require.NoError(t, err)
	assert.Equal(t, "SoftHSM", cp.Default().Manufacturer())
}
