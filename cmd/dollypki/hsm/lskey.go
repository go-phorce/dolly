package hsm

import (
	"fmt"
	"time"

	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/pkg/errors"
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

	out := c.Writer()
	printSlot := func(slotID uint, description, label, manufacturer, model, serial string) error {
		if isDefaultSlot || serial == filterSerial || label == filterLabel {
			fmt.Fprintf(out, "Slot: %d\n", slotID)
			fmt.Fprintf(out, "  Description:  %s\n", description)
			fmt.Fprintf(out, "  Token serial: %s\n", serial)
			fmt.Fprintf(out, "  Token label:  %s\n", label)

			count := 0
			err := keyProv.EnumKeys(slotID, *flags.Prefix, func(id, label, typ, class, currentVersionID string, creationTime *time.Time) error {
				count++
				fmt.Fprintf(out, "[%d]\n", count)
				fmt.Fprintf(out, "  Id:    %s\n", id)
				fmt.Fprintf(out, "  Label: %s\n", label)
				fmt.Fprintf(out, "  Type:  %s\n", typ)
				fmt.Fprintf(out, "  Class: %s\n", class)
				fmt.Fprintf(out, "  Version: %s\n", currentVersionID)
				if creationTime != nil {
					fmt.Fprintf(out, "  Created: %s\n", creationTime.Format(time.RFC3339))
				}
				return nil
			})
			if err != nil {
				return errors.WithMessagef(err, "failed to list keys on slot %d", slotID)
			}

			if *flags.Prefix != "" && count == 0 {
				fmt.Fprintf(out, "no keys found with prefix: %s\n", *flags.Prefix)
			}
		}
		return nil
	}

	return keyProv.EnumTokens(isDefaultSlot, printSlot)
}
