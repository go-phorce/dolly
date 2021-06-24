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
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-phorce/dolly/xlog"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly/xlog", "xlog_test")

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

	xlog.SetGlobalLogLevel(xlog.INFO)
	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false))
	logger.Infof("Info log")
	logger.Errorf("Error log")
	logger.Noticef("Notice log")
	logger.Log(xlog.INFO, "log log")
	logger.Logf(xlog.INFO, "log %s", "log")

	result := b.String()
	assert.Contains(t, result, "I | xlog_test: Info log\n")
	assert.Contains(t, result, "E | xlog_test: Error log\n")
	assert.Contains(t, result, "N | xlog_test: Notice log\n")
	assert.Contains(t, result, "I | xlog_test: log log\n")

	b.Reset()
	xlog.GetFormatter().WithCaller(true)
	logger.Infof("Info log")
	logger.Errorf("Error log")
	logger.Noticef("Notice log")
	logger.Log(xlog.INFO, "log log")
	logger.Logf(xlog.INFO, "log %s", "log")

	result = b.String()
	assert.Contains(t, result, "I | xlog_test: src=Test_NewLogger, Info log\n")
	assert.Contains(t, result, "E | xlog_test: src=Test_NewLogger, Error log\n")
	assert.Contains(t, result, "N | xlog_test: src=Test_NewLogger, Notice log\n")
	assert.Contains(t, result, "I | xlog_test: src=Test_NewLogger, log log\n")
}

func Test_PrettyFormatter(t *testing.T) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetGlobalLogLevel(xlog.INFO)
	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, false).WithCaller(true))

	logger.Info("Test Info")
	result := b.String()
	expected := "I | xlog_test: src=Test_PrettyFormatter, Test Info\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Infof("Test Infof")
	result = b.String()
	expected = "I | xlog_test: src=Test_PrettyFormatter, Test Infof\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	k3 := struct {
		Foo string
	}{Foo: "bar"}

	logger.KV(xlog.INFO, "k1", 1, "k2", false, "k3", k3)
	result = b.String()
	expected = "I | xlog_test: src=Test_PrettyFormatter, k1=1, k2=false, k3={\"Foo\":\"bar\"}\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Errorf("Test Error")
	result = b.String()
	expected = "E | xlog_test: src=Test_PrettyFormatter, Test Error\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Warningf("Test Warning")
	result = b.String()
	expected = "W | xlog_test: src=Test_PrettyFormatter, Test Warning\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	// Debug level is disabled
	logger.Debugf("Test Debug")
	result = b.String()
	expected = "D | xlog_test: src=Test_PrettyFormatter, Test Debug\n"
	assert.NotContains(t, result, expected, "Log format does not match")
	b.Reset()

	xlog.SetGlobalLogLevel(xlog.DEBUG)
	logger.Debugf("Test Debug")
	result = b.String()
	expected = "D | xlog_test: src=Test_PrettyFormatter, Test Debug\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()
}

func Test_WithTracedError(t *testing.T) {
	wd, err := os.Getwd() // package dir
	require.NoError(t, err)

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
			"E | xlog_test: stack=[github.com/go-phorce/dolly/xlog/xlog_test.go:38: originateError: msg=Test_WithTracedError(1), level=0\ngithub.com/go-phorce/dolly/xlog/xlog_test.go:45: \ngithub.com/go-phorce/dolly/xlog/xlog_test.go:43: ]\n",
		},
		{
			"Test_WithTracedError(4)",
			2,
			"E | xlog_test: err=[originateError: msg=Test_WithTracedError(4), level=0]\n",
			"E | xlog_test: stack=[github.com/go-phorce/dolly/xlog/xlog_test.go:38: originateError: msg=Test_WithTracedError(4), level=0\ngithub.com/go-phorce/dolly/xlog/xlog_test.go:45: \ngithub.com/go-phorce/dolly/xlog/xlog_test.go:43: \ngithub.com/go-phorce/dolly/xlog/xlog_test.go:43: ]\n",
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
		result := b.String()[prefixLen:]
		assert.Equal(t, c.expectedErr, result, "[%d] case failed expectation", idx)
		b.Reset()

		logger.Errorf("err=[%v]", err.Error())
		result = b.String()[prefixLen:]
		assert.Equal(t, c.expectedErr, result, "[%d] case failed expectation", idx)
		b.Reset()

		logger.Errorf("stack=[%v]", errors.ErrorStack(err))
		result = b.String()[prefixLen:]
		// remove paths from the trace
		result = strings.Replace(result, wd, "github.com/go-phorce/dolly/xlog", -1)
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
			"E | xlog_test: stack=[github.com/go-phorce/dolly/xlog/xlog_test.go:38: originateError: msg=Test_WithAnnotatedError(1), level=0\ngithub.com/go-phorce/dolly/xlog/xlog_test.go:52: annotateError, level=0\ngithub.com/go-phorce/dolly/xlog/xlog_test.go:50: ]\n",
		},
		{
			"Test_WithAnnotatedError(4)",
			2,
			"E | xlog_test: err=[annotateError, level=0: originateError: msg=Test_WithAnnotatedError(4), level=0]\n",
			"E | xlog_test: stack=[github.com/go-phorce/dolly/xlog/xlog_test.go:38: originateError: msg=Test_WithAnnotatedError(4), level=0\ngithub.com/go-phorce/dolly/xlog/xlog_test.go:52: annotateError, level=0\ngithub.com/go-phorce/dolly/xlog/xlog_test.go:50: \ngithub.com/go-phorce/dolly/xlog/xlog_test.go:50: ]\n",
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
		result := b.String()[prefixLen:]
		assert.Equal(t, c.expectedErr, result, "[%d] case failed expectation", idx)
		b.Reset()

		logger.Errorf("err=[%v]", err.Error())
		result = b.String()[prefixLen:]
		assert.Equal(t, c.expectedErr, result, "[%d] case failed expectation", idx)
		b.Reset()

		logger.Errorf("stack=[%v]", errors.ErrorStack(err))
		result = b.String()[prefixLen:]
		// remove paths from the trace
		result = strings.Replace(result, wd, "github.com/go-phorce/dolly/xlog", -1)
		assert.Equal(t, c.expectedStack, result, "[%d] case failed expectation", idx)
		b.Reset()
	}
}

