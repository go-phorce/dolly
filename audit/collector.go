package audit

// Collector is an Auditor that collects audit events in memory and sends them to the supplied destination
// Auditor when Submit is called. This can be useful for collecting up a set of audit events that aren't true
// until some later condition is verified [e.g. an Update to DB was successfull]
type Collector struct {
	Destination Auditor
	events      []Event
}

// Event records a new Auditable event, its kept in memory until Submit() is called at which
// point it is sent to the Destination auditor
func (c *Collector) Event(e Event) {
	if c.events == nil {
		c.events = make([]Event, 0, 16)
	}
	c.events = append(c.events, e)
}

// Submit will flush all collected events to date to the Destination auditor. if raftIndex > 0
// the submitted events will reflect that raftIndex, otherwise the original raftIndex is preserved.
func (c *Collector) Submit(raftIndex uint64) {
	for _, e := range c.events {
		c.Destination.Event(withRaftIndex(e, raftIndex))
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

func withRaftIndex(e Event, raftIndex uint64) Event {
	if raftIndex == 0 {
		return e
	}
	if me, ok := e.(RaftIndexer); ok {
		if me.SetRaftIndex(raftIndex) {
			return e
		}
	}
	return &EventInfo{
		identity:  e.Identity(),
		contextID: e.ContextID(),
		src:       e.Source(),
		eventType: e.EventType(),
		raftIndex: raftIndex,
		message:   e.Message(),
	}
}
