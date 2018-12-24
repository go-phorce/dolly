package metrics_test

import (
	"testing"

	"github.com/go-phorce/dolly/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewMetricSinkFromURL(t *testing.T) {
	s, err := metrics.NewMetricSinkFromURL("http://localhost")
	require.Error(t, err)
	assert.Nil(t, s)
}