func Test_LevelAt(t *testing.T) {
	l, err := xlog.GetRepoLogger("github.com/go-phorce/dolly/xlog")
	require.NoError(t, err)

	l.SetRepoLogLevel(xlog.INFO)
	assert.True(t, logger.LevelAt(xlog.INFO))
	assert.False(t, logger.LevelAt(xlog.TRACE))
	assert.False(t, logger.LevelAt(xlog.DEBUG))

	l.SetRepoLogLevel(xlog.TRACE)
	assert.True(t, logger.LevelAt(xlog.INFO))
	assert.True(t, logger.LevelAt(xlog.TRACE))
	assert.False(t, logger.LevelAt(xlog.DEBUG))

	l.SetRepoLogLevel(xlog.DEBUG)
	assert.True(t, logger.LevelAt(xlog.INFO))
	assert.True(t, logger.LevelAt(xlog.TRACE))
	assert.True(t, logger.LevelAt(xlog.DEBUG))
}

func Test_PrettyFormatterDebug(t *testing.T) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetFormatter(xlog.NewPrettyFormatter(writer, true).WithCaller(true))
	xlog.SetGlobalLogLevel(xlog.INFO)

	logger.Trace("Test trace")
	logger.Tracef("Test tracef")
	result := b.String()
	expected := "T | xlog_test: src=Test_PrettyFormatterDebug, Test trace\n"
	assert.NotContains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Info("Test Info")
	logger.Infof("Test Infof")
	result = b.String()
	expected = "I | xlog_test: src=Test_PrettyFormatterDebug, Test Info\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.KV(xlog.INFO, "k1", 1, "k2", false)
	writer.Flush()
	result = b.String()
	expected = "I | xlog_test: src=Test_PrettyFormatterDebug, k1=1, k2=false\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Error("Test Error")
	logger.Errorf("Test Errorf")
	result = b.String()
	expected = "E | xlog_test: src=Test_PrettyFormatterDebug, Test Error\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Notice("Test Notice")
	logger.Noticef("Test Noticef")
	result = b.String()
	expected = "N | xlog_test: src=Test_PrettyFormatterDebug, Test Noticef\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Warning("Test Warning")
	logger.Warningf("Test Warning")
	result = b.String()
	expected = "W | xlog_test: src=Test_PrettyFormatterDebug, Test Warning\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	// Debug level is disabled
	logger.Debug("Test Debug")
	logger.Debugf("Test Debug")
	result = b.String()
	expected = "xlog_test: src=Test_PrettyFormatterDebug, Test Debug"
	assert.NotContains(t, result, expected, "Log format does not match")
	b.Reset()

	xlog.SetGlobalLogLevel(xlog.DEBUG)
	logger.Debug("Test Debug")
	logger.Debugf("Test Debug")
	result = b.String()
	expected = "[xlog_test.go:335] D | xlog_test: src=Test_PrettyFormatterDebug, Test Debug\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	xlog.SetGlobalLogLevel(xlog.TRACE)

	logger.Trace("Test trace")
	logger.Tracef("Test trace")
	result = b.String()
	expected = "T | xlog_test: src=Test_PrettyFormatterDebug, Test trace\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Flush()
}

