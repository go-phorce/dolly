package cryptoprov_test

import (
	"testing"

	"github.com/go-phorce/dolly/xpki/certutil"

	"github.com/go-phorce/dolly/xpki/cryptoprov"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GcmEncrypt_short_key(t *testing.T) {
	plainSrc := []byte("data to protect")
	key := certutil.Random(15)

	_, err := cryptoprov.GcmEncrypt(plainSrc, key)
	require.Error(t, err)
}

func Test_GcmEncrypt(t *testing.T) {
	plainSrc := []byte("data to protect")
	key := certutil.Random(32)

	encrypted, err := cryptoprov.GcmEncrypt(plainSrc, key)
	require.NoError(t, err)

	decrypted, err := cryptoprov.GcmDecrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, plainSrc, decrypted)

	_, err = cryptoprov.GcmDecrypt(encrypted[1:], key)
	require.Error(t, err)

	_, err = cryptoprov.GcmDecrypt(encrypted, certutil.Random(32))
	require.Error(t, err)

	_, err = cryptoprov.GcmDecrypt(encrypted[:2], key)
	require.Error(t, err)
}
