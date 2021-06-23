// Package xlog has slight modifications on the original code,
// adding ability to specify log lever per package,
// and exposing Logger interface, not an implementation structure.
//
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
package xlog

// Logger interface for generic logger
type Logger interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})

	Panic(args ...interface{})
	Panicf(format string, args ...interface{})

	Info(entries ...interface{})
	Infof(format string, args ...interface{})

	Error(entries ...interface{})
	Errorf(format string, args ...interface{})

	Warning(entries ...interface{})
	Warningf(format string, args ...interface{})

	Notice(entries ...interface{})
	Noticef(format string, args ...interface{})

	Debug(entries ...interface{})
	Debugf(format string, args ...interface{})

	Trace(entries ...interface{})
	Tracef(format string, args ...interface{})

	// KV logs entries in "key1=value1, ..., keyN=valueN" format
	KV(level LogLevel, entries ...interface{})

	// WithValues adds some key-value pairs of context to a logger.
	// See Info for documentation on how key/value pairs work.
	WithValues(keysAndValues ...interface{}) Logger
}
