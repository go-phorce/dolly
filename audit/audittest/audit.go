// Package audittest provides helper methods for writing tests that interact
// with the audit logging service.
package audittest

import (
	"sync"
	"testing"

	"github.com/ekspand/pkg/audit"
	"github.com/stretchr/testify/assert"
)

// Auditor is a audit.Auditor implemention that tracks in memory
// the audit event raised, allowing tests to easily verify that audit
// events were triggered
// Auditor can be safely shared across goroutines
type Auditor struct {
	events []audit.Event
	sync.Mutex
	closed bool
}

// Get returns the item at specified idx in the slice
// Ordering in audit events may not work as expected
// when multiple go-routines are appending events
// Use this method with caution
func (a *Auditor) Get(idx int) audit.Event {
	a.Lock()
	defer a.Unlock()
	return a.events[idx]
}

// GetAll returns a cloned copy of all the events
// Ordering in audit events may not work as expected
// when multiple go-routines are appending events
// Use this method with caution
func (a *Auditor) GetAll() []audit.Event {
	a.Lock()
	defer a.Unlock()
	result := make([]audit.Event, len(a.events))
	copy(result, a.events)
	return result
}

// Event records a new audit event
func (a *Auditor) Event(e audit.Event) {
	a.Lock()
	defer a.Unlock()
	if !a.closed {
		isNil := a.events == nil
		if isNil {
			a.events = make([]audit.Event, 0, 10)
		}
		a.events = append(a.events, e)
	}
}

// Close closes the auditor
// After this, auditor cannot track new audit events
// However, the events audited before calling Close() can still be queried
func (a *Auditor) Close() error {
	a.Lock()
	defer a.Unlock()
	a.closed = true
	return nil
}

// Len returns the number of captured audit events
func (a *Auditor) Len() int {
	a.Lock()
	defer a.Unlock()
	return len(a.events)
}

// Last returns the most recently recorded event, if no events
// have been recorded, this'll be flagged as an error on the
// supplied testing.T and nil returned
func (a *Auditor) Last(t *testing.T) audit.Event {
	a.Lock()
	defer a.Unlock()
	length := len(a.events)
	if assert.NotEqual(t, 0, length, "No Audit items recorded") {
		return a.events[length-1]
	}
	return nil
}

// MostRecent returns the newest audit event of the indicated event type
// or if it can't find one, will flag an error on the test.
func (a *Auditor) MostRecent(t *testing.T, eventType audit.EventType) audit.Event {
	a.Lock()
	defer a.Unlock()
	length := len(a.events)
	for i := length - 1; i >= 0; i-- {
		if a.events[i].EventType() == eventType {
			return a.events[i]
		}
	}
	assert.Fail(t, "Unable to find an audit event of type %v in captured items", eventType)
	return nil
}

// LastEvents returns all events matching indicated event type in order of most recent to least recent
func (a *Auditor) LastEvents(t *testing.T, eventType audit.EventType) []audit.Event {
	a.Lock()
	defer a.Unlock()
	var matches []audit.Event
	length := len(a.events)
	for i := length - 1; i >= 0; i-- {
		if a.events[i].EventType() == eventType {
			matches = append(matches, a.events[i])
		}
	}
	return matches
}

// Reset will clear out any previous captured audit events
// and also opens the auditor if its closed
func (a *Auditor) Reset() {
	a.Lock()
	defer a.Unlock()
	a.events = make([]audit.Event, 0, 10)
	a.closed = false
}
