package fileutil

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/juju/errors"
)

const (
	// FileSource specifies to load config from a file
	FileSource = "file://"
	// EnvSource specifies to load config from an environment variable
	EnvSource = "env://"
)

// LoadConfigWithSchema returns a configuration loaded from file:// or env://
// If config does not start with file:// or env://, then the value is returned as is
func LoadConfigWithSchema(config string) (string, error) {
	if strings.HasPrefix(config, FileSource) {
		fn := strings.TrimPrefix(config, FileSource)
		f, err := ioutil.ReadFile(fn)
		if err != nil {
			return config, errors.Trace(err)
		}
		// file content
		config = string(f)
	} else if strings.HasPrefix(config, EnvSource) {
		env := strings.TrimPrefix(config, EnvSource)
		// ENV content
		config = os.Getenv(env)
		if config == "" {
			return "", errors.Errorf("Environment variable %q is not set", env)
		}
	}

	return config, nil
}
