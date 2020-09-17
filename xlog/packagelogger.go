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
	pkg   string
	level LogLevel
}

const calldepth = 2

func (p *PackageLogger) internalLog(depth int, inLevel LogLevel, entries ...interface{}) {
	logger.Lock()
	defer logger.Unlock()
	if inLevel != CRITICAL && p.level < inLevel {
		return
	}
	if logger.formatter != nil {
		logger.formatter.Format(p.pkg, inLevel, depth+1, entries...)
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
	p.internalLog(calldepth, l, fmt.Sprintf(format, args...))
}

// Log a message at any level between ERROR and TRACE
func (p *PackageLogger) Log(l LogLevel, args ...interface{}) {
	p.internalLog(calldepth, l, fmt.Sprint(args...))
}

// log stdlib compatibility

// Println is implementation for stdlib compatibility
func (p *PackageLogger) Println(args ...interface{}) {
	p.internalLog(calldepth, INFO, fmt.Sprintln(args...))
}

// Printf is implementation for stdlib compatibility
func (p *PackageLogger) Printf(format string, args ...interface{}) {
	p.Logf(INFO, format, args...)
}

// Print is implementation for stdlib compatibility
func (p *PackageLogger) Print(args ...interface{}) {
	p.internalLog(calldepth, INFO, fmt.Sprint(args...))
}

// Panic and fatal

// Panicf is implementation for stdlib compatibility
func (p *PackageLogger) Panicf(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	p.internalLog(calldepth, CRITICAL, s)
	panic(s)
}

// Panic is implementation for stdlib compatibility
func (p *PackageLogger) Panic(args ...interface{}) {
	s := fmt.Sprint(args...)
	p.internalLog(calldepth, CRITICAL, s)
	panic(s)
}

// Fatalf is implementation for stdlib compatibility
func (p *PackageLogger) Fatalf(format string, args ...interface{}) {
	p.Logf(CRITICAL, format, args...)
	os.Exit(1)
}

// Fatal is implementation for stdlib compatibility
func (p *PackageLogger) Fatal(args ...interface{}) {
	s := fmt.Sprint(args...)
	p.internalLog(calldepth, CRITICAL, s)
	os.Exit(1)
}

// Fatalln is implementation for stdlib compatibility
func (p *PackageLogger) Fatalln(args ...interface{}) {
	s := fmt.Sprintln(args...)
	p.internalLog(calldepth, CRITICAL, s)
	os.Exit(1)
}

// Error Functions

// Errorf is implementation for stdlib compatibility
func (p *PackageLogger) Errorf(format string, args ...interface{}) {
	p.Logf(ERROR, format, args...)
}

// Error is implementation for stdlib compatibility
func (p *PackageLogger) Error(entries ...interface{}) {
	p.internalLog(calldepth, ERROR, entries...)
}

// Warning Functions

// Warningf is implementation for stdlib compatibility
func (p *PackageLogger) Warningf(format string, args ...interface{}) {
	if p.level < WARNING {
		return
	}
	p.Logf(WARNING, format, args...)
}

// Warning is implementation for stdlib compatibility
func (p *PackageLogger) Warning(entries ...interface{}) {
	if p.level < WARNING {
		return
	}
	p.internalLog(calldepth, WARNING, entries...)
}

// Notice Functions

// Noticef is implementation for stdlib compatibility
func (p *PackageLogger) Noticef(format string, args ...interface{}) {
	if p.level < NOTICE {
		return
	}
	p.Logf(NOTICE, format, args...)
}

// Notice is implementation for stdlib compatibility
func (p *PackageLogger) Notice(entries ...interface{}) {
	if p.level < NOTICE {
		return
	}
	p.internalLog(calldepth, NOTICE, entries...)
}

// Info Functions

// Infof is implementation for stdlib compatibility
func (p *PackageLogger) Infof(format string, args ...interface{}) {
	if p.level < INFO {
		return
	}
	p.Logf(INFO, format, args...)
}

// Info is implementation for stdlib compatibility
func (p *PackageLogger) Info(entries ...interface{}) {
	if p.level < INFO {
		return
	}
	p.internalLog(calldepth, INFO, entries...)
}

// Debug Functions

// Debugf is implementation for stdlib compatibility
func (p *PackageLogger) Debugf(format string, args ...interface{}) {
	if p.level < DEBUG {
		return
	}
	p.Logf(DEBUG, format, args...)
}

// Debug is implementation for stdlib compatibility
func (p *PackageLogger) Debug(entries ...interface{}) {
	if p.level < DEBUG {
		return
	}
	p.internalLog(calldepth, DEBUG, entries...)
}

// Trace Functions

// Tracef is implementation for stdlib compatibility
func (p *PackageLogger) Tracef(format string, args ...interface{}) {
	if p.level < TRACE {
		return
	}
	p.Logf(TRACE, format, args...)
}

// Trace is implementation for stdlib compatibility
func (p *PackageLogger) Trace(entries ...interface{}) {
	if p.level < TRACE {
		return
	}
	p.internalLog(calldepth, TRACE, entries...)
}

// Flush the logs
func (p *PackageLogger) Flush() {
	logger.Lock()
	defer logger.Unlock()
	logger.formatter.Flush()
}
