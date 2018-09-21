package crypto11

import (
	pkcs11 "github.com/miekg/pkcs11"
)

// GenRandom fills data with random bytes generated via PKCS#11 using the default slot.
func (lib *PKCS11Lib) GenRandom(data []byte) (n int, err error) {
	var result []byte
	if err = lib.withSession(lib.Slot.id, func(session pkcs11.SessionHandle) error {
		result, err = lib.Ctx.GenerateRandom(session, len(data))
		return err
	}); err != nil {
		return 0, err
	}
	copy(data, result)
	return len(result), err
}
