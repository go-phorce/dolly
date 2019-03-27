package fileutil_test

import (
	"testing"

	"github.com/go-phorce/dolly/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LoadConfigWithSchema_plain(t *testing.T) {
	c, err := fileutil.LoadConfigWithSchema("test_data")
	require.NoError(t, err)
	assert.Equal(t, "test_data", c)
}

func Test_LoadConfigWithSchema_file(t *testing.T) {
	c, err := fileutil.LoadConfigWithSchema("file://./load.go")
	require.NoError(t, err)
	require.NotEmpty(t, c)
	assert.Contains(t, c, "package fileutil")
}

func Test_LoadConfigWithSchema_env(t *testing.T) {
	_, err := fileutil.LoadConfigWithSchema("env://TEST_ENV")
	require.Error(t, err)

	c, err := fileutil.LoadConfigWithSchema("env://PATH")
	require.NoError(t, err)
	require.NotEmpty(t, c)
}