func Test_StringFormatter(t *testing.T) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetFormatter(xlog.NewStringFormatter(writer).WithCaller(true))
	xlog.SetGlobalLogLevel(xlog.INFO)

	logger.Infof("Test Info")
	result := b.String()
	expected := " xlog_test: src=Test_StringFormatter, Test Info\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Errorf("Test Error")
	result = b.String()
	expected = " xlog_test: src=Test_StringFormatter, Test Error\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Warningf("Test Warning")
	result = b.String()
	expected = " xlog_test: src=Test_StringFormatter, Test Warning\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	// Debug level is disabled
	logger.Debugf("Test Debug")
	result = b.String()
	expected = "[packagelogger.go:166] xlog_test: src=Test_StringFormatter, Test Debug\n"
	assert.NotContains(t, result, expected, "Log format does not match")
	b.Reset()

	xlog.SetGlobalLogLevel(xlog.DEBUG)

	log2 := logger.WithValues("count", 1)
	log2.Debugf("Test Debug")
	result = b.String()
	expected = "xlog_test: src=Test_StringFormatter, count=1, Test Debug\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	log2.KV(xlog.INFO, "k1", 1, "k2", false)
	result = b.String()
	expected = "xlog_test: src=Test_StringFormatter, count=1, k1=1, k2=false\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()
}

func Test_LogFormatter(t *testing.T) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetGlobalLogLevel(xlog.INFO)
	f := xlog.NewLogFormatter(writer, "test ", 0)
	xlog.SetFormatter(f)

	logger.Infof("Test Info")
	writer.Flush()
	result := b.String()
	expected := " xlog_test: Test Info\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.KV(xlog.INFO, "k1", 1, "k2", false)
	writer.Flush()
	result = b.String()
	expected = "test xlog_test: k1=1, k2=false\n"
	assert.Equal(t, expected, result, "Log format does not match")
	b.Reset()

	logger.Errorf("Test Error")
	writer.Flush()
	result = b.String()
	expected = " xlog_test: Test Error\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	logger.Warningf("Test Warning")
	writer.Flush()
	result = b.String()
	expected = " xlog_test: Test Warning\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	// Debug level is disabled
	logger.Debugf("Test Debug")
	writer.Flush()
	result = b.String()
	expected = "[packagelogger.go:166] xlog_test: Test Debug\n"
	assert.NotContains(t, result, expected, "Log format does not match")
	b.Reset()

	xlog.SetGlobalLogLevel(xlog.DEBUG)
	logger.Debugf("Test Debug")
	writer.Flush()
	result = b.String()
	expected = "xlog_test: Test Debug\n"
	assert.Contains(t, result, expected, "Log format does not match")
	b.Reset()

	f.Flush()
}

func Test_ColorFormatterDebug(t *testing.T) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	xlog.SetFormatter(xlog.NewColorFormatter(writer, true))
	xlog.SetGlobalLogLevel(xlog.DEBUG)

	logger.Infof("Test Info")
	result := b.String()
	expected := string(xlog.LevelColors[xlog.INFO]) + " I | xlog_test: Test Info\n\033[0m"
	assert.Contains(t, result, expected)
	b.Reset()

	logger.KV(xlog.INFO, "k1", 1, "err", fmt.Errorf("not found"))
	writer.Flush()
	result = b.String()
	expected = "I | xlog_test: k1=1, err=\"not found\"\n"
	assert.Contains(t, result, expected)
	b.Reset()

	logger.Errorf("Test Error")
	result = b.String()
	expected = string(xlog.LevelColors[xlog.ERROR]) + " E | xlog_test: Test Error\n\033[0m"
	assert.Contains(t, result, expected)
	b.Reset()

	logger.Error("unable to find: ", fmt.Errorf("not found"))
	result = b.String()
	expected = string(xlog.LevelColors[xlog.ERROR]) + " E | xlog_test: unable to find: not found\n\x1b[0m"
	assert.Contains(t, result, expected)
	b.Reset()

	logger.Warningf("Test Warning")
	result = b.String()
	expected = string(xlog.LevelColors[xlog.WARNING]) + " W | xlog_test: Test Warning\n\033[0m"
	assert.Contains(t, result, expected)
	b.Reset()

	logger.Tracef("Test Trace")
	result = b.String()
	expected = string(xlog.LevelColors[xlog.TRACE]) + " T | xlog_test: Test Trace\n\033[0m"
	assert.Contains(t, result, expected)
	b.Reset()

	logger.Debugf("Test Debug")
	result = b.String()
	expected = string(xlog.LevelColors[xlog.DEBUG]) + " D | xlog_test: Test Debug\n\033[0m"
	assert.Contains(t, result, expected)
	b.Reset()
}

func Test_NilFormatter(t *testing.T) {
	f := xlog.NewNilFormatter()
	f.FormatKV("pkg", xlog.DEBUG, 1)
	f.Format("pkg", xlog.DEBUG, 1)
	f.Flush()
}
