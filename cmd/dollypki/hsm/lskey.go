package hsm

import (
	"time"

	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/juju/errors"
)

// LsKeyFlags specifies flags for the Keys action
type LsKeyFlags struct {
	// Token specifies slot token
	Token *string
	// Serial specifies slot serial
	Serial *string
	// Prefix specifies key label prefix
	Prefix *string
}

func ensureLsKeyFlags(f *LsKeyFlags) *LsKeyFlags {
	var (
		emptyString = ""
	)
	if f.Token == nil {
		f.Token = &emptyString
	}
	if f.Serial == nil {
		f.Serial = &emptyString
	}
	if f.Prefix == nil {
		f.Prefix = &emptyString
	}
	return f
}

// Keys shows keys
func Keys(c ctl.Control, p interface{}) error {
	flags := ensureLsKeyFlags(p.(*LsKeyFlags))

	keyProv, ok := c.(*cli.Cli).CryptoProv().Default().(cryptoprov.KeyManager)
	if !ok {
		return errors.Errorf("unsupported command for this crypto provider")
	}

	isDefaultSlot := *flags.Serial == "" && *flags.Token == ""
	filterSerial := *flags.Serial
	if filterSerial == "" {
		filterSerial = "--@--"
	}
	filterLabel := *flags.Token
	if filterLabel == "" {
		filterLabel = "--@--"
	}

	printSlot := func(slotID uint, description, label, manufacturer, model, serial string) error {
		if isDefaultSlot || serial == filterSerial || label == filterLabel {
			c.Printf("Slot: %d\n", slotID)
			c.Printf("  Description:  %s\n", description)
			c.Printf("  Token serial: %s\n", serial)
			c.Printf("  Token label:  %s\n", label)

			count := 0
			err := keyProv.EnumKeys(slotID, *flags.Prefix, func(id, label, typ, class, currentVersionID string, creationTime *time.Time) error {
				count++
				c.Printf("[%d]\n", count)
				c.Printf("  Id:    %s\n", id)
				c.Printf("  Label: %s\n", label)
				c.Printf("  Type:  %s\n", typ)
				c.Printf("  Class: %s\n", class)
				c.Printf("  CurrentVersionID:  %s\n", currentVersionID)
				if creationTime != nil {
					c.Printf("  CreationTime: %s\n", creationTime.Format(time.RFC3339))
				}
				return nil
			})
			if err != nil {
				c.Printf("failed to list keys on slot %d: %v\n", slotID, err)
				return nil
			}

			if *flags.Prefix != "" && count == 0 {
				c.Printf("no keys found with prefix: %s\n", *flags.Prefix)
			}
		}
		return nil
	}

	return keyProv.EnumTokens(isDefaultSlot, printSlot)
}
