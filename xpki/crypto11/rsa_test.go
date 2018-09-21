package crypto11

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rsaSizes = []int{1024, 2048, 4096}

func TestNativeRSA(t *testing.T) {
	var err error
	var key *rsa.PrivateKey
	for _, nbits := range rsaSizes {
		key, err = rsa.GenerateKey(rand.Reader, nbits)
		require.NoError(t, err)

		err = key.Validate()
		require.NoError(t, err)

		testRsaSigning(t, key, nbits)
		testRsaEncryption(t, key, nbits)
	}
}

func TestHardRSA(t *testing.T) {
	var err error
	var priv *PKCS11PrivateKeyRSA
	var key2, key3 crypto.PrivateKey
	var id, label string

	for _, nbits := range rsaSizes {
		priv, err = p11lib.GenerateRSAKeyPair(nbits, Signing)
		require.NoError(t, err)
		require.NotNil(t, priv)

		err = priv.Validate()
		require.NoError(t, err)

		testRsaSigning(t, priv, nbits)
		// Get a fresh handle to  the key
		id, label, err = p11lib.Identify(&priv.key.PKCS11Object)
		require.NoError(t, err)

		key2, err = p11lib.FindKeyPair(id, "")
		require.NoError(t, err)

		testRsaSigning(t, key2.(*PKCS11PrivateKeyRSA), nbits)
		key3, err = p11lib.FindKeyPair("", label)

		require.NoError(t, err)
		testRsaSigning(t, key3.(crypto.Signer), nbits)
	}

	for _, nbits := range rsaSizes {
		priv, err = p11lib.GenerateRSAKeyPair(nbits, Encryption)
		require.NoError(t, err)
		require.NotNil(t, priv)

		err = priv.Validate()
		require.NoError(t, err)

		testRsaEncryption(t, priv, nbits)
		// Get a fresh handle to  the key
		id, label, err = p11lib.Identify(&priv.key.PKCS11Object)
		require.NoError(t, err)

		key2, err = p11lib.FindKeyPair(id, "")
		require.NoError(t, err)
		require.NotNil(t, key2)
	}
}

// TODO: PSS
func testRsaSigning(t *testing.T, key crypto.Signer, nbits int) {
	testRsaSigningPKCS1v15(t, key, crypto.SHA1)
	// testRsaSigningPKCS1v15(t, key, crypto.SHA224)
	testRsaSigningPKCS1v15(t, key, crypto.SHA256)
	// testRsaSigningPKCS1v15(t, key, crypto.SHA384)
	if nbits > 1024 { // key too smol for SHA512 with sLen=hLen
		testRsaSigningPKCS1v15(t, key, crypto.SHA512)
	}
	// testRsaSigningPSS(t, key, crypto.SHA1)
	// testRsaSigningPSS(t, key, crypto.SHA224)
	// testRsaSigningPSS(t, key, crypto.SHA256)
	// testRsaSigningPSS(t, key, crypto.SHA384)
	if nbits > 1024 { // key too smol for SHA512 with sLen=hLen
		// testRsaSigningPSS(t, key, crypto.SHA512)
	}
}

func testRsaSigningPKCS1v15(t *testing.T, key crypto.Signer, hashFunction crypto.Hash) {
	var err error
	var sig []byte

	plaintext := []byte("sign me with PKCS#1 v1.5")
	h := hashFunction.New()
	h.Write(plaintext)
	plaintextHash := h.Sum([]byte{})
	sig, err = key.Sign(rand.Reader, plaintextHash, hashFunction)
	require.NoError(t, err)

	rsaPubkey := key.Public().(crypto.PublicKey).(*rsa.PublicKey)
	err = rsa.VerifyPKCS1v15(rsaPubkey, hashFunction, plaintextHash, sig)
	require.NoError(t, err)
}

