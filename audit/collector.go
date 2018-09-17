package audit

// Collector is an Auditor that collects audit events in memory and sends them to the supplied destination
// Auditor when Submit is called. This can be useful for collecting up a set of audit events that aren't true
// until some later condition is verified [e.g. an Update to DB was successfull]
type Collector struct {
	Destination Auditor
	events      []*eventInfo
}

// Audit records a new Auditable event, its kept in memory until Submit() is called at which
// point it is sent to the Destination auditor
func (c *Collector) Audit(source string,
	eventType string,
	identity string,
	contextID string,
	raftIndex uint64,
	message string) {
	if c.events == nil {
		c.events = make([]*eventInfo, 0, 16)
	}
	e := &eventInfo{
		identity:  identity,
		contextID: contextID,
		source:    source,
		eventType: eventType,
		raftIndex: raftIndex,
		message:   message,
	}

	c.events = append(c.events, e)
}

// Submit will flush all collected events to date to the Destination auditor. if raftIndex > 0
// the submitted events will reflect that raftIndex, otherwise the original raftIndex is preserved.
func (c *Collector) Submit(raftIndex uint64) {
	for _, e := range c.events {
		re := withRaftIndex(e, raftIndex)
		c.Destination.Audit(re.source, re.eventType, re.identity, re.contextID, re.raftIndex, re.message)
	}
	c.events = nil
}

// Close will remove any collected events from this collector.
func (c *Collector) Close() error {
	c.events = nil
	return nil
}

// RaftIndexer identifies Event implementations that allow the raft index
// to be set
type RaftIndexer interface {
	// SetRaftIndex allows the raft index to be potentially updated
	// it returns true if the index was succesfully updated, false if
	// not. [typically impls should only allow it be set if it wasn't
	// previously set]
	SetRaftIndex(i uint64) bool
}

func withRaftIndex(e *eventInfo, raftIndex uint64) *eventInfo {
	if raftIndex == 0 {
		return e
	}
	if e.SetRaftIndex(raftIndex) {
		return e
	}
	return &eventInfo{
		identity:  e.identity,
		contextID: e.contextID,
		source:    e.source,
		eventType: e.eventType,
		raftIndex: raftIndex,
		message:   e.message,
	}
}

// eventInfo provides a default impl of Event
type eventInfo struct {
	identity  string
	contextID string
	source    string
	eventType string
	raftIndex uint64
	message   string
}

// SetRaftIndex implements the RaftIndexer interface, which allows the Collector
// to defer supplying the raft index til later on
func (e *eventInfo) SetRaftIndex(i uint64) bool {
	if e.raftIndex == 0 || e.raftIndex == i {
		e.raftIndex = i
		return true
	}
	return false
}
