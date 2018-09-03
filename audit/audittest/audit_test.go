package audittest

import (
	"strconv"
	"testing"

	"sync"

	"github.com/go-phorce/pkg/audit"
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
	foo  = audit.New("bob/bob1-1", "1234", srcBar, evtFoo, 12345, "%s:%s", "HASH", "FOO")
	foo2 = audit.New("bob2/bob2-2", "12345", srcBar, evtFoo, 12346, "%s:%s", "HASH", "FOO2")
)

func TestAuditor_Last(t *testing.T) {
	a := Auditor{}
	assert.Equal(t, 0, a.Len())
	a.Event(foo)
	e := a.Last(t)
	assert.Equal(t, "bob/bob1-1", e.Identity())
	assert.Equal(t, "1234", e.ContextID())
	assert.Equal(t, srcBar, e.Source())
	assert.Equal(t, evtFoo, e.EventType())
	assert.Equal(t, uint64(12345), e.RaftIndex())
	assert.Equal(t, "HASH:FOO", e.Message())
	assert.Equal(t, 1, a.Len())
}

func TestAuditor_Close(t *testing.T) {
	a := Auditor{}
	assert.False(t, a.closed)
	a.Close()
	assert.True(t, a.closed)
	a.Event(foo)
	assert.Equal(t, 0, a.Len(), "Expected events not to be recorded after closing the auditor")
}

func TestAuditor_Reset(t *testing.T) {
	// Normal reset
	a := Auditor{}
	assert.Equal(t, 0, a.Len())
	a.Event(foo)
	assert.Equal(t, 1, a.Len())
	a.Reset()
	assert.Equal(t, 0, a.Len())
	// reset after close
	a = Auditor{}
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
	a := Auditor{}
	assert.Equal(t, 0, len(a.events))
	assert.Equal(t, 0, a.Len())
	a.Event(foo)
	a.Event(foo2)
	assert.Equal(t, 2, len(a.events))
	assert.Equal(t, 2, a.Len())
	assert.Equal(t, "12345", a.Last(t).ContextID())
	assert.Equal(t, a.Last(t), a.events[1])
	assert.Equal(t, "1234", a.events[0].ContextID())
	a.Reset()
	assert.Equal(t, 0, len(a.events))
	assert.Equal(t, 0, a.Len())
}

func TestAuditor_MostRecent(t *testing.T) {
	a := Auditor{}
	assert.Equal(t, 0, a.Len())
	a.Event(audit.New("bob/bob1-1", "1234", srcFoo, evtBar, 12345, "%s:%s", "HASH", "FOO"))
	a.Event(foo2)
	e := a.MostRecent(t, evtBar)
	assert.Equal(t, "bob/bob1-1", e.Identity())
	assert.Equal(t, "1234", e.ContextID())
	assert.Equal(t, srcFoo, e.Source())
	assert.Equal(t, evtBar, e.EventType())
	assert.Equal(t, uint64(12345), e.RaftIndex())
	assert.Equal(t, "HASH:FOO", e.Message())
	assert.Equal(t, 2, a.Len())
}

func TestAuditor_ConcurrentSliceAppend(t *testing.T) {
	a := Auditor{}
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
	a := Auditor{}
	assert.Equal(t, 0, a.Len())
	a.events = append(a.events, foo)
	a.events = append(a.events, foo2)
	assert.Equal(t, 2, a.Len())
	var wg sync.WaitGroup
	wg.Add(2)
	actual1 := make([]audit.Event, 0)
	actual2 := make([]audit.Event, 0)
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
