package util

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-phorce/dolly/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PublishUptime(t *testing.T) {
	im := metrics.NewInmemSink(time.Minute, time.Minute*5)
	_, err := metrics.NewGlobal(metrics.DefaultConfig("svc1"), im)
	require.NoError(t, err)

	PublishHeartbeat("svc1")
	PublishUptime("svc1", time.Second)

	// get samples in memory
	data := im.Data()
	require.NotEqual(t, 0, len(data))

	for k := range data[0].Gauges {
		t.Log("Gauge:", k)
	}
	for k := range data[0].Counters {
		t.Log("Counter:", k)
	}

	assertGauge := func(key string) {
		s, exists := data[0].Gauges[key]
		require.True(t, exists, "Expected metric with key %s to exist, but it doesn't", key)
		assert.True(t, s.Value > 0 && s.Value <= 1, "Unexpected value for metric %s", key)
	}
	assertCounter := func(key string, expectedCount int) {
		s, exists := data[0].Counters[key]
		require.True(t, exists, "Expected metric with key %s to exist, but it doesn't", key)
		assert.Equal(t, expectedCount, s.Count, "Unexpected count for metric %s", key)
	}
	hostname, _ := os.Hostname()
	assertGauge(fmt.Sprintf("svc1.%s.uptime.seconds;service=svc1", hostname))
	assertCounter(fmt.Sprintf("svc1.heartbeat;service=svc1"), 1)
}
