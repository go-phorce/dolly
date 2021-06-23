package logrotate_test

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-phorce/dolly/xlog"
	"github.com/go-phorce/dolly/xlog/logrotate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Rotate(t *testing.T) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	tmpDir := filepath.Join(os.TempDir(), "tests", "logrotate")

	logRotate, err := logrotate.Initialize(tmpDir, "rotator", 1, 1, false, writer)
	require.NoError(t, err)
	defer logRotate.Close()

	logger := xlog.NewPackageLogger("github.com/go-phorce/dolly/xlog", "logrotate")
	xlog.SetGlobalLogLevel(xlog.TRACE)

	logger.Debug("1")
	logger.Debugf("%d", 2)
	logger.Info("1")
	logger.Infof("%d", 2)
	logger.KV(xlog.INFO, "k", 2)
	logger.Error("1")
	logger.Errorf("%d", 2)
	logger.Trace("1")
	logger.Tracef("%d", 2)
	logger.Notice("1")
	logger.Noticef("%d", 2)

	writer.Flush()
	assert.NotEmpty(t, b.Bytes())
}
