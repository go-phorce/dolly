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

import (
	"log"
)

// NilLogger does not produce any output
type NilLogger struct {
}

// NewNilLogger creates new nil logger
func NewNilLogger() Logger {
	return &NilLogger{}
}

// Print does nothing
func (l *NilLogger) Print(args ...interface{}) {}

// Printf does nothing
func (l *NilLogger) Printf(format string, args ...interface{}) {}

// Println does nothing
func (l *NilLogger) Println(args ...interface{}) {}

// Fatal does nothing
func (l *NilLogger) Fatal(args ...interface{}) {}

// Fatalf does nothing
func (l *NilLogger) Fatalf(format string, args ...interface{}) {}

// Fatalln does nothing
func (l *NilLogger) Fatalln(args ...interface{}) {}

// Panic does nothing
func (l *NilLogger) Panic(args ...interface{}) {
	log.Panic(args...)
}

// Panicf does nothing
func (l *NilLogger) Panicf(format string, args ...interface{}) {
	log.Panicf(format, args...)
}

// Info does nothing
func (l *NilLogger) Info(entries ...interface{}) {}

// Infof does nothing
func (l *NilLogger) Infof(format string, args ...interface{}) {}

// Error does nothing
func (l *NilLogger) Error(entries ...interface{}) {}

// Errorf does nothing
func (l *NilLogger) Errorf(format string, args ...interface{}) {}

// Warning does nothing
func (l *NilLogger) Warning(entries ...interface{}) {}

// Warningf does nothing
func (l *NilLogger) Warningf(format string, args ...interface{}) {}

// Notice does nothing
func (l *NilLogger) Notice(entries ...interface{}) {}

// Noticef does nothing
func (l *NilLogger) Noticef(format string, args ...interface{}) {}

// Debug does nothing
func (l *NilLogger) Debug(entries ...interface{}) {}

// Debugf does nothing
func (l *NilLogger) Debugf(format string, args ...interface{}) {}

// Trace does nothing
func (l *NilLogger) Trace(entries ...interface{}) {}

// Tracef does nothing
func (l *NilLogger) Tracef(format string, args ...interface{}) {}
