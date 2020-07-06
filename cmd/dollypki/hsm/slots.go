package hsm

import (
	"fmt"

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
	out := c.Writer()
	err := keyProv.EnumTokens(false, func(slotID uint, description, label, manufacturer, model, serial string) error {
		fmt.Fprintf(out, "Slot: %d\n", slotID)
		fmt.Fprintf(out, "  Description:  %s\n", description)
		fmt.Fprintf(out, "  Token serial: %s\n", serial)
		fmt.Fprintf(out, "  Token label:  %s\n", label)
		return nil
	})
	if err != nil {
		return errors.Annotate(err, "unable to list slots")
	}

	return nil
}
