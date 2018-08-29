package audittest

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ekspand/pkg/algorithms/slices"
	"github.com/ekspand/pkg/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CompareTo will compare the captured sequence of audit event to the supplied csv file
// valid columns that can be used to compare in the CSV match the Event interface function names
// if assertAllMatched is true, then this will log a failure if there are audit events left
// captured after matching all the ones in the csv file.
func (a *Auditor) CompareTo(t *testing.T, csvFile string, assertAllMatched bool) {
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

func compareOne(t *testing.T, rowIdx int, columnNames []string, expectedValues []string, actual audit.Event) {
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
	var ae *audit.Event
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
