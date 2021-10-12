package crypto11

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"time"

	"github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

// CurrentSlotID returns current slot ID
func (p11lib *PKCS11Lib) CurrentSlotID() uint {
	return p11lib.Slot.id
}

// EnumTokens enumerates tokens
func (p11lib *PKCS11Lib) EnumTokens(currentSlotOnly bool, slotInfoFunc func(slotID uint, description, label, manufacturer, model, serial string) error) error {
	if currentSlotOnly {
		return slotInfoFunc(p11lib.Slot.id,
			p11lib.Slot.description,
			p11lib.Slot.label,
			p11lib.Slot.manufacturer,
			p11lib.Slot.model,
			p11lib.Slot.serial)
	}

	list, err := p11lib.TokensInfo()
	if err != nil {
		return errors.WithStack(err)
	}
	for _, ti := range list {
		err = slotInfoFunc(ti.id, ti.description, ti.label, ti.manufacturer, ti.model, ti.serial)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// TokensInfo returns list of tokens
func (p11lib *PKCS11Lib) TokensInfo() ([]*SlotTokenInfo, error) {
	list := []*SlotTokenInfo{}
	slots, err := p11lib.Ctx.GetSlotList(true)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	logger.Tracef("slots=%d", len(slots))

	for _, slotID := range slots {
		si, err := p11lib.Ctx.GetSlotInfo(slotID)
		if err != nil {
			return nil, errors.WithMessagef(err, "GetSlotInfo: %d", slotID)
		}
		ti, err := p11lib.Ctx.GetTokenInfo(slotID)
		if err != nil {
			logger.Errorf(
				"reason=GetTokenInfo, slotID=%d, ManufacturerID=%q, SlotDescription=%q, err=[%+v]",
				slotID,
				si.ManufacturerID,
				si.SlotDescription,
				err,
			)
		} else if ti.SerialNumber != "" || ti.Label != "" {
			list = append(list, &SlotTokenInfo{
				id:           slotID,
				description:  si.SlotDescription,
				label:        ti.Label,
				manufacturer: strings.TrimSpace(ti.ManufacturerID),
				model:        strings.TrimSpace(ti.Model),
				serial:       ti.SerialNumber,
				flags:        ti.Flags,
			})

		}
	}
	return list, nil
}

// EnumKeys returns lists of keys on the slot
func (p11lib *PKCS11Lib) EnumKeys(slotID uint, prefix string, keyInfoFunc func(id, label, typ, class, currentVersionID string, creationTime *time.Time) error) error {
	sh, err := p11lib.Ctx.OpenSession(slotID, pkcs11.CKF_SERIAL_SESSION)
	if err != nil {
		return errors.WithMessagef(err, "OpenSession on slot %d", slotID)
	}
	defer p11lib.Ctx.CloseSession(sh)

	keys, err := p11lib.ListKeys(sh, pkcs11.CKO_PRIVATE_KEY, ^uint(0))
	if err != nil {
		return errors.WithStack(err)
	}

	for _, obj := range keys {
		attributes := []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_ID, 0),
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, 0),
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, 0),
			pkcs11.NewAttribute(pkcs11.CKA_CLASS, 0),
		}
		if attributes, err = p11lib.Ctx.GetAttributeValue(sh, obj, attributes); err != nil {
			return errors.WithMessagef(err, "GetAttributeValue on key")
		}

		keyLabel := string(attributes[1].Value)
		if prefix != "" && !strings.HasPrefix(keyLabel, prefix) {
			continue
		}
		keyID := string(attributes[0].Value)
		err := keyInfoFunc(
			keyID,
			keyLabel,
			KeyTypeNames[BytesToUlong(attributes[2].Value)],
			ObjectClassNames[BytesToUlong(attributes[3].Value)],
			"",
			nil,
		)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// DestroyKeyPairOnSlot destroys key pair
