package resolve_test

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/go-phorce/dolly/algorithms/guid"
	"github.com/go-phorce/dolly/fileutil/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ResolveDirectory(t *testing.T) {
	tmpDir := path.Join(os.TempDir(), "resolve-test", guid.MustCreate())
	testData := []struct {
		dir     string
		baseDir string
		create  bool
		err     string
	}{
		{
			dir:     "a1/a2",
			baseDir: tmpDir,
			create:  false,
			err:     "no such file or directory",
		},
		{
			dir:     "a1/a2",
			baseDir: tmpDir,
			create:  true,
			err:     "",
		},
		{
			dir:     "a1/a2",
			baseDir: tmpDir,
			create:  false,
			err:     "",
		},
	}

	// Run test
	for idx, v := range testData {
		t.Run(fmt.Sprintf("[%d] %s", idx, v.dir), func(t *testing.T) {
			d, err := resolve.Directory(v.dir, v.baseDir, v.create)
			if v.err != "" {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), v.err))
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, d)
				assert.True(t, strings.HasSuffix(d, v.dir))
			}
		})
	}
}
