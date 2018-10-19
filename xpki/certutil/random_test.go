package certutil_test

import (
	"testing"

	"github.com/go-phorce/dolly/xpki/certutil"
	"github.com/stretchr/testify/assert"
)

func Test_RandomString(t *testing.T) {
	tcases := []int{1, 8, 13, 96, 512, 1024}
	for _, tc := range tcases {
		rnd := certutil.RandomString(tc)
		assert.Equal(t, tc, len(rnd))
		assert.NotContains(t, rnd, "=")
		assert.NotContains(t, rnd, "/")
	}
}