func (p11lib *PKCS11Lib) DestroyKeyPairOnSlot(slotID uint, keyID string) error {
	var err error
	session, err := p11lib.Ctx.OpenSession(slotID, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		return errors.WithMessagef(err, "OpenSession on slot %d", slotID)
	}
	defer p11lib.Ctx.CloseSession(session)

	logger.Tracef("slot=0x%X, id=%q", slotID, keyID)

	id := []byte(keyID)

	var privHandle, pubHandle pkcs11.ObjectHandle
	if privHandle, err = p11lib.findKey(session, keyID, "", pkcs11.CKO_PRIVATE_KEY, ^uint(0)); err != nil {
		logger.Warningf("reason=not_found, type=CKO_PRIVATE_KEY, err=[%+v]", err)
	}
	if pubHandle, err = p11lib.findKey(session, keyID, "", pkcs11.CKO_PUBLIC_KEY, ^uint(0)); err != nil {
		logger.Warningf("reason=not_found, type=CKO_PUBLIC_KEY, err=[%+v]", err)
	}

	if privHandle != 0 {
		err = p11lib.Ctx.DestroyObject(session, privHandle)
		if err != nil {
			return errors.WithStack(err)
		}
		logger.Infof("type=CKO_PRIVATE_KEY, slot=0x%X, id=%q", slotID, keyID)
	}

	if pubHandle != 0 {
		err = p11lib.Ctx.DestroyObject(session, pubHandle)
		if err != nil {
			return errors.WithStack(err)
		}
		logger.Infof("type=CKO_PUBLIC_KEY, slot=0x%X, id=%q", slotID, string(id))
	}
	return nil
}

// KeyInfo retrieves info about key with the specified id
func (p11lib *PKCS11Lib) KeyInfo(slotID uint, keyID string, includePublic bool, keyInfoFunc func(id, label, typ, class, currentVersionID, pubKey string, creationTime *time.Time) error) error {
	logger.Tracef("slot=0x%X, id=%q", slotID, keyID)
	var err error
	session, err := p11lib.Ctx.OpenSession(slotID, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		return errors.WithMessagef(err, "OpenSession on slot %d", slotID)
	}
	defer p11lib.Ctx.CloseSession(session)

	logger.Tracef("slot=0x%X, id=%q", slotID, keyID)

	var privHandle pkcs11.ObjectHandle
	if privHandle, err = p11lib.findKey(session, keyID, "", pkcs11.CKO_PRIVATE_KEY, ^uint(0)); err != nil {
		logger.Warningf("reason=not_found, type=CKO_PRIVATE_KEY, err=[%+v]", err)
	}

	attributes := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_ID, 0),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, 0),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, 0),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, 0),
	}
	if attributes, err = p11lib.Ctx.GetAttributeValue(session, privHandle, attributes); err != nil {
		return errors.WithMessagef(err, "GetAttributeValue on key")
	}

	keyLabel := string(attributes[1].Value)
	keyID = string(attributes[0].Value)

	pubKey := ""
	if includePublic {
		pubKey, err = p11lib.getPublicKeyPEM(slotID, keyID)
		if err != nil {
			return errors.WithMessagef(err, "reason='failed on GetPublicKey', slotID=%d, keyID=%q", slotID, keyID)
		}
	}

	err = keyInfoFunc(
		keyID,
		keyLabel,
		KeyTypeNames[BytesToUlong(attributes[2].Value)],
		ObjectClassNames[BytesToUlong(attributes[3].Value)],
		"",
		pubKey,
		nil,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// getPublicKeyPEM retrieves public key for the specified key
func (p11lib *PKCS11Lib) getPublicKeyPEM(slotID uint, keyID string) (string, error) {
	priv, err := p11lib.FindKeyPairOnSlot(slotID, keyID, "")
	if err != nil {
		return "", errors.WithMessagef(err, "reason=FindKeyPairOnSlot, slotID=%d, uriID=%s",
			slotID, keyID)
	}

	pub, err := ConvertToPublic(priv)
	if err != nil {
		return "", errors.WithStack(err)
	}

	pemKey, err := EncodePublicKeyToPEM(pub)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(pemKey), nil
}

// EncodePublicKeyToPEM returns PEM encoded public key
func EncodePublicKeyToPEM(pubKey crypto.PublicKey) (asn1Bytes []byte, err error) {
	asn1Bytes, err = x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var pemkey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}

	b := bytes.NewBuffer([]byte{})

	err = pem.Encode(b, pemkey)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return b.Bytes(), nil
}
