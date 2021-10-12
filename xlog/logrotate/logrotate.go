package logrotate

// Copyright 2018 salesforce.com
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

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-phorce/dolly/xlog"
	"github.com/pkg/errors"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type logrotator struct {
	oldFormatter xlog.Formatter
	logger       io.Writer
	channel      *ChannelWriter
	closed       bool
}

// Initialize creates a lumberjack log rotator and redirects logs output to it.
// To ensure that any queued/buffered but unwritten log entries are flushed to disk
// call Stop() on the returned stopper before exiting the process.
// Once stopped, you can't resume the logger, you need to create a new one.
func Initialize(logFolder, baseFilename string, maxAge, maxSize int, buffered bool, extraSink io.Writer) (io.Closer, error) {
	err := os.MkdirAll(logFolder, 0755)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fileWriter := lumberjack.Logger{
		Filename: filepath.Join(logFolder, baseFilename+".log"),
		MaxAge:   maxAge,
		MaxSize:  maxSize,
	}

	l := &logrotator{
		logger:       bufio.NewWriterSize(&fileWriter, 8192),
		oldFormatter: xlog.GetFormatter(),
	}

	if extraSink != nil {
		l.logger = io.MultiWriter(l.logger, extraSink)
	}

	if buffered {
		l.channel = NewChannelWriter(l.logger, 256, time.Second)
	}

	xlog.SetFormatter(xlog.NewDefaultFormatter(l.destination()))

	return l, nil
}

func (c *logrotator) destination() io.Writer {
	if c.channel != nil {
		return c.channel
	}
	return c.logger
}

// Close will ensure that queued/buffered but unwritten log entries are flushed to disk
func (c *logrotator) Close() error {
	if c.closed {
		return errors.New("already closed")
	}
	c.closed = true

	// restore output
	xlog.SetFormatter(c.oldFormatter)

	if c.channel != nil {
		c.channel.Stop()
		c.channel = nil
	}
	return nil
}
