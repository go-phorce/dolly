package hsm

import (
	"fmt"
	"os"
	"time"

	"github.com/go-phorce/dolly/cmd/dollypki/cli"
	"github.com/go-phorce/dolly/ctl"
	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/juju/errors"
)

// RmKeyFlags specifies flags for the delete key action
type RmKeyFlags struct {
	// Token specifies slot token
	Token *string
	// Serial specifies slot serial
	Serial *string
	// ID specifies key Id
	ID *string
	// Prefix specifies key label prefix
	Prefix *string
	// Force specifies an option to delete keys without additional confirmation
	Force *bool
}

func ensureRmKeyFlags(f *RmKeyFlags) *RmKeyFlags {
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
	if f.Prefix == nil {
		f.Prefix = &emptyString
	}
	if f.Force == nil {
		f.Force = &falseVal
	}
	return f
}

// RmKey destroys a key
func RmKey(c ctl.Control, p interface{}) error {
	var err error
	flags := ensureRmKeyFlags(p.(*RmKeyFlags))

	keyProv, ok := c.(*cli.Cli).CryptoProv().Default().(cryptoprov.KeyManager)
	if !ok {
		return errors.Errorf("unsupported command for this crypto provider")
	}

	filterSerial := *flags.Serial
	isDefaultSlot := *flags.Serial == ""

	if isDefaultSlot {
		filterSerial = "--@--"
	}

	printSlot := func(slotID uint, description, label, manufacturer, model, serial string) error {
		if isDefaultSlot || serial == filterSerial {
			if *flags.Prefix == "" && *flags.ID == "" {
				return errors.Errorf("either of --prefix and --id must be specified")
			}

			if *flags.Prefix != "" && *flags.ID != "" {
				return errors.Errorf("--prefix and --id should not be specified together, prefix=%s, id=%s", *flags.Prefix, *flags.ID)
			}

			out := c.Writer()
			if *flags.ID != "" {
				err = keyProv.DestroyKeyPairOnSlot(slotID, *flags.ID)
				if err != nil {
					return errors.Annotatef(err, "failed DestroyKeyPairOnSlot: %s", *flags.ID)
				}
				fmt.Fprintf(out, "destroyed key: %s", *flags.ID)
				return nil
			}

			if *flags.Prefix != "" {
				var keysToDestroy []string
				err := keyProv.EnumKeys(slotID, *flags.Prefix, func(id, label, typ, class, currentVersionID string, creationTime *time.Time) error {
					keysToDestroy = append(keysToDestroy, id)
					return nil
				})
				if err != nil {
					fmt.Fprintf(out, "failed to list keys on slot %d: %v\n", slotID, errors.ErrorStack(err))
					return nil
				}

				if len(keysToDestroy) == 0 {
					fmt.Fprintf(out, "no keys found with prefix: %s\n", *flags.Prefix)
					return nil
				}

				fmt.Fprintf(out, "found %d key with prefix: %s\n", len(keysToDestroy), *flags.Prefix)
				for _, keyID := range keysToDestroy {
					fmt.Fprintf(out, "key: %s\n", keyID)
				}

				if *flags.Force {
					err = destroyKeys(c, keyProv, slotID, keysToDestroy)
					if err != nil {
						fmt.Fprintf(out, "failed to destroy keys: [%v]\n", err)
						return nil
					}
				} else {
					isConfirmed, err := ctl.AskForConfirmation(out, os.Stdin, "WARNING: Destroyed keys can not be recovered. Type Y to continue or N to cancel.")
					if err != nil {
						return errors.Annotatef(err, "failed to get a confirmation for prefix: %s", *flags.Prefix)
					}

					if !isConfirmed {
						return nil
					}
					err = destroyKeys(c, keyProv, slotID, keysToDestroy)
					if err != nil {
						fmt.Fprintf(out, "failed to destroy keys: [%v]\n", err)
						return nil
					}
				}
			}
		}
		return nil
	}

	return keyProv.EnumTokens(isDefaultSlot, printSlot)
}

func destroyKeys(c ctl.Control, keyProv cryptoprov.KeyManager, slotID uint, keys []string) error {
	for _, keyID := range keys {
		err := keyProv.DestroyKeyPairOnSlot(slotID, keyID)
		if err != nil {
			return errors.Annotatef(err, "DestroyKeyPairOnSlot failed: slotID=%d, keyID=%s", slotID, keyID)
		}
		fmt.Fprintf(c.Writer(), "destroyed key: %s\n", keyID)
	}
	return nil
}
