// Package logrotate implements additional functionality for io writers & closers
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
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// ChannelWriter provides an io.Writer that defers the write to a background
// go routine. You might for example use this for a log.Logger destination
type ChannelWriter struct {
	write    chan []byte
	stop     chan bool
	stopped  chan bool
	running  uint32
	buffPool sync.Pool
}

// NewChannelWriter provides an instance of io.Writer that
// forwards all write over a channel to a background go routine
// that does the actual write, this can stop disk I/O cluttering
// up app processing. [at the potential risk of loosing some
// writes during a crash]
//
// dest is the io.Writer that we're wrapping
// bufferDepth controls the size of the channel buffer (if this buffer fills, it'll start to block the writers)
// flushInterval if the writer is a bufio.Writer (or any other writer with a Flush() error method), then we'll flush at this interval when there are no writes.
// you can pass zero for this if you don't want this behavour
func NewChannelWriter(dest io.Writer, bufferDepth int, flushInterval time.Duration) *ChannelWriter {
	cw := ChannelWriter{
		write:   make(chan []byte, bufferDepth),
		stop:    make(chan bool),
		stopped: make(chan bool),
		running: 1,
	}
	cw.buffPool.New = func() interface{} {
		return make([]byte, 0, 256)
	}
	go cw.listen(dest, flushInterval)
	return &cw
}

// IsStopped returns true if this ChannelWriter has been stopped
func (cw *ChannelWriter) IsStopped() bool {
	return atomic.LoadUint32(&cw.running) == 0
}

// Stop tells the background writer to stop processing [if its running]
// Once stopped you can not restart it, it is expected that you throw
// this away once stopped.
// Stop will drain the current contents of the write channel before stopping
// Stop() will block until the channel is drained and the output flushed.
func (cw *ChannelWriter) Stop() {
	if atomic.CompareAndSwapUint32(&cw.running, 1, 0) {
		cw.stop <- true
		<-cw.stopped // wait til we've finished draining the queue and have flushed the output
	}
}

// Write implements the io.Writer interface
func (cw *ChannelWriter) Write(d []byte) (int, error) {
	// the documented sematics of Write are that we can't hold onto the supplied
	// bytes past the end of the function, so we need to create a copy to Put
	// on the channel.
	buff := cw.buffPool.Get().([]byte)
	buff = append(buff[:0], d...)
	cw.write <- buff
	return len(d), nil
}

type flushable interface {
	Flush() error
}

// listen is our background go-routine, it reads from the channel and does
// the writes. It also flushes on a regular basis if configured to do so.
func (cw *ChannelWriter) listen(dest io.Writer, flushInterval time.Duration) {
	defer func() {
		cw.stopped <- true
	}()
	var flushChan <-chan time.Time
	flusher, canFlush := dest.(flushable)
	if canFlush && flushInterval > 0 {
		ft := time.NewTicker(flushInterval)
		flushChan = ft.C
		defer ft.Stop()
	} else {
		flushChan = make(chan time.Time)
	}
	for true {
		select {
		case <-flushChan:
			if canFlush {
				flusher.Flush()
			}
		case b := <-cw.write:
			dest.Write(b)
			cw.buffPool.Put(b)
		case <-cw.stop:
			// drain what's left of the Write channel
			for true {
				select {
				case b := <-cw.write:
					dest.Write(b)
				default:
					if canFlush {
						flusher.Flush()
					}
					return
				}
			}
		}
	}
}
