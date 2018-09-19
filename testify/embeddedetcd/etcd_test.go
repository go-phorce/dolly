package embeddedetcd_test

import (
	"os"
	"testing"

	"github.com/go-phorce/dolly/testify/embeddedetcd"
	"github.com/stretchr/testify/require"
)

func Test_StartEtcd(t *testing.T) {
	path, s, err := embeddedetcd.Start("test-integration-etcd", true)
	require.NoError(t, err)

	s.Close()
	os.RemoveAll(path)
}
