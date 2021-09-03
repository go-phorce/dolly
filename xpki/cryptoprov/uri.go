package cryptoprov

import (
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/juju/errors"
)

// PrivateKeyURI holds PKCS#11 private key information.
//
// A token may be identified either by serial number or label.  If
// both are specified then the first match wins.
type PrivateKeyURI interface {
	// Token manufacturer
	Manufacturer() string

	// Model manufacturer
	Model() string

	// Token serial number
	TokenSerial() string

	// Token label
	TokenLabel() string

	// Key ID
	ID() string
}

type keyURI struct {
	manufacturer string
	model        string
	tokenSerial  string
	tokenLabel   string
	id           string
}

// Token manufacturer
func (k *keyURI) Manufacturer() string {
	return k.manufacturer
}

// Model manufacturer
func (k *keyURI) Model() string {
	return k.model
}

// Token serial number
func (k *keyURI) TokenSerial() string {
	return k.tokenSerial
}

// Token label
func (k *keyURI) TokenLabel() string {
	return k.tokenLabel
}

// Key ID
func (k *keyURI) ID() string {
	return k.id
}

// ParseTokenURI parses a PKCS #11 URI into a PKCS #11
// configuration. Note that the module path will override the module
// name if present.
func ParseTokenURI(uri string) (TokenConfig, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Annotatef(err, "invalid URI: %s", uri)
	}
	if u.Scheme != "pkcs11" {
		return nil, errors.Annotate(ErrInvalidURI, uri)
	}

	c := new(tokenConfig)

	pk11PAttr, err := url.ParseQuery(strings.ReplaceAll(u.Opaque, ";", "&"))
	if err != nil {
		return nil, errors.Annotatef(err, "invalid URI: %s", uri)
	}

	setIfPresent(pk11PAttr, "manufacturer", &c.Man)
	setIfPresent(pk11PAttr, "model", &c.Mod)
	setIfPresent(pk11PAttr, "module-name", &c.Dir)
	setIfPresent(pk11PAttr, "module-path", &c.Dir)
	setIfPresent(pk11PAttr, "token", &c.Label)
	setIfPresent(pk11PAttr, "serial", &c.Serial)
	setIfPresent(pk11PAttr, "pin-value", &c.Pwd)

	var pinSourceURI string
	setIfPresent(pk11PAttr, "pin-source", &pinSourceURI)
	if pinSourceURI == "" {
		return c, nil
	}

	pinURI, err := url.Parse(pinSourceURI)
	if pinURI.Opaque != "" && pinURI.Path == "" {
		pinURI.Path = pinURI.Opaque
	}
	if err != nil || pinURI.Scheme != "file" || pinURI.Path == "" {
		return nil, errors.Annotate(ErrInvalidURI, uri)
	}

	pin, err := ioutil.ReadFile(pinURI.Path)
	if err != nil {
		return nil, errors.Annotate(ErrInvalidURI, uri)
	}

	c.Pwd = strings.TrimSpace(string(pin))
	c.Man = strings.TrimSpace(strings.TrimRight(string(c.Man), "\x00"))
	c.Mod = strings.TrimSpace(string(c.Mod))

	return c, nil
}

// ParsePrivateKeyURI parses a PKCS #11 URI into a key configuration
func ParsePrivateKeyURI(uri string) (PrivateKeyURI, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Annotatef(err, "invalid URI: %s", uri)
	}
	if u.Scheme != "pkcs11" {
		return nil, errors.Annotate(ErrInvalidURI, uri)
	}

	c := new(keyURI)

	pk11PAttr, err := url.ParseQuery(strings.ReplaceAll(u.Opaque, ";", "&"))
	if err != nil {
		return nil, errors.Annotatef(err, "invalid URI: %s", uri)
	}

	setIfPresent(pk11PAttr, "manufacturer", &c.manufacturer)
	setIfPresent(pk11PAttr, "model", &c.model)
	setIfPresent(pk11PAttr, "token", &c.tokenLabel)
	setIfPresent(pk11PAttr, "serial", &c.tokenSerial)
	setIfPresent(pk11PAttr, "id", &c.id)
	var objtype string
	setIfPresent(pk11PAttr, "type", &objtype)
	if objtype != "private" || c.tokenSerial == "" || c.id == "" {
		return nil, errors.Annotate(ErrInvalidPrivateKeyURI, uri)
	}

	c.manufacturer = strings.TrimSpace(strings.TrimRight(string(c.manufacturer), "\x00"))
	c.model = strings.TrimSpace(string(c.model))

	return c, nil
}

func setIfPresent(val url.Values, k string, target *string) {
	sv := val.Get(k)
	if sv != "" {
		*target = sv
	}
}
