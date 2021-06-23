package cryptoprov

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-phorce/dolly/fileutil/resolve"
	"github.com/juju/errors"
)

// TokenConfig holds PKCS#11 configuration information.
//
// A token may be identified either by serial number or label.  If
// both are specified then the first match wins.
//
// Supply this to Configure(), or alternatively use ConfigureFromFile().
type TokenConfig interface {
	// Manufacturer name of the manufacturer
	Manufacturer() string

	// Model name of the device
	Model() string

	// Full path to PKCS#11 library
	Path() string

	// Token serial number
	TokenSerial() string

	// Token label
	TokenLabel() string

	// Pin is a secret to access the token.
	// If it's prefixed with `file:`, then it will be loaded from the file.
	Pin() string

	// Comma separated key=value pair of attributes(e.g. "ServiceName=x,UserName=y")
	Attributes() string
}

type tokenConfig struct {
	Man    string `json:"Manufacturer"`
	Mod    string `json:"Model"`
	Dir    string `json:"Path"`
	Serial string `json:"TokenSerial"`
	Label  string `json:"TokenLabel"`
	Pwd    string `json:"Pin"`
	Attrs  string `json:"Attributes"`
}

// Manufacturer name of the manufacturer
func (c *tokenConfig) Manufacturer() string {
	return c.Man
}

// Model name of the device
func (c *tokenConfig) Model() string {
	return c.Mod
}

// Full path to PKCS#11 library
func (c *tokenConfig) Path() string {
	return c.Dir
}

// Token serial number
func (c *tokenConfig) TokenSerial() string {
	return c.Serial
}

// Token label
func (c *tokenConfig) TokenLabel() string {
	return c.Label
}

// Pin is a secret to access the token.
// If it's prefixed with `file:`, then it will be loaded from the file.
func (c *tokenConfig) Pin() string {
	return c.Pwd
}

// Attributes is list of additional key=value pairs
func (c *tokenConfig) Attributes() string {
	return c.Attrs
}

// LoadTokenConfig loads PKCS#11 token configuration
func LoadTokenConfig(filename string) (TokenConfig, error) {
	cfr, err := os.Open(filename)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer cfr.Close()
	tokenConfig := new(tokenConfig)
	err = json.NewDecoder(cfr).Decode(tokenConfig)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to decode file: %s", filename)
	}

	pin := tokenConfig.Pin()
	if strings.HasPrefix(pin, "file:") {
		pinfile := pin[5:]

		// try to resolve pin file
		cwd, _ := os.Getwd()
		folders := []string{
			"",
			cwd,
			filepath.Dir(filename),
		}

		for _, folder := range folders {
			if resolved, err := resolve.File(pinfile, folder); err == nil {
				pinfile = resolved
				break
			}
			logger.Warningf("reason=resolve, pinfile=%q, basedir=%q", pinfile, folder)
		}

		pb, err := ioutil.ReadFile(pinfile)
		if err != nil {
			return nil, errors.Annotatef(err, "unable to load PIN for configuration: %s", filename)
		}
		tokenConfig.Pwd = string(pb)
	}

	return tokenConfig, nil
}
