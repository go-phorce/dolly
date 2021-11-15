// Copyright 2015 CoreOS, Inc.
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

package xlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"runtime"
	"strings"
	"time"
)

// Formatter defines an interface for formatting logs
type Formatter interface {
	// Format log entry string to the stream,
	// the entries are separated by space
	Format(pkg string, level LogLevel, depth int, entries ...interface{})
	// FormatKV log entry string to the stream,
	// the entries are key/value pairs
	FormatKV(pkg string, level LogLevel, depth int, entries ...interface{})
	// Flush the logs
	Flush()
	// WithCaller allows to configure if the caller shall be logged
	WithCaller(bool) Formatter
}

// NewStringFormatter returns string-based formatter
func NewStringFormatter(w io.Writer) Formatter {
	return &StringFormatter{
		w:          bufio.NewWriter(w),
		withCaller: true,
	}
}

// StringFormatter defines string-based formatter
type StringFormatter struct {
	w          *bufio.Writer
	withCaller bool
}

// FormatKV log entry string to the stream,
// the entries are key/value pairs
func (s *StringFormatter) FormatKV(pkg string, level LogLevel, depth int, entries ...interface{}) {
	s.Format(pkg, level, depth+1, flatten(entries...))
}

// WithCaller allows to configure if the caller shall be logged
func (s *StringFormatter) WithCaller(val bool) Formatter {
	s.withCaller = val
	return s
}

// Format log entry string to the stream
func (s *StringFormatter) Format(pkg string, l LogLevel, depth int, entries ...interface{}) {
	now := time.Now().UTC()
	s.w.WriteString(now.Format(time.RFC3339))
	s.w.WriteByte(' ')
	writeEntries(s.w, pkg, l, depth+1, s.withCaller, entries...)
	s.Flush()
}

func writeEntries(w *bufio.Writer, pkg string, _ LogLevel, depth int, withCaller bool, entries ...interface{}) {
	if pkg != "" {
		w.WriteString(pkg + ": ")
	}

	if withCaller {
		w.WriteString("src=")
		w.WriteString(callerName(depth + 1))
		w.WriteString(", ")
	}

	str := fmt.Sprint(entries...)
	endsInNL := strings.HasSuffix(str, "\n")
	w.WriteString(str)
	if !endsInNL {
		w.WriteString("\n")
	}
}

// Flush the logs
func (s *StringFormatter) Flush() {
	s.w.Flush()
}

// NewPrettyFormatter returns an instance of PrettyFormatter
func NewPrettyFormatter(w io.Writer, debug bool) Formatter {
	return &PrettyFormatter{
		w:          bufio.NewWriter(w),
		debug:      debug,
		withCaller: true,
	}
}

// PrettyFormatter provides default logs format
type PrettyFormatter struct {
	w          *bufio.Writer
	debug      bool
	withCaller bool
}

// WithCaller allows to configure if the caller shall be logged
func (c *PrettyFormatter) WithCaller(val bool) Formatter {
	c.withCaller = val
	return c
}

// FormatKV log entry string to the stream,
// the entries are key/value pairs
func (c *PrettyFormatter) FormatKV(pkg string, level LogLevel, depth int, entries ...interface{}) {
	c.Format(pkg, level, depth+1, flatten(entries...))
}

// Format log entry string to the stream
func (c *PrettyFormatter) Format(pkg string, l LogLevel, depth int, entries ...interface{}) {
	now := time.Now()
	ts := now.Format("2006-01-02 15:04:05")
	c.w.WriteString(ts)
	ms := now.Nanosecond() / 1000
	c.w.WriteString(fmt.Sprintf(".%06d", ms))
	if c.debug {
		_, file, line, ok := runtime.Caller(depth) // It's always the same number of frames to the user's call.
		if !ok {
			file = "???"
			line = 1
		} else {
			slash := strings.LastIndex(file, "/")
			if slash >= 0 {
				file = file[slash+1:]
			}
		}
		if line < 0 {
			line = 0 // not a real line number
		}
		c.w.WriteString(fmt.Sprintf(" [%s:%d]", file, line))
	}
	c.w.WriteString(fmt.Sprint(" ", l.Char(), " | "))
	writeEntries(c.w, pkg, l, depth+1, c.withCaller, entries...)
	c.Flush()
}

// Flush the logs
func (c *PrettyFormatter) Flush() {
	c.w.Flush()
}

// NewColorFormatter returns an instance of ColorFormatter
func NewColorFormatter(w io.Writer, color bool) Formatter {
	return &ColorFormatter{
		w:          bufio.NewWriter(w),
		color:      color,
		withCaller: true,
	}
}

// ColorFormatter provides colorful logs format
type ColorFormatter struct {
	w          *bufio.Writer
	color      bool
	withCaller bool
}

// color pallete map
var (
	ColorOff         = []byte("\033[0m")
	colorRed         = []byte("\033[0;31m")
	colorGreen       = []byte("\033[0;32m")
	colorOrange      = []byte("\033[0;33m")
	colorBlue        = []byte("\033[0;34m")
	colorPurple      = []byte("\033[0;35m")
	colorCyan        = []byte("\033[0;36m")
	colorGray        = []byte("\033[0;37m") // TRACE
	colorLightRed    = []byte("\033[0;91m") // ERROR
	colorLightGreen  = []byte("\033[0;92m") // NOTICE
	colorLightOrange = []byte("\033[0;93m") // WARN
	colorLightBlue   = []byte("\033[0;94m") // DEBUG
	colorLightCyan   = []byte("\033[0;96m") // INFO
)

