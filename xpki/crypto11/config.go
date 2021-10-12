package crypto11

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-phorce/dolly/xlog"
	pkcs11 "github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xpki", "crypto11")

const maxSessionsChan = 1024

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

type config struct {
	Man    string `json:"Manufacturer"`
	Mod    string `json:"Model"`
	Dir    string `json:"Path"`
	Serial string `json:"TokenSerial"`
	Label  string `json:"TokenLabel"`
	Pwd    string `json:"Pin"`
	Attrs  string `json:"Attributes"`
}

// Manufacturer name of the manufacturer
func (c *config) Manufacturer() string {
	return c.Man
}

// Model name of the device
func (c *config) Model() string {
	return c.Mod
}

// Full path to PKCS#11 library
func (c *config) Path() string {
	return c.Dir
}

// Token serial number
func (c *config) TokenSerial() string {
	return c.Serial
}

// Token label
func (c *config) TokenLabel() string {
	return c.Label
}

// Pin is a secret to access the token.
// If it's prefixed with `file:`, then it will be loaded from the file.
func (c *config) Pin() string {
	return c.Pwd
}

// Comma separated key=value pair of attributes(e.g. "ServiceName=x,UserName=y")
func (c *config) Attributes() string {
	return c.Attrs
}

// Init configures PKCS#11 from a TokenConfig, and opens default slot
func Init(config TokenConfig) (*PKCS11Lib, error) {
	var err error
	var flags uint

	lib := &PKCS11Lib{
		Config:       config,
		Slot:         nil,
		sessionPools: map[uint]chan pkcs11.SessionHandle{},
	}

	lib.Ctx = pkcs11.New(config.Path())
	if lib.Ctx == nil {
		return nil, errors.WithMessage(errCannotOpenPKCS11, config.Path())
	}
	if err = lib.Ctx.Initialize(); err != nil && err.(pkcs11.Error) != pkcs11.CKR_CRYPTOKI_ALREADY_INITIALIZED {
		return nil, errors.WithMessagef(err, "initialize PKCS#11 library: %s", config.Path())
	}

	slots, err := lib.TokensInfo()
	if err != nil {
		return nil, errors.WithMessage(err, "TokensInfo failed")
	}

	for _, slot := range slots {
		logger.Tracef("state=search, slot=%d, serial=%q, label=%q", slot.id, slot.serial, slot.label)
		if slot.serial == config.TokenSerial() || slot.label == config.TokenLabel() {
			lib.Slot = slot
			flags = slot.flags
			logger.Infof("state=found, slot=%d, serial=%q, label=%q", slot.id, slot.serial, slot.label)
			break
		}
	}

	if lib.Slot == nil {
		return nil, errors.WithStack(errTokenNotFound)
	}

	lib.sessionPools[lib.Slot.id] = make(chan pkcs11.SessionHandle, maxSessionsChan)

	if err = lib.withSession(lib.Slot.id, func(session pkcs11.SessionHandle) error {
		if flags&pkcs11.CKF_LOGIN_REQUIRED != 0 {
			err = lib.Ctx.Login(session, pkcs11.CKU_USER, config.Pin())
			if err != nil && err.(pkcs11.Error) != pkcs11.CKR_USER_ALREADY_LOGGED_IN {
				return errors.WithMessage(err, "login into PKCS#11 token")
			}
		}
		return nil
	}); err != nil {
		return nil, errors.WithMessage(err, "open PKCS#11 session")
	}
	return lib, nil
}

// ConfigureFromFile configures PKCS#11 from a name configuration file.
//
// Configuration files are a JSON representation of the PKCSConfig object.
// The return value is as for Configure().
//
// Note that if CRYPTO11_CONFIG_PATH is set in the environment,
// configuration will be read from that file, overriding any later
// runtime configuration.
func ConfigureFromFile(configLocation string) (*PKCS11Lib, error) {
	cfg, err := LoadTokenConfig(configLocation)
	if err != nil {
		return nil, errors.WithMessagef(err, "load p11 config: %q", configLocation)
	}
	lib, err := Init(cfg)
	if err != nil {
		return nil, errors.WithMessagef(err, "initialize p11 config: %q", configLocation)
	}
	return lib, nil
}

// LoadTokenConfig loads PKCS#11 token configuration
func LoadTokenConfig(filename string) (TokenConfig, error) {
	cfr, err := os.Open(filename)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer cfr.Close()
	tokenConfig := new(config)
	err = json.NewDecoder(cfr).Decode(tokenConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	pin := tokenConfig.Pin()
	if strings.HasPrefix(pin, "file:") {
		pb, err := ioutil.ReadFile(strings.TrimLeft(pin, "file:"))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		tokenConfig.Pwd = string(pb)
	}

	return tokenConfig, nil
}
