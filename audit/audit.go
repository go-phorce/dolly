package audit

import (
	"fmt"
	"io"
)

// Source declares the general area that the audit event was raised from
type Source interface {
	ID() int
	String() string
}

// EventType defines a specific event from the Source
type EventType interface {
	ID() int
	String() string
}

// Auditor defines an interface that can receive information about audit events
type Auditor interface {
	// Call at shutdown to cleanly close the audit destination
	io.Closer

	// Call Event to record a new Auditable event
	Event(e Event)
}

// Event defines an abstract source of the details of an audit event
type Event interface {
	// Identity returns the identity of the user that triggered this event, typically this is <role>/<cn>
	Identity() string
	// ContextID is the request ContextID that the event was triggered in [this can be used for cross service correlation of logs]
	ContextID() string
	// Source indicates the area that the event was triggered by
	Source() Source
	// EventType indicates the specific event that occured
	EventType() EventType
	// RaftIndex indicates the index# of the raft log in RAFT that the event occured in [if applicable]
	RaftIndex() uint64
	// Message contains any additional information about this event that is eventType specific
	Message() string
}

// EventInfo provides a default impl of Event
type EventInfo struct {
	identity  string
	contextID string
	src       Source
	eventType EventType
	raftIndex uint64
	message   string
}

// New returns a newly constructed Event instance
//
// identity     identity of the source
// contextID    context of the request
// src          the source of the event, typically one of the specific APIs, or built-in integrations
// event        the specific event that happend
// raftIndex    if relevant the raft index from RAFT that this event occured in
// message/vals additional info to be captured, will be event specific
func New(identity, contextID string, src Source, event EventType, raftIndex uint64, message string, vals ...interface{}) Event {
	return &EventInfo{
		identity:  identity,
		contextID: contextID,
		src:       src,
		eventType: event,
		raftIndex: raftIndex,
		message:   fmt.Sprintf(message, vals...),
	}
}

// Identity returns the user [typically the Client Cert role/commonName] that triggered this event
func (e *EventInfo) Identity() string {
	return e.identity
}

// ContextID returns a per request contextID that can be used for cross service diagnostics
func (e *EventInfo) ContextID() string {
	return e.contextID
}

// Source returns the area that generated the event
func (e *EventInfo) Source() Source {
	return e.src
}

// EventType returns the specific event
func (e *EventInfo) EventType() EventType {
	return e.eventType
}

// RaftIndex returns the raft index that this event is recorded in [if applicable]
func (e *EventInfo) RaftIndex() uint64 {
	return e.raftIndex
}

// SetRaftIndex implements the RaftIndexer interface, which allows the Collector
// to defer supplying the raft index til later on
func (e *EventInfo) SetRaftIndex(i uint64) bool {
	if e.raftIndex == 0 || e.raftIndex == i {
		e.raftIndex = i
		return true
	}
	return false
}

// Message returns any additional per eventType information to be captured
func (e *EventInfo) Message() string {
	return e.message
}
