package gpg_test

import (
	"strconv"
	"testing"

	"github.com/go-phorce/dolly/xpki/gpg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_KeyRingFromFile(t *testing.T) {
	cases := []struct {
		file  string
		count int
	}{
		{
			file:  "testdata/gpg-keys-2",
			count: 2,
		},
		{
			file:  "testdata/RPM-GPG-KEY-CentOS-7",
			count: 1,
		},
	}

	for _, cs := range cases {
		t.Run(cs.file, func(t *testing.T) {
			list, err := gpg.KeyRingFromFile(cs.file)
			require.NoError(t, err)
			assert.Equal(t, cs.count, len(list))
		})
	}

	list, err := gpg.KeyRingFromFile("testdata/gpg-key-bad")
	require.NoError(t, err)
	assert.Equal(t, 0, len(list))
}

func Test_KeyRingFromFiles(t *testing.T) {
	cases := []struct {
		files []string
		count int
	}{
		{
			files: []string{"testdata/gpg-keys-2"},
			count: 2,
		},
		{
			files: []string{"testdata/RPM-GPG-KEY-CentOS-7"},
			count: 1,
		},
		{
			files: []string{"testdata/gpg-keys-2", "testdata/RPM-GPG-KEY-CentOS-7"},
			count: 3,
		},
	}

	for i, cs := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			list, err := gpg.KeyRingFromFiles(cs.files)
			require.NoError(t, err)
			assert.Equal(t, cs.count, len(list))
		})
	}

	list, err := gpg.KeyRingFromFiles([]string{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(list))

	list, err = gpg.KeyRingFromFiles([]string{"missing_file"})
	require.Error(t, err)
	assert.Equal(t, "open missing_file: no such file or directory", err.Error())
}
