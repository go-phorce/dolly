package xlog_test

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-phorce/dolly/xlog"
)

func Test_Hijack(t *testing.T) {
	w := bytes.NewBuffer([]byte{})
	f := xlog.NewStringFormatter(w)
	xlog.SetFormatter(f)
	xlog.SetGlobalLogLevel(xlog.INFO)

	log.SetFlags(0)
	log.SetPrefix("prefix:")
	log.Println("testing")

	assert.Contains(t, string(w.Bytes()), " prefix:testing\n")
}

func Test_Hijack_Error(t *testing.T) {
	w := bytes.NewBuffer([]byte{})
	f := xlog.NewStringFormatter(w)
	xlog.SetFormatter(f)
	xlog.SetGlobalLogLevel(xlog.ERROR)

	log.Println("testing")

	assert.Empty(t, w.Bytes())
}