// LevelColors provides colors map
var LevelColors = map[LogLevel][]byte{
	CRITICAL: colorLightRed,
	ERROR:    colorLightRed,
	WARNING:  colorLightOrange,
	NOTICE:   colorLightGreen,
	INFO:     colorLightCyan,
	DEBUG:    colorGray,
	TRACE:    colorGray,
}

// WithCaller allows to configure if the caller shall be logged
func (c *ColorFormatter) WithCaller(val bool) Formatter {
	c.withCaller = val
	return c
}

// FormatKV log entry string to the stream,
// the entries are key/value pairs
func (c *ColorFormatter) FormatKV(pkg string, level LogLevel, depth int, entries ...interface{}) {
	c.Format(pkg, level, depth+1, flatten(entries...))
}

// Format log entry string to the stream
func (c *ColorFormatter) Format(pkg string, l LogLevel, depth int, entries ...interface{}) {
	now := time.Now()
	ts := now.Format("2006-01-02 15:04:05")
	c.w.WriteString(ts)
	ms := now.Nanosecond() / 1000
	c.w.WriteString(fmt.Sprintf(".%06d", ms))
	if c.color {
		c.w.Write(LevelColors[l])
	}
	c.w.WriteString(fmt.Sprint(" ", l.Char(), " | "))
	writeEntries(c.w, pkg, l, depth+1, c.withCaller, entries...)
	if c.color {
		c.w.Write(ColorOff)
	}
	c.Flush()
}

// Flush the logs
func (c *ColorFormatter) Flush() {
	c.w.Flush()
}

// LogFormatter emulates the form of the traditional built-in logger.
type LogFormatter struct {
	logger     *log.Logger
	prefix     string
	withCaller bool
}

// NewLogFormatter is a helper to produce a new LogFormatter struct. It uses the
// golang log package to actually do the logging work so that logs look similar.
func NewLogFormatter(w io.Writer, prefix string, flag int) Formatter {
	return &LogFormatter{
		logger: log.New(w, "", flag), // don't use prefix here
		prefix: prefix,               // save it instead
	}
}

// WithCaller allows to configure if the caller shall be logged
func (lf *LogFormatter) WithCaller(val bool) Formatter {
	lf.withCaller = val
	return lf
}

// FormatKV log entry string to the stream,
// the entries are key/value pairs
func (lf *LogFormatter) FormatKV(pkg string, level LogLevel, depth int, entries ...interface{}) {
	lf.Format(pkg, level, depth+1, flatten(entries...))
}

// Format builds a log message for the LogFormatter. The LogLevel is ignored.
func (lf *LogFormatter) Format(pkg string, _ LogLevel, _ int, entries ...interface{}) {
	str := fmt.Sprint(entries...)
	prefix := lf.prefix
	if pkg != "" {
		prefix = fmt.Sprintf("%s%s: ", prefix, pkg)
	}
	lf.logger.Output(5, fmt.Sprintf("%s%v", prefix, str)) // call depth is 5
}

// Flush is included so that the interface is complete, but is a no-op.
func (lf *LogFormatter) Flush() {
	// noop
}

// NilFormatter is a no-op log formatter that does nothing.
type NilFormatter struct {
}

// NewNilFormatter is a helper to produce a new LogFormatter struct. It logs no
// messages so that you can cause part of your logging to be silent.
func NewNilFormatter() Formatter {
	return &NilFormatter{}
}

// WithCaller allows to configure if the caller shall be logged
func (c *NilFormatter) WithCaller(val bool) Formatter {
	return c
}

// FormatKV log entry string to the stream,
// the entries are key/value pairs
func (*NilFormatter) FormatKV(pkg string, level LogLevel, depth int, entries ...interface{}) {
}

// Format does nothing.
func (*NilFormatter) Format(_ string, _ LogLevel, _ int, _ ...interface{}) {
	// noop
}

// Flush is included so that the interface is complete, but is a no-op.
func (*NilFormatter) Flush() {
	// noop
}

func flatten(kvList ...interface{}) string {
	size := len(kvList)
	buf := bytes.Buffer{}
	for i := 0; i < size; i += 2 {
		k, ok := kvList[i].(string)
		if !ok {
			panic(fmt.Sprintf("key is not a string: %s", String(kvList[i])))
		}
		var v interface{}
		if i+1 < size {
			v = kvList[i+1]
		}
		if i > 0 {
			buf.WriteRune(',')
			buf.WriteRune(' ')
		}
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(String(v))

	}
	return buf.String()
}

// String returns string value stuitable for logging
func String(value interface{}) string {
	if err, ok := value.(error); ok {
		// if error does not support json.Marshaler,
		// the print the full details
		if _, ok := value.(json.Marshaler); !ok {
			value = fmt.Sprintf("%+v", err)
		}
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.Encode(value)
	return strings.TrimSpace(buffer.String())
}

func callerName(depth int) string {
	pc, _, _, ok := runtime.Caller(depth)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		name := details.Name()

		idx := strings.LastIndex(name, ".")
		if idx >= 0 {
			name = name[idx+1:]
		}
		return name
	}
	return "n/a"
}
