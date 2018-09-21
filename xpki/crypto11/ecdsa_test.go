package crypto11

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var curves = []elliptic.Curve{
	// elliptic.P224(),
	elliptic.P256(),
	// TODO: add for softhsm
	// elliptic.P384(),
	// elliptic.P521(),
	// plus something with explicit parameters
}

func TestNativeECDSA(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Log("Skipping TestNativeECDSA on Mac")
		return
	}

	var err error
	var key *ecdsa.PrivateKey
	for i, curve := range curves {
		if key, err = ecdsa.GenerateKey(curve, rand.Reader); err != nil {
			t.Errorf("[%d] crypto.ecdsa.GenerateKey, curve=%v: %v", i, curve, err)
			return
		}
		testEcdsaSigning(t, key, crypto.SHA1)
		//testEcdsaSigning(t, key, crypto.SHA224)
		testEcdsaSigning(t, key, crypto.SHA256)
		//testEcdsaSigning(t, key, crypto.SHA384)
		testEcdsaSigning(t, key, crypto.SHA512)
	}
}

func TestHardECDSA(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Log("Skipping TestHardECDSA on Mac")
		return
	}
	var err error
	var priv *PKCS11PrivateKeyECDSA
	var key2, key3 crypto.PrivateKey
	var id, label string

	for i, curve := range curves {
		priv, err = p11lib.GenerateECDSAKeyPair(curve)
		require.NoError(t, err, "[%d] crypto11.GenerateECDSAKeyPair, curve=%v: %v", i, curve, err)
		require.NotNil(t, priv)

		testEcdsaSigning(t, priv, crypto.SHA1)
		testEcdsaSigning(t, priv, crypto.SHA224)
		testEcdsaSigning(t, priv, crypto.SHA256)
		testEcdsaSigning(t, priv, crypto.SHA384)
		testEcdsaSigning(t, priv, crypto.SHA512)
		// Get a fresh handle to  the key
		id, label, err = p11lib.Identify(&priv.key.PKCS11Object)
		require.NoError(t, err, "crypto11.Identify: %v", err)

		key2, err = p11lib.FindKeyPair(id, "")
		require.NoError(t, err, "crypto11.FindKeyPair: %v", err)
		testEcdsaSigning(t, key2.(*PKCS11PrivateKeyECDSA), crypto.SHA256)

		key3, err = p11lib.FindKeyPair("", label)
		require.NoError(t, err, "crypto11.FindKeyPair: %v", err)
		testEcdsaSigning(t, key3.(crypto.Signer), crypto.SHA384)
	}
}

func testEcdsaSigning(t *testing.T, key crypto.Signer, hashFunction crypto.Hash) {
	var err error
	var sigDER []byte
	var sig dsaSignature

	plaintext := []byte("sign me with ECDSA")
	h := hashFunction.New()
	h.Write(plaintext)
	plaintextHash := h.Sum([]byte{}) // weird API
	sigDER, err = key.Sign(rand.Reader, plaintextHash, nil)
	require.NoError(t, err, "crypto11.ECDSSign: %v", err)

	err = sig.unmarshalDER(sigDER)
	require.NoError(t, err)

	ecdsaPubkey := key.Public().(crypto.PublicKey).(*ecdsa.PublicKey)
	assert.True(t, ecdsa.Verify(ecdsaPubkey, plaintextHash, sig.R, sig.S), "ECDSA Verify (hash %v): %v", hashFunction, err)
}