func testRsaSigningPSS(t *testing.T, key crypto.Signer, hashFunction crypto.Hash) {
	var err error
	var sig []byte

	plaintext := []byte("sign me with PSS")
	h := hashFunction.New()
	h.Write(plaintext)
	plaintextHash := h.Sum([]byte{})

	pssOptions := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: hashFunction}
	sig, err = key.Sign(rand.Reader, plaintextHash, pssOptions)
	require.NoError(t, err)

	rsaPubkey := key.Public().(crypto.PublicKey).(*rsa.PublicKey)
	err = rsa.VerifyPSS(rsaPubkey, hashFunction, plaintextHash, sig, pssOptions)
	require.NoError(t, err)
}

// TODO: larger HASH, with label
func testRsaEncryption(t *testing.T, key crypto.Decrypter, nbits int) {
	testRsaEncryptionPKCS1v15(t, key)
	testRsaEncryptionOAEP(t, key, crypto.SHA1, []byte{})
	// testRsaEncryptionOAEP(t, key, crypto.SHA224, []byte{})
	if nbits > 1024 { // key too smol for SHA256
		// testRsaEncryptionOAEP(t, key, crypto.SHA256, []byte{})
	}
	//testRsaEncryptionOAEP(t, key, crypto.SHA384, []byte{})
	if nbits > 1024 { // key too smol for SHA512
		// testRsaEncryptionOAEP(t, key, crypto.SHA512, []byte{})
	}

	//
	// With label
	//

	if nbits == 1024 {
		// testRsaEncryptionOAEP(t, key, crypto.SHA1, []byte{1, 2, 3, 4})
	}
	//testRsaEncryptionOAEP(t, key, crypto.SHA224, []byte{5, 6, 7, 8})
	// testRsaEncryptionOAEP(t, key, crypto.SHA256, []byte{9})
	// testRsaEncryptionOAEP(t, key, crypto.SHA384, []byte{10, 11, 12, 13, 14, 15})
	if nbits > 1024 {
		// testRsaEncryptionOAEP(t, key, crypto.SHA512, []byte{16, 17, 18})
	}
}

func testRsaEncryptionPKCS1v15(t *testing.T, key crypto.Decrypter) {
	var err error
	var ciphertext, decrypted []byte

	plaintext := []byte("encrypt me with old and busted crypto")
	rsaPubkey := key.Public().(crypto.PublicKey).(*rsa.PublicKey)
	ciphertext, err = rsa.EncryptPKCS1v15(rand.Reader, rsaPubkey, plaintext)
	require.NoError(t, err)

	decrypted, err = key.Decrypt(rand.Reader, ciphertext, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, bytes.Compare(plaintext, decrypted), "PKCS#1v1.5 Decrypt (nil options): wrong answer")
	options := &rsa.PKCS1v15DecryptOptions{SessionKeyLen: 0}

	decrypted, err = key.Decrypt(rand.Reader, ciphertext, options)
	require.NoError(t, err)

	assert.Equal(t, 0, bytes.Compare(plaintext, decrypted), "PKCS#1v1.5 Decrypt: wrong answer")
}

func testRsaEncryptionOAEP(t *testing.T, key crypto.Decrypter, hashFunction crypto.Hash, label []byte) {
	var err error
	var ciphertext, decrypted []byte

	plaintext := []byte("encrypt me with new hotness")
	h := hashFunction.New()
	rsaPubkey := key.Public().(crypto.PublicKey).(*rsa.PublicKey)
	ciphertext, err = rsa.EncryptOAEP(h, rand.Reader, rsaPubkey, plaintext, label)
	require.NoError(t, err, "OAEP Encrypt")

	options := &rsa.OAEPOptions{Hash: hashFunction, Label: label}
	decrypted, err = key.Decrypt(rand.Reader, ciphertext, options)
	require.NoError(t, err, "OAEP Decrypt")

	assert.Equal(t, 0, bytes.Compare(plaintext, decrypted), "OAEP Decrypt: wrong answer")
}
