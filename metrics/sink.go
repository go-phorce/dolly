package metrics

import (
	"net/url"
	"time"

	"github.com/pkg/errors"
)

// Tag is used to add dimentions to metrics
type Tag struct {
	Name  string
	Value string
}

// The Sink interface is used to transmit metrics information
// to an external system
type Sink interface {
	// SetGauge should retain the last value it is set to
	SetGauge(key []string, val float32, tags []Tag)
	// IncrCounter should accumulate values
	IncrCounter(key []string, val float32, tags []Tag)
	// AddSample is for timing information, where quantiles are used
	AddSample(key []string, val float32, tags []Tag)
}

// Provider basics
type Provider interface {
	SetGauge(key []string, val float32, tags ...Tag)
	IncrCounter(key []string, val float32, tags ...Tag)
	AddSample(key []string, val float32, tags ...Tag)
	MeasureSince(key []string, start time.Time, tags ...Tag)
}

// BlackholeSink is used to just blackhole messages
type BlackholeSink struct{}

// SetGauge should retain the last value it is set to
func (*BlackholeSink) SetGauge(key []string, val float32, tags []Tag) {}

// IncrCounter should accumulate values
func (*BlackholeSink) IncrCounter(key []string, val float32, tags []Tag) {}

// AddSample is for timing information, where quantiles are used
func (*BlackholeSink) AddSample(key []string, val float32, tags []Tag) {}

// FanoutSink is used to sink to fanout values to multiple sinks
type FanoutSink []Sink

// NewFanoutSink creates fan-out sink
func NewFanoutSink(sinks ...Sink) FanoutSink {
	return FanoutSink(sinks)
}

// SetGauge should retain the last value it is set to
func (fh FanoutSink) SetGauge(key []string, val float32, tags []Tag) {
	for _, s := range fh {
		s.SetGauge(key, val, tags)
	}
}

// IncrCounter should accumulate values
func (fh FanoutSink) IncrCounter(key []string, val float32, tags []Tag) {
	for _, s := range fh {
		s.IncrCounter(key, val, tags)
	}
}

// AddSample is for timing information, where quantiles are used
func (fh FanoutSink) AddSample(key []string, val float32, tags []Tag) {
	for _, s := range fh {
		s.AddSample(key, val, tags)
	}
}

// sinkURLFactoryFunc is an generic interface around the *SinkFromURL() function provided
// by each sink type
type sinkURLFactoryFunc func(*url.URL) (Sink, error)

// sinkRegistry supports the generic NewMetricSink function by mapping URL
// schemes to metric sink factory functions
var sinkRegistry = map[string]sinkURLFactoryFunc{
	"statsd": NewStatsdSinkFromURL,
	"inmem":  NewInmemSinkFromURL,
}

// NewMetricSinkFromURL allows a generic URL input to configure any of the
// supported sinks. The scheme of the URL identifies the type of the sink, the
// and query parameters are used to set options.
//
// "statsd://" - Initializes a StatsdSink. The host and port are passed through
// as the "addr" of the sink
//
// "statsite://" - Initializes a StatsiteSink. The host and port become the
// "addr" of the sink
//
// "inmem://" - Initializes an InmemSink. The host and port are ignored. The
// "interval" and "retain" query parameters must be specified with valid
// durations, see NewInmemSink for details.
func NewMetricSinkFromURL(urlStr string) (Sink, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	sinkURLFactoryFunc := sinkRegistry[u.Scheme]
	if sinkURLFactoryFunc == nil {
		return nil, errors.Errorf("cannot create metric sink, unrecognized sink name: %q", u.Scheme)
	}

	return sinkURLFactoryFunc(u)
}
