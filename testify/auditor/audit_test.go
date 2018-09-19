package auditor_test

import (
	"testing"

	"github.com/go-phorce/dolly/testify/auditor"
	"github.com/stretchr/testify/assert"
)

func Test_Audit(t *testing.T) {
	a := auditor.NewInMemory()
	defer a.Close()
	events := a.GetAll()
	assert.NotNil(t, events)
	assert.Empty(t, events)

	a.Audit("source1", "evt1", "bob/bob1-1", "1234", 1234, "message1")
	a.Audit("source2", "evt2", "bob/bob1-1", "2345", 1235, "message2")

	events = a.GetAll()
	assert.Equal(t, 2, len(events))

	evt := a.Get(0)
	assert.Equal(t, "source1", evt.Source)
	assert.Equal(t, "evt1", evt.EventType)
	evt = a.Get(1)
	assert.Equal(t, "source2", evt.Source)
	assert.Equal(t, "evt2", evt.EventType)

	a.Reset()
	events = a.GetAll()
	assert.NotNil(t, events)
	assert.Empty(t, events)

	assert.NoError(t, a.Close())
}
