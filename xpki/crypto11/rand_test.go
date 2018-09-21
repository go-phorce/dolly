package crypto11

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomReader(t *testing.T) {
	var a [32768]byte
	var err error
	var n int
	for _, size := range []int{1, 16, 32, 256, 347, 4096, 32768} {
		n, err = p11lib.GenRandom(a[:size])
		require.NoError(t, err, "crypto11.PKCS11RandRead.Read: %v", err)

		assert.Equal(t, size, n, "crypto11.PKCS11RandRead.Read: only got %d bytes expected %d", n, size)
	}
}
