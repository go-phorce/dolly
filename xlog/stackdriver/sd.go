package stackdriver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/go-phorce/dolly/xlog"
)

type severity string

const (
	severityDebug    severity = "DEBUG"
	severityInfo     severity = "INFO"
	severityNotice   severity = "NOTICE"
	severityWarning  severity = "WARNING"
	severityError    severity = "ERROR"
	severityCritical severity = "CRITICAL"
	severityAlert    severity = "ALERT"
)

var levelsToSeverity = map[xlog.LogLevel]severity{
	xlog.DEBUG:    severityDebug,
	xlog.TRACE:    severityDebug,
	xlog.INFO:     severityInfo,
	xlog.NOTICE:   severityNotice,
	xlog.WARNING:  severityWarning,
	xlog.ERROR:    severityError,
	xlog.CRITICAL: severityCritical,
}

// formatter provides logs format for StackDriver
type formatter struct {
	w          *bufio.Writer
	logName    string
	withCaller bool
}

// NewFormatter returns an instance of StackdriverFormatter
func NewFormatter(w io.Writer, logName string) xlog.Formatter {
	return &formatter{
		w:          bufio.NewWriter(w),
		logName:    logName,
		withCaller: true,
	}
}

// WithCaller allows to configure if the caller shall be logged
func (c *formatter) WithCaller(val bool) xlog.Formatter {
	c.withCaller = val
	return c
}

// FormatKV log entry string to the stream,
// the entries are key/value pairs
func (c *formatter) FormatKV(pkg string, level xlog.LogLevel, depth int, entries ...interface{}) {
	c.Format(pkg, level, depth+1, flatten(entries...))
}

// Format log entry string to the stream
func (c *formatter) Format(pkg string, l xlog.LogLevel, depth int, entries ...interface{}) {
	severity := levelsToSeverity[l]
	if severity == "" {
		severity = severityInfo
	}

	str := fmt.Sprint(entries...)
	ee := entry{
		LogName:   c.logName,
		Component: pkg,
		Time:      time.Now().UTC().Format(time.RFC3339),
		Message:   str,
		Severity:  severity,
	}

	fn, file, line := callerName(depth + 1)
	ee.Source = &reportLocation{
		Function: fn,
	}

	if l <= xlog.ERROR {
		ee.Source.FilePath = path.Base(file)
		ee.Source.LineNumber = line
	}

	b, err := json.Marshal(ee)
	if err == nil {
		c.w.Write(b)
		c.w.WriteByte('\n')
	}

	c.Flush()
}

// Flush the logs
func (c *formatter) Flush() {
	c.w.Flush()
}

type entry struct {
	LogName   string          `json:"logName,omitempty"`
	Component string          `json:"component,omitempty"`
	Time      string          `json:"timestamp,omitempty"`
	Message   string          `json:"message,omitempty"`
	Severity  severity        `json:"severity,omitempty"`
	Source    *reportLocation `json:"sourceLocation,omitempty"`
}

type reportLocation struct {
	FilePath   string `json:"file,omitempty"`
	LineNumber int    `json:"line,omitempty"`
	Function   string `json:"function,omitempty"`
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

func callerName(depth int) (string, string, int) {
	pc, file, line, ok := runtime.Caller(depth)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		name := details.Name()

		idx := strings.LastIndex(name, ".")
		if idx >= 0 {
			name = name[idx+1:]
		}
		return name, file, line
	}
	return "n/a", file, line
}
