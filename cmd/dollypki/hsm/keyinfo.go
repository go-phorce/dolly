package hsm

import (
	"fmt"
	"time"

	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/juju/errors"
)

// KeyInfoFlags specifies flags for the key info action
type KeyInfoFlags struct {
	// Token specifies slot token
	Token *string
	// Serial specifies slot serial
	Serial *string
	// ID specifies key Id
	ID *string
	// Prefix specifies if public key should be listed
	Public *bool
}

func ensureKeyInfoFlags(f *KeyInfoFlags) *KeyInfoFlags {
	var (
		emptyString = ""
		falseVal    = false
	)
	if f.Token == nil {
		f.Token = &emptyString
	}
	if f.Serial == nil {
		f.Serial = &emptyString
	}
	if f.ID == nil {
		f.ID = &emptyString
	}
	if f.Public == nil {
		f.Public = &falseVal
	}
	return f
}

// KeyInfo retrieves info about a key
func KeyInfo(c ctl.Control, p interface{}) error {
	flags := ensureKeyInfoFlags(p.(*KeyInfoFlags))

	keyProv, ok := c.(*cli.Cli).CryptoProv().Default().(cryptoprov.KeyManager)
	if !ok {
		return errors.Errorf("unsupported command for this crypto provider")
	}

	filterSerial := *flags.Serial
	isDefaultSlot := filterSerial == ""

	if isDefaultSlot {
		filterSerial = "--@--"
	}

	out := c.Writer()

	slotCount := 0
	printSlot := func(slotID uint, description, label, manufacturer, model, serial string) error {
		if isDefaultSlot || serial == filterSerial {
			slotCount++

			fmt.Fprintf(out, "Slot: %d\n", slotID)
			fmt.Fprintf(out, "  Description:  %s\n", description)
			fmt.Fprintf(out, "  Token serial: %s\n", serial)

			count := 0
			err := keyProv.KeyInfo(slotID, *flags.ID, *flags.Public, func(id, label, typ, class, currentVersionID, pubKey string, creationTime *time.Time) error {
				count++
				fmt.Fprintf(out, "[%d]\n", count)
				fmt.Fprintf(out, "  Id:    %s\n", id)
				fmt.Fprintf(out, "  Label: %s\n", label)
				fmt.Fprintf(out, "  Type:  %s\n", typ)
				fmt.Fprintf(out, "  Class: %s\n", class)
				fmt.Fprintf(out, "  Version: %s\n", currentVersionID)
				fmt.Fprintf(out, "  Public key: \n%s\n", pubKey)
				if creationTime != nil {
					fmt.Fprintf(out, "  Created: %s\n", creationTime.Format(time.RFC3339))
				}
				return nil
			})
			if err != nil {
				fmt.Fprintf(out, "failed to get key info on slot %d, keyID %s: %v\n", slotID, *flags.ID, err)
				return nil
			}

			if count == 0 {
				fmt.Fprintf(out, "no keys found with ID: %s\n", *flags.ID)
			}
		}
		return nil
	}

	keyProv.EnumTokens(isDefaultSlot, printSlot)
	if slotCount == 0 {
		fmt.Fprintf(out, "no slots found with serial: %s\n", filterSerial)
	}

	return nil
}
