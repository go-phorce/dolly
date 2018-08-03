// Copyright 2018, Denis Issoupov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package xlog_test

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ekspand/pkg/xlog"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var logger = xlog.NewPackageLogger("github.com/ekspand/pkg/pkg", "xlog_test")

const logPrefixFormt = "2018-04-17 20:53:46.589926 "

// NOTE: keep the xxxError() functions at the beginnign of the file,
// as tests produce the error stack

func originateError(errmsg string, level int) error {
	return errors.Errorf("originateError: msg=%s, level=%d", errmsg, level)
}

func traceError(errmsg string, levels int) error {
	if levels > 0 {
		return errors.Trace(traceError(errmsg, levels-1))
	}
	return errors.Trace(originateError(errmsg, 0))
}

func annotateError(errmsg string, levels int) error {
	if levels > 0 {
		return errors.Trace(annotateError(errmsg, levels-1))
	}
	return errors.Annotatef(originateError(errmsg, 0), "annotateError, level=%d", levels)
}

func withTracedError(errmsg string, levels int) error {
	return traceError(errmsg, levels)
}

func withAnnotateError(errmsg string, levels int) error {
	return annotateError(errmsg, levels)
}

func Test_NewLogger(t *testing.T) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))
	logger.Infof("Test Log")

	result := string(b.Bytes())
	expected := "I | xlog_test: Test Log\n"
	assert.Contains(t, result, expected, "Log format does not match")
}

func Test_Levels(t *testing.T) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))

	logger.Infof("Test Info\n")
	result := string(b.Bytes())
	expected := "I | xlog_test: Test Info\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Errorf("Test Error\n")
	result = string(b.Bytes())
	expected = "E | xlog_test: Test Error\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Warningf("Test Warning\n")
	result = string(b.Bytes())
	expected = "W | xlog_test: Test Warning\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	// Debug level is disabled
	logger.Debugf("Test Debug\n")
	result = string(b.Bytes())
	expected = "D | xlog_test: Test Debug\n"
	assert.NotContains(t, result, expected, "Log format does not match")
	b.Reset()

	xlog.SetGlobalLogLevel(xlog.DEBUG)
	logger.Debugf("Test Debug\n")
	result = string(b.Bytes())
	expected = "D | xlog_test: Test Debug\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()
}

func Test_WithTracedError(t *testing.T) {
	wd, _ := os.Getwd() // package dir

	cases := []struct {
		msg           string
		levels        int
		expectedErr   string
		expectedStack string
	}{
		{
			"Test_WithTracedError(1)",
			1,
			"E | xlog_test: err=[originateError: msg=Test_WithTracedError(1), level=0]\n",
			"E | xlog_test: stack=[/xlog_test.go:37: originateError: msg=Test_WithTracedError(1), level=0\n/xlog_test.go:44: \n/xlog_test.go:42: ]\n",
		},
		{
			"Test_WithTracedError(4)",
			2,
			"E | xlog_test: err=[originateError: msg=Test_WithTracedError(4), level=0]\n",
			"E | xlog_test: stack=[/xlog_test.go:37: originateError: msg=Test_WithTracedError(4), level=0\n/xlog_test.go:44: \n/xlog_test.go:42: \n/xlog_test.go:42: ]\n",
		},
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))

	prefixLen := len(logPrefixFormt)
	for idx, c := range cases {
		err := withTracedError(c.msg, c.levels)
		require.Error(t, err)

		logger.Errorf("err=[%v]", err)
		result := string(b.Bytes())[prefixLen:]
		assert.Equal(t, c.expectedErr, result, "[%d] case failed expectation", idx)
		b.Reset()

		logger.Errorf("err=[%v]", err.Error())
		result = string(b.Bytes())[prefixLen:]
		assert.Equal(t, c.expectedErr, result, "[%d] case failed expectation", idx)
		b.Reset()

		logger.Errorf("stack=[%v]", errors.ErrorStack(err))
		result = string(b.Bytes())[prefixLen:]
		// remove paths from the trace
		result = strings.Replace(result, wd, "", -1)
		assert.Equal(t, c.expectedStack, result, "[%d] case failed expectation", idx)
		b.Reset()
	}
}

func Test_WithAnnotatedError(t *testing.T) {
	wd, _ := os.Getwd() // package dir

	cases := []struct {
		msg           string
		levels        int
		expectedErr   string
		expectedStack string
	}{
		{
			"Test_WithAnnotatedError(1)",
			1,
			"E | xlog_test: err=[annotateError, level=0: originateError: msg=Test_WithAnnotatedError(1), level=0]\n",
			"E | xlog_test: stack=[/xlog_test.go:37: originateError: msg=Test_WithAnnotatedError(1), level=0\n/xlog_test.go:51: annotateError, level=0\n/xlog_test.go:49: ]\n",
		},
		{
			"Test_WithAnnotatedError(4)",
			2,
			"E | xlog_test: err=[annotateError, level=0: originateError: msg=Test_WithAnnotatedError(4), level=0]\n",
			"E | xlog_test: stack=[/xlog_test.go:37: originateError: msg=Test_WithAnnotatedError(4), level=0\n/xlog_test.go:51: annotateError, level=0\n/xlog_test.go:49: \n/xlog_test.go:49: ]\n",
		},
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))

	prefixLen := len(logPrefixFormt)
	for idx, c := range cases {
		err := withAnnotateError(c.msg, c.levels)
		require.Error(t, err)

		logger.Errorf("err=[%v]", err)
		result := string(b.Bytes())[prefixLen:]
		assert.Equal(t, c.expectedErr, result, "[%d] case failed expectation", idx)
		b.Reset()

		logger.Errorf("err=[%v]", err.Error())
		result = string(b.Bytes())[prefixLen:]
		assert.Equal(t, c.expectedErr, result, "[%d] case failed expectation", idx)
		b.Reset()

		logger.Errorf("stack=[%v]", errors.ErrorStack(err))
		result = string(b.Bytes())[prefixLen:]
		// remove paths from the trace
		result = strings.Replace(result, wd, "", -1)
		assert.Equal(t, c.expectedStack, result, "[%d] case failed expectation", idx)
		b.Reset()
	}
}
