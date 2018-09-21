package crypto11

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LoadConfigTwice(t *testing.T) {
	f, err := findConfigFilePath("softhsm_unittest.json")
	require.NoError(t, err)

	c, err := LoadTokenConfig(f)
	require.NoError(t, err, "failed to load HSM config in dir: %s", f)

	assert.NotEmpty(t, c.Path)
	assert.NotEmpty(t, c.Pin)
	assert.NotEmpty(t, c.TokenLabel)

	p11, err := Init(c)
	require.NoError(t, err)
	require.NotNil(t, p11)

	p11_2, err := Init(c)
	require.NoError(t, err)
	require.NotNil(t, p11_2)
}
