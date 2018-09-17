package certutil

import (
	"crypto"
	"encoding/hex"
)

// SHA1Hex returns hex-encoded SHA1
func SHA1Hex(data []byte) (string, error) {
	return HashToHex(crypto.SHA1, data)
}

// SHA256Hex returns hex-encoded SHA256
func SHA256Hex(data []byte) (string, error) {
	return HashToHex(crypto.SHA256, data)
}

// HashToHex returns hex-encoded digest
func HashToHex(hash crypto.Hash, data []byte) (string, error) {
	h := hash.New()
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
