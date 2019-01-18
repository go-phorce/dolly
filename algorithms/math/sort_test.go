package math

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_Sort_Nil(t *testing.T) {
	keysSorted := SortedKeys(nil)
	require.Equal(t, 0, len(keysSorted))
}

func Test_Sort(t *testing.T) {
	d := map[string]string{
		"key3": "value3",
		"key1": "value1",
		"key2": "value2",
		"key4": "value1",
	}
	keysSorted := SortedKeys(d)
	require.Equal(t, 4, len(keysSorted))
	require.Equal(t, "key1", keysSorted[0])
	require.Equal(t, "key2", keysSorted[1])
	require.Equal(t, "key3", keysSorted[2])
	require.Equal(t, "key4", keysSorted[3])
}
