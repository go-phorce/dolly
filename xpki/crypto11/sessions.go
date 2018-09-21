package crypto11

import (
	"github.com/juju/errors"
	pkcs11 "github.com/miekg/pkcs11"
)

// Create a new session for a given slot
func (lib *PKCS11Lib) newSession(slot uint) (pkcs11.SessionHandle, error) {
	session, err := lib.Ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return session, nil
}

// withSession run a function with a session
//
// setupSessions must have been called for the slot already,
// otherwise there will be a panic.
func (lib *PKCS11Lib) withSession(slot uint, f func(session pkcs11.SessionHandle) error) error {
	//logger.Tracef("api=withSession, slot=0x%X", slot)
	var session pkcs11.SessionHandle
	var err error
	sessionPool := lib.sessionPools[slot]
	select {
	case session = <-sessionPool:
		// nop
	default:
		if session, err = lib.newSession(slot); err != nil {
			return errors.Trace(err)
		}
	}
	defer func() {
		// TODO better would be to close the session if the pool is full
		sessionPool <- session
	}()
	return f(session)
}

// setupSessions creates the session pool for a given slot,
// if it does not exist already.
func (lib *PKCS11Lib) setupSessions(slot uint, max int) error {
	lib.sessionPoolMutex.Lock()
	defer lib.sessionPoolMutex.Unlock()
	if max <= 0 {
		max = maxSessionsChan
	}
	if _, ok := lib.sessionPools[slot]; !ok {
		lib.sessionPools[slot] = make(chan pkcs11.SessionHandle, max)
	}
	return nil
}
