package metrics

import (
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/go-phorce/dolly/xlog"
)

var logger = xlog.NewPackageLogger("github.com/go-phorce/dolly", "metrics")

var prov Metrics

func init() {
	prov = new(nilmetrics)
}

// The MetricSink interface is used to transmit metrics information
// to an external system
type MetricSink metrics.MetricSink

// Config is used to configure metrics settings
type Config metrics.Config

// Metrics basics
type Metrics interface {
	SetGauge(key []string, val float32)
	IncrCounter(key []string, val float32)
	AddSample(key []string, val float32)
	MeasureSince(key []string, start time.Time)
}

// SetProvider for metrics
func SetProvider(p Metrics) {
	prov = p
}

// New is used to create a new instance of Metrics
func New(conf *Config, sink MetricSink) (Metrics, error) {
	m, err := metrics.New(
		(*metrics.Config)(conf),
		sink.(metrics.MetricSink))

	return Metrics(m), err
}

// DefaultConfig provides a sane default configuration
func DefaultConfig(serviceName string) *Config {
	return (*Config)(metrics.DefaultConfig(serviceName))
}

//
// Standard go-metrics
//
type stdmetrics struct{}

// NewStandardProvider returns standard provider
func NewStandardProvider() Metrics {
	return new(stdmetrics)
}

// SetGauge wraps SetGauge from armon/go-metrics
func (*stdmetrics) SetGauge(key []string, val float32) {
	metrics.SetGauge(key, val)
}

// IncrCounter wraps IncrCounter from armon/go-metrics
func (*stdmetrics) IncrCounter(key []string, val float32) {
	metrics.IncrCounter(key, val)
}

// AddSample wraps AddSample from armon/go-metrics
func (*stdmetrics) AddSample(key []string, val float32) {
	metrics.AddSample(key, val)
}

// MeasureSince wraps MeasureSince from armon/go-metrics
func (*stdmetrics) MeasureSince(key []string, start time.Time) {
	metrics.MeasureSince(key, start)
}

//
// nil metrics
//
type nilmetrics struct{}

// SetGauge wraps SetGauge from armon/go-metrics
func (*nilmetrics) SetGauge(key []string, val float32) {
}

// IncrCounter wraps IncrCounter from armon/go-metrics
func (*nilmetrics) IncrCounter(key []string, val float32) {
}

// AddSample wraps AddSample from armon/go-metrics
func (*nilmetrics) AddSample(key []string, val float32) {
}

// MeasureSince wraps MeasureSince from armon/go-metrics
func (*nilmetrics) MeasureSince(key []string, start time.Time) {
}

//
// Current provider
//

// SetGauge wraps SetGauge from armon/go-metrics
func SetGauge(key []string, val float32) {
	prov.SetGauge(key, val)
}

// IncrCounter wraps IncrCounter from armon/go-metrics
func IncrCounter(key []string, val float32) {
	prov.IncrCounter(key, val)
}

// AddSample wraps AddSample from armon/go-metrics
func AddSample(key []string, val float32) {
	prov.AddSample(key, val)
}

// MeasureSince wraps MeasureSince from armon/go-metrics
func MeasureSince(key []string, start time.Time) {
	prov.MeasureSince(key, start)
}
