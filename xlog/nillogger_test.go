package xlog_test

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/go-phorce/dolly/xlog"
	"github.com/stretchr/testify/assert"
)

func Test_NewNilLogger(t *testing.T) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))

	logger := xlog.NewNilLogger()

	logger.Debug("1")
	logger.Debugf("%d", 2)
	logger.Info("1")
	logger.Infof("%d", 2)
	logger.Error("1")
	logger.Errorf("%d", 2)
	logger.Trace("1")
	logger.Tracef("%d", 2)
	logger.Notice("1")
	logger.Noticef("%d", 2)
	logger.Print("1")
	logger.Println("1")
	logger.Printf("%d", 2)

	assert.Empty(t, b.Bytes())
}
