// Package audittest provides helper methods for writing tests that interact
// with the audit logging service.
package audittest

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// event provides a default impl of event
type event struct {
	identity  string
	contextID string
	source    string
	eventType string
	raftIndex uint64
	message   string
}

// auditor is a audit.auditor implemention that tracks in memory
// the audit event raised, allowing tests to easily verify that audit
// events were triggered
// auditor can be safely shared across goroutines
type auditor struct {
	events []*event
	sync.Mutex
	closed bool
}

// Get returns the item at specified idx in the slice
// Ordering in audit events may not work as expected
// when multiple go-routines are appending events
// Use this method with caution
func (a *auditor) Get(idx int) *event {
	a.Lock()
	defer a.Unlock()
	return a.events[idx]
}

// GetAll returns a cloned copy of all the events
// Ordering in audit events may not work as expected
// when multiple go-routines are appending events
// Use this method with caution
func (a *auditor) GetAll() []*event {
	a.Lock()
	defer a.Unlock()
	result := make([]*event, len(a.events))
	copy(result, a.events)
	return result
}

// Event records a new audit event
func (a *auditor) Event(e *event) {
	a.Lock()
	defer a.Unlock()
	if !a.closed {
		isNil := a.events == nil
		if isNil {
			a.events = make([]*event, 0, 10)
		}
		a.events = append(a.events, e)
	}
}

// Close closes the auditor
// After this, auditor cannot track new audit events
// However, the events audited before calling Close() can still be queried
func (a *auditor) Close() error {
	a.Lock()
	defer a.Unlock()
	a.closed = true
	return nil
}

// Len returns the number of captured audit events
func (a *auditor) Len() int {
	a.Lock()
	defer a.Unlock()
	return len(a.events)
}

// Last returns the most recently recorded event, if no events
// have been recorded, this'll be flagged as an error on the
// supplied testing.T and nil returned
func (a *auditor) Last(t *testing.T) *event {
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
func (a *auditor) MostRecent(t *testing.T, eventType string) *event {
	a.Lock()
	defer a.Unlock()
	length := len(a.events)
	for i := length - 1; i >= 0; i-- {
		if a.events[i].eventType == eventType {
			return a.events[i]
		}
	}
	assert.Failf(t, "Unable to find an audit event of type '%s' in captured items", eventType)
	return nil
}

// LastEvents returns all events matching indicated event type in order of most recent to least recent
func (a *auditor) LastEvents(t *testing.T, eventType string) []*event {
	a.Lock()
	defer a.Unlock()
	var matches []*event
	length := len(a.events)
	for i := length - 1; i >= 0; i-- {
		if a.events[i].eventType == eventType {
			matches = append(matches, a.events[i])
		}
	}
	return matches
}

// Reset will clear out any previous captured audit events
// and also opens the auditor if its closed
func (a *auditor) Reset() {
	a.Lock()
	defer a.Unlock()
	a.events = make([]*event, 0, 10)
	a.closed = false
}
