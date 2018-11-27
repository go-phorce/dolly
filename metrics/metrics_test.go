package metrics_test

import (
	"testing"
	"time"

	gometrics "github.com/armon/go-metrics"
	"github.com/armon/go-metrics/datadog"
	"github.com/go-phorce/dolly/metrics"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func run(times int) {
	for i := 0; i < times; i++ {
		metrics.SetGauge([]string{"test", "metrics", "gauge"}, float32(i))
		metrics.IncrCounter([]string{"test", "metrics", "counter"}, float32(i))
		metrics.AddSample([]string{"test", "metrics", "sample"}, float32(i))
		metrics.MeasureSince([]string{"test", "metrics", "since"}, time.Now().Add(time.Duration(i)*time.Second))
	}
}

func Test_DefaultProv(t *testing.T) {
	run(10)
}

func Test_SetProvider(t *testing.T) {
	im := gometrics.NewInmemSink(time.Second, time.Minute)
	prov, err := metrics.New(&metrics.Config{}, im)
	require.NoError(t, err)
	metrics.SetProvider(prov)
	run(10)
}

func Test_SetProviderDatadog(t *testing.T) {
	d, err := datadog.NewDogStatsdSink("127.0.0.1:8125", "dolly")
	require.NoError(t, err)

	prov, err := metrics.New(&metrics.Config{}, d)
	require.NoError(t, err)
	metrics.SetProvider(prov)
	run(10)
}

//
// Mock
//
type mockedSink struct {
	t *testing.T
	mock.Mock
}

func (m *mockedSink) SetGauge(key []string, val float32) {
	m.t.Logf("SetGauge key=%v", key)
	m.Called(key, val)
}

func (m *mockedSink) SetGaugeWithLabels(key []string, val float32, labels []gometrics.Label) {
	m.t.Logf("SetGaugeWithLabels key=%v", key)
	m.Called(key, val, labels)
}

func (m *mockedSink) EmitKey(key []string, val float32) {
	m.t.Logf("EmitKey key=%v", key)
	m.Called(key, val)
}

func (m *mockedSink) IncrCounter(key []string, val float32) {
	m.t.Logf("IncrCounter key=%v", key)
	m.Called(key, val)
}

func (m *mockedSink) IncrCounterWithLabels(key []string, val float32, labels []gometrics.Label) {
	m.t.Logf("IncrCounterWithLabels key=%v", key)
	m.Called(key, val, labels)
}

func (m *mockedSink) AddSample(key []string, val float32) {
	m.t.Logf("AddSample key=%v", key)
	m.Called(key, val)
}

func (m *mockedSink) AddSampleWithLabels(key []string, val float32, labels []gometrics.Label) {
	m.t.Logf("AddSampleWithLabels key=%v", key)
	m.Called(key, val, labels)
}

func Test_Emit(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		mocked := &mockedSink{t: t}

		// setup expectations
		mocked.AssertNotCalled(t, "SetGauge", mock.Anything, mock.Anything)
		mocked.AssertNotCalled(t, "SetGaugeWithLabels", mock.Anything, mock.Anything, mock.Anything)
		mocked.AssertNotCalled(t, "EmitKey", mock.Anything, mock.Anything)
		mocked.AssertNotCalled(t, "IncrCounter", mock.Anything, mock.Anything)
		mocked.AssertNotCalled(t, "IncrCounterWithLabels", mock.Anything, mock.Anything, mock.Anything)
		mocked.AssertNotCalled(t, "AddSample", mock.Anything, mock.Anything)
		mocked.AssertNotCalled(t, "AddSampleWithLabels", mock.Anything, mock.Anything, mock.Anything)

		prov, err := metrics.New(&metrics.Config{}, mocked)
		require.NoError(t, err)
		metrics.SetProvider(prov)

		run(1)

		// assert that the expectations were met
		mocked.AssertExpectations(t)
	})

	t.Run("enabled config", func(t *testing.T) {
		mocked := &mockedSink{t: t}

		// setup expectations
		mocked.AssertNotCalled(t, "SetGauge", mock.Anything, mock.Anything)
		mocked.On("SetGaugeWithLabels", mock.Anything, mock.Anything, mock.Anything).Times(0)
		mocked.AssertNotCalled(t, "EmitKey", mock.Anything, mock.Anything)
		mocked.AssertNotCalled(t, "IncrCounter", mock.Anything, mock.Anything)
		mocked.On("IncrCounterWithLabels", mock.Anything, mock.Anything, mock.Anything).Times(1)
		mocked.AssertNotCalled(t, "AddSample", mock.Anything, mock.Anything)
		mocked.On("AddSampleWithLabels", mock.Anything, mock.Anything, mock.Anything).Times(2)

		prov, err := metrics.New(&metrics.Config{
			ServiceName:    "dolly",
			EnableHostname: true,
			FilterDefault:  true,
		}, mocked)
		require.NoError(t, err)
		metrics.SetProvider(prov)

		run(1)

		// assert that the expectations were met
		mocked.AssertExpectations(t)
	})
}

func Test_FanoutSink(t *testing.T) {
	mocked := &mockedSink{t: t}
	fan := metrics.NewFanoutSink(mocked, mocked)

	// setup expectations
	mocked.AssertNotCalled(t, "SetGauge", mock.Anything, mock.Anything)
	mocked.On("SetGaugeWithLabels", mock.Anything, mock.Anything, mock.Anything).Times(0)
	mocked.On("EmitKey", mock.Anything, mock.Anything).Times(2)
	mocked.AssertNotCalled(t, "IncrCounter", mock.Anything, mock.Anything)
	mocked.On("IncrCounterWithLabels", mock.Anything, mock.Anything, mock.Anything).Times(6)
	mocked.AssertNotCalled(t, "AddSample", mock.Anything, mock.Anything)
	mocked.On("AddSampleWithLabels", mock.Anything, mock.Anything, mock.Anything).Times(8)

	prov, err := metrics.New(
		&metrics.Config{
			ServiceName:    "dolly",
			EnableHostname: true,
			FilterDefault:  true,
		},
		fan)
	require.NoError(t, err)
	metrics.SetProvider(prov)

	run(1)

	fan.SetGauge([]string{"test", "metrics", "gauge"}, float32(0))
	fan.EmitKey([]string{"test", "metrics", "gauge"}, float32(0))
	fan.IncrCounter([]string{"test", "metrics", "counter"}, float32(0))
	fan.AddSample([]string{"test", "metrics", "sample"}, float32(0))
	fan.SetGaugeWithLabels([]string{"test", "metrics", "gauge"}, float32(0), nil)
	fan.IncrCounterWithLabels([]string{"test", "metrics", "counter"}, float32(0), nil)
	fan.AddSampleWithLabels([]string{"test", "metrics", "sample"}, float32(0), nil)

	// assert that the expectations were met
	mocked.AssertExpectations(t)
}
