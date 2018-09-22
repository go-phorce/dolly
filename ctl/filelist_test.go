package ctl_test

import (
	"testing"

	"github.com/go-phorce/dolly/ctl"
	"github.com/stretchr/testify/assert"
)

func Test_FilesList(t *testing.T) {
	fl := new(ctl.FilesList)
	assert.True(t, fl.IsCumulative())
	fl.Set("val1")
	fl.Set("val2")
	assert.Equal(t, "val1,val2", fl.String())
}
