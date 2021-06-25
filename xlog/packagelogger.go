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
	"fmt"
	"os"
)

// PackageLogger is logger implementation for packages
type PackageLogger struct {
	pkg    string
	level  LogLevel
	values []interface{}
}

const calldepth = 2

type entriesType int

const (
	plain entriesType = iota
	kv
)

// WithValues adds some key-value pairs of context to a logger.
// See Info for documentation on how key/value pairs work.
func (p *PackageLogger) WithValues(keysAndValues ...interface{}) Logger {
	return &PackageLogger{
		pkg:    p.pkg,
		level:  p.level,
		values: append(p.values, keysAndValues...),
	}
}

func (p *PackageLogger) internalLog(t entriesType, depth int, inLevel LogLevel, entries ...interface{}) {
	logger.Lock()
	defer logger.Unlock()
	if inLevel != CRITICAL && p.level < inLevel {
		return
	}
	if len(p.values) > 0 {
		entries = append(p.values, entries...)
	}
	if logger.formatter != nil {
		if t == plain {
			logger.formatter.Format(p.pkg, inLevel, depth+1, entries...)
		} else {
			logger.formatter.FormatKV(p.pkg, inLevel, depth+1, entries...)
		}
	}
}

func (p *PackageLogger) internalLogf(depth int, inLevel LogLevel, format string, args ...interface{}) {
	logger.Lock()
	defer logger.Unlock()
	if inLevel != CRITICAL && p.level < inLevel {
		return
	}
	if logger.formatter != nil {
		s := fmt.Sprintf(format, args...)
		if len(p.values) > 0 {
			s = flatten(p.values...) + ", " + s
		}
		logger.formatter.Format(p.pkg, inLevel, depth+1, s)
	}
}

// LevelAt returns the current log level
func (p *PackageLogger) LevelAt(l LogLevel) bool {
	logger.Lock()
	defer logger.Unlock()
	return p.level >= l
}

// Logf a formatted string at any level between ERROR and TRACE
func (p *PackageLogger) Logf(l LogLevel, format string, args ...interface{}) {
	p.internalLogf(calldepth, l, format, args...)
}

// Log a message at any level between ERROR and TRACE
func (p *PackageLogger) Log(l LogLevel, args ...interface{}) {
	p.internalLog(plain, calldepth, l, args...)
}

// Panic and fatal

// Panicf is implementation for stdlib compatibility
func (p *PackageLogger) Panicf(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	p.internalLog(plain, calldepth, CRITICAL, s)
	panic(s)
}

// Panic is implementation for stdlib compatibility
func (p *PackageLogger) Panic(args ...interface{}) {
	s := fmt.Sprint(args...)
	p.internalLog(plain, calldepth, CRITICAL, s)
	panic(s)
}

// Fatalf is implementation for stdlib compatibility
func (p *PackageLogger) Fatalf(format string, args ...interface{}) {
	p.internalLogf(calldepth, CRITICAL, format, args...)
	os.Exit(1)
}

// Fatal is implementation for stdlib compatibility
func (p *PackageLogger) Fatal(args ...interface{}) {
	s := fmt.Sprint(args...)
	p.internalLog(plain, calldepth, CRITICAL, s)
	os.Exit(1)
}

// Error Functions

// Errorf is implementation for stdlib compatibility
func (p *PackageLogger) Errorf(format string, args ...interface{}) {
	p.internalLogf(calldepth, ERROR, format, args...)
}

// Error is implementation for stdlib compatibility
func (p *PackageLogger) Error(entries ...interface{}) {
	p.internalLog(plain, calldepth, ERROR, entries...)
}

// Warning Functions

// Warningf is implementation for stdlib compatibility
func (p *PackageLogger) Warningf(format string, args ...interface{}) {
	p.internalLogf(calldepth, WARNING, format, args...)
}

// Warning is implementation for stdlib compatibility
func (p *PackageLogger) Warning(entries ...interface{}) {
	p.internalLog(plain, calldepth, WARNING, entries...)
}

// Notice Functions

// Noticef is implementation for stdlib compatibility
func (p *PackageLogger) Noticef(format string, args ...interface{}) {
	p.internalLogf(calldepth, NOTICE, format, args...)
}

// Notice is implementation for stdlib compatibility
func (p *PackageLogger) Notice(entries ...interface{}) {
	p.internalLog(plain, calldepth, NOTICE, entries...)
}

// Info Functions

// Infof is implementation for stdlib compatibility
func (p *PackageLogger) Infof(format string, args ...interface{}) {
	p.internalLogf(calldepth, INFO, format, args...)
}

// Info is implementation for stdlib compatibility
func (p *PackageLogger) Info(entries ...interface{}) {
	p.internalLog(plain, calldepth, INFO, entries...)
}

// KV prints key=value pairs
func (p *PackageLogger) KV(l LogLevel, entries ...interface{}) {
	p.internalLog(kv, calldepth, l, entries...)
}

// Debug Functions

// Debugf is implementation for stdlib compatibility
func (p *PackageLogger) Debugf(format string, args ...interface{}) {
	p.internalLogf(calldepth, DEBUG, format, args...)
}

// Debug is implementation for stdlib compatibility
func (p *PackageLogger) Debug(entries ...interface{}) {
	p.internalLog(plain, calldepth, DEBUG, entries...)
}

// Trace Functions

// Tracef is implementation for stdlib compatibility
func (p *PackageLogger) Tracef(format string, args ...interface{}) {
	p.internalLogf(calldepth, TRACE, format, args...)
}

// Trace is implementation for stdlib compatibility
func (p *PackageLogger) Trace(entries ...interface{}) {
	p.internalLog(plain, calldepth, TRACE, entries...)
}

// Flush the logs
func (p *PackageLogger) Flush() {
	logger.Lock()
	defer logger.Unlock()
	logger.formatter.Flush()
}
