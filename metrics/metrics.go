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

// Tag is used to add dimentions to metrics
type Tag struct {
	Name  string
	Value string
}

// Metrics basics
type Metrics interface {
	SetGauge(key []string, val float32, tags ...Tag)
	IncrCounter(key []string, val float32, tags ...Tag)
	AddSample(key []string, val float32, tags ...Tag)
	MeasureSince(key []string, start time.Time, tags ...Tag)
}

func metricsLabels(tags []Tag) []metrics.Label {
	if len(tags) == 0 {
		return nil
	}

	labels := make([]metrics.Label, len(tags))
	for i, tag := range tags {
		labels[i].Name = tag.Name
		labels[i].Value = tag.Value
	}
	return labels
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

	return &stdmetrics{m}, err
}

// DefaultConfig provides a sane default configuration
func DefaultConfig(serviceName string) *Config {
	return (*Config)(metrics.DefaultConfig(serviceName))
}

//
// Standard go-metrics
//
type stdmetrics struct {
	proxy *metrics.Metrics
}

// NewStandardProvider returns standard provider
func NewStandardProvider() Metrics {
	return new(stdmetrics)
}

// SetGauge wraps SetGauge from armon/go-metrics
func (std *stdmetrics) SetGauge(key []string, val float32, tags ...Tag) {
	labels := metricsLabels(tags)
	if std.proxy != nil {
		std.proxy.SetGaugeWithLabels(key, val, labels)
	} else {
		metrics.SetGaugeWithLabels(key, val, labels)
	}
}

// IncrCounter wraps IncrCounter from armon/go-metrics
func (std *stdmetrics) IncrCounter(key []string, val float32, tags ...Tag) {
	labels := metricsLabels(tags)
	if std.proxy != nil {
		std.proxy.IncrCounterWithLabels(key, val, labels)
	} else {
		metrics.IncrCounterWithLabels(key, val, labels)
	}
}

// AddSample wraps AddSample from armon/go-metrics
func (std *stdmetrics) AddSample(key []string, val float32, tags ...Tag) {
	labels := metricsLabels(tags)
	if std.proxy != nil {
		std.proxy.AddSampleWithLabels(key, val, labels)
	} else {
		metrics.AddSampleWithLabels(key, val, labels)
	}
}

// MeasureSince wraps MeasureSince from armon/go-metrics
func (std *stdmetrics) MeasureSince(key []string, start time.Time, tags ...Tag) {
	labels := metricsLabels(tags)
	if std.proxy != nil {
		std.proxy.MeasureSinceWithLabels(key, start, labels)
	} else {
		metrics.MeasureSinceWithLabels(key, start, labels)
	}
}

//
// nil metrics
//
type nilmetrics struct{}

// SetGauge wraps SetGauge from armon/go-metrics
func (*nilmetrics) SetGauge(key []string, val float32, tags ...Tag) {
}

// IncrCounter wraps IncrCounter from armon/go-metrics
func (*nilmetrics) IncrCounter(key []string, val float32, tags ...Tag) {
}

// AddSample wraps AddSample from armon/go-metrics
func (*nilmetrics) AddSample(key []string, val float32, tags ...Tag) {
}

// MeasureSince wraps MeasureSince from armon/go-metrics
func (*nilmetrics) MeasureSince(key []string, start time.Time, tags ...Tag) {
}

//
// Current provider
//

// SetGauge wraps SetGauge from armon/go-metrics
func SetGauge(key []string, val float32, tags ...Tag) {
	prov.SetGauge(key, val, tags...)
}

// IncrCounter wraps IncrCounter from armon/go-metrics
func IncrCounter(key []string, val float32, tags ...Tag) {
	prov.IncrCounter(key, val, tags...)
}

// AddSample wraps AddSample from armon/go-metrics
func AddSample(key []string, val float32, tags ...Tag) {
	prov.AddSample(key, val, tags...)
}

// MeasureSince wraps MeasureSince from armon/go-metrics
func MeasureSince(key []string, start time.Time, tags ...Tag) {
	prov.MeasureSince(key, start, tags...)
}
