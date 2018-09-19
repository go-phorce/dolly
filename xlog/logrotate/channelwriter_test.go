package logrotate

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

type testWriter struct {
	lock   sync.Mutex
	writes [][]byte
}

func (t *testWriter) Write(b []byte) (int, error) {
	c := make([]byte, len(b))
	copy(c, b)
	t.lock.Lock()
	defer t.lock.Unlock()
	t.writes = append(t.writes, c)
	return len(b), nil
}

func (t *testWriter) NumWrites() int {
	t.lock.Lock()
	defer t.lock.Unlock()
	return len(t.writes)
}

type testFlushWriter struct {
	testWriter
	flushCount int32
}

func (tf *testFlushWriter) Flush() error {
	tf.lock.Lock()
	defer tf.lock.Unlock()
	tf.flushCount++
	return nil
}

func (tf *testFlushWriter) NumFlushes() int32 {
	tf.lock.Lock()
	defer tf.lock.Unlock()
	return tf.flushCount
}

func TestChannelWriter_Flushes(t *testing.T) {
	dest := &testFlushWriter{}
	cw := NewChannelWriter(dest, 200, time.Millisecond)
	defer cw.Stop()
	waitUntil := time.Now().Add(time.Second)
	for dest.NumFlushes() == 0 {
		if time.Now().After(waitUntil) {
			t.Fatalf("Gave up waiting to be flushed")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestChannelWriter_BufioIsFlushable(t *testing.T) {
	dest := &testWriter{}
	w := bufio.NewWriter(dest)
	var f flushable
	f = w // verify that our flushable includes bufio.Writer
	if f == nil {
		t.Errorf("bufio.Writer should be a flushable")
	}
}

func TestChannelWriter_Writes(t *testing.T) {
	dest := &testWriter{}
	cw := NewChannelWriter(dest, 200, time.Millisecond)
	var writer io.Writer = cw // ensure cw can be used as an io.Writer

	defer cw.Stop()
	numMessages := 400
	exp := make([][]byte, 0, numMessages)
	for i := 0; i < numMessages; i++ {
		w := []byte(fmt.Sprintf("message %d", i))
		wcopy := append([]byte(nil), w...)
		exp = append(exp, wcopy)
		writer.Write(w)
		// ensure that the writer doesn't hold onto the bytes the general
		// expectation for io.Write is that the caller owns the data after
		// write returns
		w[0] = 'X'
	}
	if cw.IsStopped() {
		t.Errorf("ChannelWriter.IsStopped() reports true, but we haven't called Stop() yet")
	}
	waitUntil := time.Now().Add(time.Second)
	for dest.NumWrites() < numMessages {
		if time.Now().After(waitUntil) {
			t.Fatalf("Gave up waiting for background writes to turn up, got %d out of %d", dest.NumWrites(), numMessages)
		}
		if dest.NumWrites() > 0 {
			// Stop should drain the channel, so we should still get all our expected messages
			cw.Stop()
			if !cw.IsStopped() {
				t.Errorf("Called Stop() on the ChannelWriter, but IsStopped() says no!")
			}
		}
		time.Sleep(time.Millisecond)
	}
	dest.lock.Lock()
	defer dest.lock.Unlock()
	for i, e := range exp {
		if !bytes.Equal(e, dest.writes[i]) {
			t.Errorf("Write %d: expecting: %s, got %s", i, e, dest.writes[i])
		}
	}
}
