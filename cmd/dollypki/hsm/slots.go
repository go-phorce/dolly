package hsm

import (
	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/juju/errors"
)

// Slots shows hsm slots
func Slots(c ctl.Control, _ interface{}) error {
	keyProv, ok := c.(*cli.Cli).CryptoProv().Default().(cryptoprov.KeyManager)
	if !ok {
		return errors.Errorf("unsupported command for this crypto provider")
	}

	err := keyProv.EnumTokens(false, func(slotID uint, description, label, manufacturer, model, serial string) error {
		c.Printf("Slot: %d\n", slotID)
		c.Printf("  Description:  %s\n", description)
		c.Printf("  Token serial: %s\n", serial)
		c.Printf("  Token label:  %s\n", label)
		return nil
	})
	if err != nil {
		return errors.Annotate(err, "Enum tokens failed")
	}

	return nil
}
