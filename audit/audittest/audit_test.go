package audittest

import (
	"strconv"
	"sync"
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

var (
	foo  = &event{"bob/bob1-1", "1234-2345", srcBar.String(), evtFoo.String(), 12345, "message1"}
	foo2 = &event{"bob2/bob2-2", "2345-3456", srcBar.String(), evtFoo.String(), 12346, "message2"}
)

func TestAuditor_Last(t *testing.T) {
	a := auditor{}
	assert.Equal(t, 0, a.Len())
	a.Event(foo)
	e := a.Last(t)
	assert.Equal(t, "bob/bob1-1", e.identity)
	assert.Equal(t, "1234-2345", e.contextID)
	assert.Equal(t, srcBar.String(), e.source)
	assert.Equal(t, evtFoo.String(), e.eventType)
	assert.Equal(t, uint64(12345), e.raftIndex)
	assert.Equal(t, "message1", e.message)
	assert.Equal(t, 1, a.Len())
}

func TestAuditor_Close(t *testing.T) {
	a := auditor{}
	assert.False(t, a.closed)
	a.Close()
	assert.True(t, a.closed)
	a.Event(foo)
	assert.Equal(t, 0, a.Len(), "Expected events not to be recorded after closing the auditor")
}

func TestAuditor_Reset(t *testing.T) {
	// Normal reset
	a := auditor{}
	assert.Equal(t, 0, a.Len())
	a.Event(foo)
	assert.Equal(t, 1, a.Len())
	a.Reset()
	assert.Equal(t, 0, a.Len())
	// reset after close
	a = auditor{}
	assert.Equal(t, 0, a.Len())
	a.Event(foo)
	assert.Equal(t, 1, a.Len())
	a.Close()
	assert.Equal(t, true, a.closed)
	a.Reset()
	assert.Equal(t, false, a.closed)
	a.Event(foo)
	assert.Equal(t, 1, a.Len())
}

func TestAuditor_MultipleEvents(t *testing.T) {
	a := auditor{}
	assert.Equal(t, 0, len(a.events))
	assert.Equal(t, 0, a.Len())
	a.Event(foo)
	a.Event(foo2)
	assert.Equal(t, 2, len(a.events))
	assert.Equal(t, 2, a.Len())
	assert.Equal(t, "2345-3456", a.Last(t).contextID)
	assert.Equal(t, a.Last(t), a.events[1])
	assert.Equal(t, "1234-2345", a.events[0].contextID)
	a.Reset()
	assert.Equal(t, 0, len(a.events))
	assert.Equal(t, 0, a.Len())
}

func TestAuditor_MostRecent(t *testing.T) {
	a := auditor{}
	assert.Equal(t, 0, a.Len())
	a.Event(foo)
	a.Event(foo2)
	e := a.MostRecent(t, foo2.eventType)
	assert.Equal(t, foo2.identity, e.identity)
	assert.Equal(t, foo2.contextID, e.contextID)
	assert.Equal(t, foo2.source, e.source)
	assert.Equal(t, foo2.eventType, e.eventType)
	assert.Equal(t, foo2.raftIndex, e.raftIndex)
	assert.Equal(t, foo2.message, e.message)
	assert.Equal(t, 2, a.Len())
}

func TestAuditor_ConcurrentSliceAppend(t *testing.T) {
	a := auditor{}
	assert.Equal(t, 0, a.Len())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		a.Event(foo)
		wg.Done()
	}()
	go func() {
		a.Event(foo2)
		wg.Done()
	}()
	wg.Wait()
	assert.Equal(t, 2, a.Len())
	// trying to figure out the order of events since
	// we cant control the order in which goroutines append
	firstExpected := foo
	secondExpected := foo2
	if a.events[0] == foo2 {
		firstExpected = foo2
		secondExpected = foo
	}
	assert.Equal(t, a.events[0], firstExpected)
	assert.Equal(t, a.events[1], secondExpected)
}

func TestAuditor_GetAll(t *testing.T) {
	a := auditor{}
	assert.Equal(t, 0, a.Len())
	a.events = append(a.events, foo)
	a.events = append(a.events, foo2)
	assert.Equal(t, 2, a.Len())
	var wg sync.WaitGroup
	wg.Add(2)
	actual1 := make([]*event, 0)
	actual2 := make([]*event, 0)
	// concurrently read the audit events
	go func() {
		for _, e := range a.GetAll() {
			actual1 = append(actual1, e)
		}
		wg.Done()
	}()
	go func() {
		for _, e := range a.GetAll() {
			actual2 = append(actual2, e)
		}
		wg.Done()
	}()
	wg.Wait()
	assert.Equal(t, a.events, actual1)
	assert.Equal(t, a.events, actual2)
}
