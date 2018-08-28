package audit

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testSource int

const (
	srcFoo testSource = iota
	srcBar
)

func (i testSource) ID() int {
	return int(i)
}

func (i testSource) String() string {
	return "src" + strconv.Itoa(int(i))
}

type testEventType int

const (
	evtBar testEventType = iota
	evtFoo
)

func (i testEventType) ID() int {
	return int(i)
}

func (i testEventType) String() string {
	return "type" + strconv.Itoa(int(i))
}

func Test_CollectorSubmit(t *testing.T) {
	dest := auditor{}
	c := Collector{Destination: &dest}
	assert.Nil(t, dest.event)

	c.Event(New("alice/alice1-1", "Context-1", srcBar, evtFoo, 0, "%s.%d", "HASH", 123))
	assert.Nil(t, dest.event)

	// providing a raft index should update the event submitted to this raft index
	c.Submit(123)
	assert.NotNil(t, dest.event)
	assert.Equal(t, "alice/alice1-1", dest.event.Identity())
	assert.Equal(t, "Context-1", dest.event.ContextID())
	assert.Equal(t, srcBar, dest.event.Source())
	assert.Equal(t, evtFoo, dest.event.EventType())
	assert.EqualValues(t, 123, dest.event.RaftIndex())
	assert.Equal(t, "HASH.123", dest.event.Message())

	// calling submit again shouldn't submit anything from the previous submit
	dest.event = nil
	c.Submit(123)
	assert.Nil(t, dest.event)

	// Submit with 0 should preseve the raftIndex in the original events
	c.Event(New("eve/eve1-1", "Context-2", srcBar, evtFoo, 124, "%s.%d", "HASH", 121))
	c.Submit(0)
	assert.NotNil(t, dest.event)
	assert.Equal(t, "eve/eve1-1", dest.event.Identity())
	assert.Equal(t, "Context-2", dest.event.ContextID())
	assert.Equal(t, srcBar, dest.event.Source())
	assert.Equal(t, evtFoo, dest.event.EventType())
	assert.EqualValues(t, 124, dest.event.RaftIndex())
	assert.Equal(t, "HASH.121", dest.event.Message())
}

func Test_CollectorClose(t *testing.T) {
	// Closing the collector doesn't submit the events
	dest := auditor{}
	c := Collector{Destination: &dest}
	assert.Nil(t, dest.event)
	c.Event(New("alice/alice1-1", "Context-1", srcBar, evtFoo, 424242, "%s.%d", "HASH", 123))
	assert.Nil(t, dest.event)
	c.Close()
	assert.Nil(t, dest.event)
}

func Test_CollectorNoRaftIndexer(t *testing.T) {
	e := new(eventInfoNoSet)
	w := withRaftIndex(e, 1234)
	assert.Equal(t, e.Identity(), w.Identity())
	assert.Equal(t, e.ContextID(), w.ContextID())
	assert.Equal(t, e.Source(), w.Source())
	assert.Equal(t, e.EventType(), w.EventType())
	assert.Equal(t, e.Message(), w.Message())
	assert.Equal(t, uint64(1234), w.RaftIndex())
}

func Test_CollectorRaftIndex(t *testing.T) {
	e := New("alice/alice1-1", "Context-1", srcBar, evtFoo, 0, "hello")
	w := withRaftIndex(e, 1234)
	assert.Exactly(t, e, w)
	assert.Equal(t, uint64(1234), w.RaftIndex())
}

type eventInfoNoSet struct {
}

func (e *eventInfoNoSet) Identity() string {
	return "bob/bob1-1"
}
func (e *eventInfoNoSet) ContextID() string {
	return "1"
}
func (e *eventInfoNoSet) Source() Source {
	return srcBar
}
func (e *eventInfoNoSet) EventType() EventType {
	return evtFoo
}
func (e *eventInfoNoSet) RaftIndex() uint64 {
	return 10
}
func (e *eventInfoNoSet) Message() string {
	return "hello"
}
