package certutil

import (
	"crypto"
	"encoding/hex"
	"hash"
	"strings"

	"github.com/juju/errors"
)

var hashToStr = map[crypto.Hash]string{
	crypto.MD4:         "MD4",
	crypto.MD5:         "MD5",
	crypto.SHA1:        "SHA1",
	crypto.SHA224:      "SHA224",
	crypto.SHA256:      "SHA256",
	crypto.SHA384:      "SHA384",
	crypto.SHA512:      "SHA512",
	crypto.MD5SHA1:     "MD5SHA1",
	crypto.RIPEMD160:   "RIPEMD160",
	crypto.SHA3_224:    "SHA3_224",
	crypto.SHA3_256:    "SHA3_256",
	crypto.SHA3_384:    "SHA3_384",
	crypto.SHA3_512:    "SHA3_512",
	crypto.SHA512_224:  "SHA512_224",
	crypto.SHA512_256:  "SHA512_256",
	crypto.BLAKE2s_256: "BLAKE2s_256",
	crypto.BLAKE2b_256: "BLAKE2b_256",
	crypto.BLAKE2b_384: "BLAKE2b_384",
	crypto.BLAKE2b_512: "BLAKE2b_512",
}

var strToHash = map[string]crypto.Hash{
	"MD5":        crypto.MD5,
	"SHA1":       crypto.SHA1,
	"SHA224":     crypto.SHA224,
	"SHA256":     crypto.SHA256,
	"SHA384":     crypto.SHA384,
	"SHA512":     crypto.SHA512,
	"SHA512_224": crypto.SHA512_224,
	"SHA512_256": crypto.SHA512_256,
	/* NOT SUPPORTED YET
	"MD5SHA1":    crypto.MD5SHA1,
	"RIPEMD160":  crypto.RIPEMD160,
	"SHA3_224":    crypto.SHA3_224,
	"SHA3_256":    crypto.SHA3_256,
	"SHA3_384":    crypto.SHA3_384,
	"SHA3_512":    crypto.SHA3_512,
	"BLAKE2s_256": crypto.BLAKE2s_256,
	"BLAKE2b_256": crypto.BLAKE2b_256,
	"BLAKE2b_384": crypto.BLAKE2b_384,
	"BLAKE2b_512": crypto.BLAKE2b_512,
	*/
}

// HashAlgoToStr converts hash algorithm to string
func HashAlgoToStr(hash crypto.Hash) string {
	return hashToStr[hash]
}

// StrToHashAlgo converts string to hash algorithm
func StrToHashAlgo(algo string) crypto.Hash {
	return strToHash[strings.ToUpper(algo)]
}

// NewHash returns hash instance
func NewHash(algo string) (hash.Hash, error) {
	if h, ok := strToHash[strings.ToUpper(algo)]; ok {
		return h.New(), nil
	}

	return nil, errors.Errorf("unsupported hash algorithm: %q", algo)
}

// ParseHexDigestWithPrefix parses encoded digest in {alg}:{hex} format
func ParseHexDigestWithPrefix(digest string) (hash.Hash, []byte, error) {
	digestParts := strings.Split(digest, ":")
	if len(digestParts) != 2 || len(digestParts[0]) == 0 || len(digestParts[1]) == 0 {
		return nil, nil, errors.Errorf("invalid digest format: %q", digest)
	}

	h, err := NewHash(digestParts[0])
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	d, err := hex.DecodeString(digestParts[1])
	if err != nil {
		return nil, nil, errors.Annotatef(err, "failed to decode digest %q", digestParts[1])
	}

	return h, d, nil
}

// Digest returns computed digest bytes
func Digest(hash crypto.Hash, data []byte) []byte {
	h := hash.New()
	_, err := h.Write(data)
	if err != nil {
		logger.Panicf("digest failed: %s", errors.Trace(err))
	}
	return h.Sum(nil)
}

// SHA1 returns SHA1 digest
func SHA1(data []byte) []byte {
	return Digest(crypto.SHA1, data)
}

// SHA256 returns SHA256 digest
func SHA256(data []byte) []byte {
	return Digest(crypto.SHA256, data)
}

// SHA1Hex returns hex-encoded SHA1
func SHA1Hex(data []byte) string {
	return HashToHex(crypto.SHA1, data)
}

// SHA256Hex returns hex-encoded SHA256
func SHA256Hex(data []byte) string {
	return HashToHex(crypto.SHA256, data)
}

// HashToHex returns hex-encoded digest
func HashToHex(hash crypto.Hash, data []byte) string {
	return hex.EncodeToString(Digest(hash, data))
}
