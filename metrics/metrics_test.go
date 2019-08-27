package metrics_test

import (
	"testing"
	"time"

	"github.com/go-phorce/dolly/metrics"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func run(p metrics.Provider, times int) {
	for i := 0; i < times; i++ {
		p.SetGauge([]string{"test", "metrics", "gauge"}, float32(i))
		p.IncrCounter([]string{"test", "metrics", "counter"}, float32(i))
		p.AddSample([]string{"test", "metrics", "sample"}, float32(i))
		p.MeasureSince([]string{"test", "metrics", "since"}, time.Now().Add(time.Duration(i)*time.Second))
	}
}

func Test_SetProvider(t *testing.T) {
	im := metrics.NewInmemSink(time.Second, time.Minute)
	prov, err := metrics.New(&metrics.Config{
		FilterDefault: true,
	}, im)
	require.NoError(t, err)
	run(prov, 10)
}

func Test_SetProviderDatadog(t *testing.T) {
	d, err := metrics.NewDogStatsdSink("127.0.0.1:8125", "dolly")
	require.NoError(t, err)

	prov, err := metrics.New(&metrics.Config{
		FilterDefault: true,
	}, d)
	require.NoError(t, err)
	run(prov, 10)
}

func Test_SetProviderPrometheus(t *testing.T) {
	d, err := metrics.NewPrometheusSink()
	require.NoError(t, err)

	prov, err := metrics.New(&metrics.Config{
		FilterDefault: true,
	}, d)
	require.NoError(t, err)
	run(prov, 10)
}

//
// Mock
//
type mockedSink struct {
	t *testing.T
	mock.Mock
}

func (m *mockedSink) SetGauge(key []string, val float32, labels []metrics.Tag) {
	m.t.Logf("SetGauge key=%v", key)
	m.Called(key, val, labels)
}

func (m *mockedSink) IncrCounter(key []string, val float32, labels []metrics.Tag) {
	m.t.Logf("IncrCounter key=%v", key)
	m.Called(key, val, labels)
}

func (m *mockedSink) AddSample(key []string, val float32, labels []metrics.Tag) {
	m.t.Logf("AddSample key=%v", key)
	m.Called(key, val, labels)
}

func Test_Emit(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		mocked := &mockedSink{t: t}

		// setup expectations
		mocked.AssertNotCalled(t, "SetGauge", mock.Anything, mock.Anything, mock.Anything)
		mocked.AssertNotCalled(t, "IncrCounter", mock.Anything, mock.Anything, mock.Anything)
		mocked.AssertNotCalled(t, "AddSample", mock.Anything, mock.Anything, mock.Anything)

		prov, err := metrics.New(&metrics.Config{}, mocked)
		require.NoError(t, err)

		run(prov, 1)

		// assert that the expectations were met
		mocked.AssertExpectations(t)
	})

	t.Run("enabled config", func(t *testing.T) {
		mocked := &mockedSink{t: t}

		// setup expectations
		mocked.On("SetGauge", mock.Anything, mock.Anything, mock.Anything).Times(0)
		mocked.On("IncrCounter", mock.Anything, mock.Anything, mock.Anything).Times(1)
		mocked.On("AddSample", mock.Anything, mock.Anything, mock.Anything).Times(2)

		prov, err := metrics.New(&metrics.Config{
			ServiceName:    "dolly",
			EnableHostname: true,
			FilterDefault:  true,
		}, mocked)
		require.NoError(t, err)

		run(prov, 1)

		// assert that the expectations were met
		mocked.AssertExpectations(t)
	})
}

func Test_FanoutSink(t *testing.T) {
	mocked := &mockedSink{t: t}
	fan := metrics.NewFanoutSink(mocked, mocked, metrics.NewInmemSink(time.Minute, time.Minute*5))

	// setup expectations
	mocked.On("SetGauge", mock.Anything, mock.Anything, mock.Anything).Times(0)
	mocked.On("IncrCounter", mock.Anything, mock.Anything, mock.Anything).Times(4)
	mocked.On("AddSample", mock.Anything, mock.Anything, mock.Anything).Times(6)

	prov, err := metrics.New(
		&metrics.Config{
			ServiceName:    "dolly",
			EnableHostname: true,
			FilterDefault:  true,
		},
		fan)
	require.NoError(t, err)

	run(prov, 1)

	fan.SetGauge([]string{"test", "metrics", "gauge"}, float32(0), nil)
	fan.IncrCounter([]string{"test", "metrics", "counter"}, float32(0), nil)
	fan.AddSample([]string{"test", "metrics", "sample"}, float32(0), nil)

	// assert that the expectations were met
	mocked.AssertExpectations(t)
}
