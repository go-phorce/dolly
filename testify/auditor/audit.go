package auditor

import (
	"sync"
)

// Event provides a default impl of Event
type Event struct {
	Identity  string
	ContextID string
	Source    string
	EventType string
	RaftIndex uint64
	Message   string
}

// InMemory is ahelper Auditor to keep events in memory
type InMemory struct {
	events []*Event
	sync.Mutex
	closed bool
}

// NewInMemory creates new Auditor that captures events in memory
func NewInMemory() *InMemory {
	return &InMemory{
		events: make([]*Event, 0, 10),
	}
}

// Get returns the item at specified idx in the slice
// Ordering in audit events may not work as expected
// when multiple go-routines are appending events
// Use this method with caution
func (a *InMemory) Get(idx int) *Event {
	a.Lock()
	defer a.Unlock()
	return a.events[idx]
}

// Find returns first event satisfying the filter
func (a *InMemory) Find(source, eventType string) *Event {
	a.Lock()
	defer a.Unlock()
	for _, e := range a.events {
		if e.Source == source && e.EventType == eventType {
			return e
		}
	}
	return nil
}

// GetAll returns a cloned copy of all the events
// Ordering in audit events may not work as expected
// when multiple go-routines are appending events
// Use this method with caution
func (a *InMemory) GetAll() []*Event {
	a.Lock()
	defer a.Unlock()
	result := make([]*Event, len(a.events))
	copy(result, a.events)
	return result
}

// Event records a new audit event
func (a *InMemory) Event(e *Event) {
	a.Lock()
	defer a.Unlock()
	if !a.closed {
		isNil := a.events == nil
		if isNil {
			a.events = make([]*Event, 0, 10)
		}
		a.events = append(a.events, e)
	}
}

// Reset will remove any collected events from this collector.
func (a *InMemory) Reset() {
	a.Lock()
	defer a.Unlock()
	a.closed = false
	a.events = make([]*Event, 0, 10)
}

// Len returns the number of captured audit events
func (a *InMemory) Len() int {
	a.Lock()
	defer a.Unlock()
	return len(a.events)
}

// Close closes the auditor
// After this, auditor cannot track new audit events
// However, the events audited before calling Close() can still be queried
func (a *InMemory) Close() error {
	a.Lock()
	defer a.Unlock()
	a.closed = true
	return nil
}

// Audit interface impl
func (a *InMemory) Audit(
	source string,
	eventType string,
	identity string,
	contextID string,
	raftIndex uint64,
	message string,
) {
	a.Event(&Event{
		Source:    source,
		EventType: eventType,
		Identity:  identity,
		ContextID: contextID,
		RaftIndex: raftIndex,
		Message:   message,
	})
}
