package metrics_test

import (
	"syscall"
	"testing"
	"time"

	"github.com/go-phorce/dolly/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewInmemSink(t *testing.T) {
	im := metrics.NewInmemSink(100*time.Millisecond, time.Second)
	prov, err := metrics.New(&metrics.Config{
		FilterDefault:        true,
		HostName:             "test",
		ServiceName:          "test",
		EnableHostname:       true,
		EnableHostnameLabel:  true,
		EnableServiceLabel:   true,
		EnableRuntimeMetrics: true,
		EnableTypePrefix:     true,
	}, im)
	require.NoError(t, err)
	run(prov, 10)
	time.Sleep(200 * time.Millisecond)
	run(prov, 10)
	time.Sleep(200 * time.Millisecond)
	run(prov, 10)
	time.Sleep(200 * time.Millisecond)

	data := im.Data()
	require.True(t, len(data) >= 3)

	d := data[0]
	for k, v := range d.Counters {
		assert.Equal(t, "counter.test.metrics.counter;host=test;service=test", k)
		s := v.String()
		assert.Contains(t, s, "Count:")
	}
	for k, v := range d.Samples {
		assert.Contains(t, k, "host=test;service=test")
		s := v.String()
		assert.NotEmpty(t, s)
		assert.Contains(t, s, "Count:")
	}

	samples, err := im.DisplayMetrics()
	require.NoError(t, err)
	assert.NotNil(t, samples)
}

func Test_InmemSink_Signal_Stop(t *testing.T) {
	im := metrics.NewInmemSink(100*time.Millisecond, time.Second)
	prov, err := metrics.New(&metrics.Config{
		FilterDefault: true,
	}, im)
	require.NoError(t, err)

	s := metrics.DefaultInmemSignal(im)
	run(prov, 10)

	s.Stop()

	data := im.Data()
	require.NotEmpty(t, data)
}

func Test_InmemSink_Signal_Signal(t *testing.T) {
	im := metrics.NewInmemSink(100*time.Millisecond, time.Second)
	prov, err := metrics.New(&metrics.Config{
		FilterDefault: true,
	}, im)
	require.NoError(t, err)

	s := metrics.DefaultInmemSignal(im)
	run(prov, 10)

	time.Sleep(time.Second)
	// send this process signal
	syscall.Kill(syscall.Getpid(), metrics.DefaultSignal)
	time.Sleep(time.Second)

	s.Stop()

	data := im.Data()
	require.NotEmpty(t, data)
}
