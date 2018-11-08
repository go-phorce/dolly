package armor_test

import (
	"io/ioutil"
	"testing"

	"github.com/go-phorce/dolly/xpki/armor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ArmorDecode(t *testing.T) {
	cases := []struct {
		file  string
		count int
	}{
		{
			file:  "testdata/RPM-GPG-KEY-CentOS-7",
			count: 1,
		},
		{
			file:  "testdata/test-gpg-keys-2",
			count: 2,
		},
	}

	for _, cs := range cases {
		t.Run(cs.file, func(t *testing.T) {
			data, err := ioutil.ReadFile(cs.file)
			require.NoError(t, err)

			count := 0
			for {
				block, rest := armor.Decode(data)
				require.NotNil(t, block)
				count++

				if rest == nil || len(rest) == 0 {
					break
				}
				data = rest
			}

			assert.Equal(t, cs.count, count)
		})
	}
}
