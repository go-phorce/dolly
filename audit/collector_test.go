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

type auditor struct {
	source    string
	eventType string
	identity  string
	contextID string
	raftIndex uint64
	message   string
}

func (a *auditor) Audit(
	source string,
	eventType string,
	identity string,
	contextID string,
	raftIndex uint64,
	message string) {
	a.source = source
	a.eventType = eventType
	a.identity = identity
	a.contextID = contextID
	a.raftIndex = raftIndex
	a.message = message
}

func (a *auditor) Close() error {
	return nil
}

func Test_CollectorSubmit(t *testing.T) {
	dest := auditor{}
	c := Collector{Destination: &dest}
	assert.Empty(t, dest.source)
	assert.Empty(t, dest.eventType)

	c.Audit(srcBar.String(), evtFoo.String(), "alice/alice1-1", "Context-1", 0, "message1")
	assert.Empty(t, dest.source)
	assert.Empty(t, dest.eventType)

	// providing a raft index should update the event submitted to this raft index
	c.Submit(123)
	assert.Equal(t, "alice/alice1-1", dest.identity)
	assert.Equal(t, "Context-1", dest.contextID)
	assert.Equal(t, srcBar.String(), dest.source)
	assert.Equal(t, evtFoo.String(), dest.eventType)
	assert.EqualValues(t, 123, dest.raftIndex)
	assert.Equal(t, "message1", dest.message)

	// calling submit again shouldn't submit anything from the previous submit
	dest.source = ""
	dest.eventType = ""
	c.Submit(123)
	assert.Empty(t, dest.source)
	assert.Empty(t, dest.eventType)

	// Submit with 0 should preseve the raftIndex in the original events
	c.Audit(srcBar.String(), evtFoo.String(), "eve/eve1-1", "Context-2", 123, "message2")
	c.Submit(0)
	assert.Equal(t, "eve/eve1-1", dest.identity)
	assert.Equal(t, "Context-2", dest.contextID)
	assert.Equal(t, srcBar.String(), dest.source)
	assert.Equal(t, evtFoo.String(), dest.eventType)
	assert.EqualValues(t, uint64(123), dest.raftIndex)
	assert.Equal(t, "message2", dest.message)
}

func Test_CollectorClose(t *testing.T) {
	// Closing the collector doesn't submit the events
	dest := auditor{}
	c := Collector{Destination: &dest}
	assert.Empty(t, dest.source)
	assert.Empty(t, dest.eventType)

	c.Audit(srcBar.String(), evtFoo.String(), "alice/alice1-1", "Context-1", 0, "message1")
	assert.Empty(t, dest.source)
	assert.Empty(t, dest.eventType)

	c.Close()
	assert.Empty(t, dest.source)
	assert.Empty(t, dest.eventType)

}
