package fileutil_test

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FolderExists(t *testing.T) {
	tmpDir := path.Join(os.TempDir(), "fileutil-test", guid.MustCreate())

	err := os.MkdirAll(tmpDir, os.ModePerm)
	require.NoError(t, err)

	defer os.RemoveAll(tmpDir)

	assert.Error(t, fileutil.FolderExists(""))
	assert.NoError(t, fileutil.FolderExists(tmpDir))

	err = fileutil.FolderExists(tmpDir + "/a")
	require.Error(t, err)
	assert.Equal(t, fmt.Sprintf("stat %s: no such file or directory", tmpDir+"/a"), err.Error())

	err = fileutil.FolderExists("./folders.go")
	require.Error(t, err)
	assert.Equal(t, "not a folder: \"./folders.go\"", err.Error())
}
