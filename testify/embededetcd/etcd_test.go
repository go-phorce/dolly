package embededetcd_test

import (
	"os"
	"testing"

	"github.com/go-phorce/dolly/testify/embededetcd"
	"github.com/stretchr/testify/require"
)

func Test_StartEtcd(t *testing.T) {
	path, s, err := embededetcd.Start("test-integration-etcd", true)
	require.NoError(t, err)

	s.Close()
	os.RemoveAll(path)
}
