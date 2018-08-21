package netutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParseURLs(t *testing.T) {
	tcases := []struct {
		tname string
		hosts []string
		err   string
	}{
		{"from nil", nil, ""},
		{"from_empty", nil, ""},
		{"valid", []string{"localhost", "123.74.56.18", "ekspand.com"}, ""},
		{"valid with path", []string{"../dir/"}, ""},
		{"valid1 with page", []string{"foo.html"}, ""},
		{"invalid with ip", []string{"http://192.168.0.%31/"}, "error"},
		{"invalid with code", []string{"http://[fe80::%231]:8080/"}, "error"},
	}

	for _, tc := range tcases {
		t.Run(tc.tname, func(t *testing.T) {
			l, err := ParseURLs(tc.hosts)
			if tc.err == "" {
				require.NoError(t, err)
				assert.Equal(t, len(tc.hosts), len(l))
			} else {
				if !assert.Error(t, err) {
					for _, u := range l {
						t.Logf("parsed url: %s", u.String())
					}
				}
			}
		})
	}
}
