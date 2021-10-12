package cryptoprov

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/pkg/errors"
)

// GcmEncrypt returns encrypted blob with GCM cipher
func GcmEncrypt(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	nonce := certutil.Random(gcm.NonceSize())

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// GcmDecrypt returns decrypted blob with GCM cipher
func GcmDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return plain, nil
}
