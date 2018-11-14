package fileutil

import (
	"os"

	"github.com/juju/errors"
)

// FolderExists ensures that folder exists
func FolderExists(dir string) error {
	if dir == "" {
		return errors.Errorf("invalid parameter")
	}

	stat, err := os.Stat(dir)
	if err != nil {
		return errors.Trace(err)
	}

	if !stat.IsDir() {
		return errors.Errorf("not a folder: %q", dir)
	}

	return nil
}
