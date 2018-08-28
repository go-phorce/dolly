package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Eventf(t *testing.T) {
	a := auditor{}
	a.Event(New("alice/alice1-1", "Context-1", srcBar, evtFoo, 424242, "%s.%d", "HASH", 123))
	require.NotNil(t, a.event)
	e := a.event
	assert.Equal(t, "alice/alice1-1", e.Identity())
	assert.Equal(t, "Context-1", e.ContextID())
	assert.Equal(t, srcBar, e.Source())
	assert.Equal(t, evtFoo, e.EventType())
	assert.Equal(t, uint64(424242), e.RaftIndex())
	assert.Equal(t, "HASH.123", e.Message())
}

func Test_SetRaftIndex(t *testing.T) {
	e := EventInfo{}
	assert.True(t, e.SetRaftIndex(10))
	assert.True(t, e.SetRaftIndex(10))
	assert.False(t, e.SetRaftIndex(11))
}

type auditor struct {
	event Event
}

func (a *auditor) Event(e Event) {
	a.event = e
}

func (a *auditor) Close() error {
	return nil
}
