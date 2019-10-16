// Package audittest provides helper methods for writing tests that interact
// with the audit logging service.
package audittest

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/go-phorce/dolly/algorithms/slices"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Failf(t, "Unable to find an audit event of type %q in captured items", eventType)
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

// CompareTo will compare the captured sequence of audit event to the supplied csv file
// valid columns that can be used to compare in the CSV match the Event interface function names
// if assertAllMatched is true, then this will log a failure if there are audit events left
// captured after matching all the ones in the csv file.
func (a *auditor) CompareTo(t *testing.T, csvFile string, assertAllMatched bool) {
	f, err := os.Open(csvFile)
	require.NoError(t, err)
	defer f.Close()
	r := csv.NewReader(bufio.NewReader(f))
	r.TrimLeadingSpace = true
	headers, err := r.Read()
	require.NoError(t, err, "Unable read header row from %s", csvFile)
	verifyHeaderNames(t, headers)
	// get a cloned copy of the audit events to work on
	aEvents := a.GetAll()
	idx := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		require.True(t, idx < len(aEvents), "Expected audit event %d not recorded", idx)
		actual := aEvents[idx]
		compareOne(t, idx+1, headers, row, actual)
		idx++
	}
	if assertAllMatched {
		require.True(t, idx >= len(aEvents), "There are %d unexpected audit events recorded", len(aEvents)-idx)
	}
}

func compareOne(t *testing.T, rowIdx int, columnNames []string, expectedValues []string, actual *event) {
	a := reflect.ValueOf(actual)
	for idx := range columnNames {
		m := a.MethodByName(columnNames[idx])
		require.True(t, m.IsValid(), "Unable to find method %s on event %s, please fix column names [valid names are %s]", columnNames[idx], actual, strings.Join(eventMethodNames(), ", "))
		fActual := m.Call(nil)[0].Interface()
		strActual, isString := fActual.(string)
		if !isString {
			strActual = fmt.Sprintf("%v", fActual)
		}
		assert.Equal(t, expectedValues[idx], strActual, "Value for %s on row %d differs", columnNames[idx], rowIdx)
	}
}

func eventMethodNames() []string {
	var ae *event
	t := reflect.TypeOf(ae).Elem()
	res := make([]string, t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		res[i] = t.Method(i).Name
	}
	return res
}

func verifyHeaderNames(t *testing.T, headers []string) {
	avail := eventMethodNames()
	for _, h := range headers {
		if !slices.ContainsString(avail, h) {
			t.Errorf("Column named %s isn't valid, valid names are %s", h, strings.Join(avail, ", "))
		}
	}
	if t.Failed() {
		t.FailNow()
	}
}
