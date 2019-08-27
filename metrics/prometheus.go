package metrics

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// DefaultPrometheusOpts is the default set of options used when creating a
	// PrometheusSink.
	DefaultPrometheusOpts = PrometheusOpts{
		Expiration: 60 * time.Second,
	}
)

// PrometheusOpts is used to configure the Prometheus Sink
type PrometheusOpts struct {
	// Expiration is the duration a metric is valid for, after which it will be
	// untracked. If the value is zero, a metric is never expired.
	Expiration time.Duration
}

// PrometheusSink provides a MetricSink that can be used
// with a prometheus server.
type PrometheusSink struct {
	mu         sync.Mutex
	gauges     map[string]prometheus.Gauge
	summaries  map[string]prometheus.Summary
	counters   map[string]prometheus.Counter
	updates    map[string]time.Time
	expiration time.Duration
}

// NewPrometheusSink creates a new PrometheusSink using the default options.
func NewPrometheusSink() (*PrometheusSink, error) {
	return NewPrometheusSinkFrom(DefaultPrometheusOpts)
}

// NewPrometheusSinkFrom creates a new PrometheusSink using the passed options.
func NewPrometheusSinkFrom(opts PrometheusOpts) (*PrometheusSink, error) {
	sink := &PrometheusSink{
		gauges:     make(map[string]prometheus.Gauge),
		summaries:  make(map[string]prometheus.Summary),
		counters:   make(map[string]prometheus.Counter),
		updates:    make(map[string]time.Time),
		expiration: opts.Expiration,
	}

	return sink, prometheus.Register(sink)
}

// Describe is needed to meet the Collector interface.
func (p *PrometheusSink) Describe(c chan<- *prometheus.Desc) {
	// We must emit some description otherwise an error is returned. This
	// description isn't shown to the user!
	prometheus.NewGauge(prometheus.GaugeOpts{Name: "Dummy", Help: "Dummy"}).Describe(c)
}

// Collect meets the collection interface and allows us to enforce our expiration
// logic to clean up ephemeral metrics if their value haven't been set for a
// duration exceeding our allowed expiration time.
func (p *PrometheusSink) Collect(c chan<- prometheus.Metric) {
	p.mu.Lock()
	defer p.mu.Unlock()

	expire := p.expiration != 0
	now := time.Now()
	for k, v := range p.gauges {
		last := p.updates[k]
		if expire && last.Add(p.expiration).Before(now) {
			delete(p.updates, k)
			delete(p.gauges, k)
		} else {
			v.Collect(c)
		}
	}
	for k, v := range p.summaries {
		last := p.updates[k]
		if expire && last.Add(p.expiration).Before(now) {
			delete(p.updates, k)
			delete(p.summaries, k)
		} else {
			v.Collect(c)
		}
	}
	for k, v := range p.counters {
		last := p.updates[k]
		if expire && last.Add(p.expiration).Before(now) {
			delete(p.updates, k)
			delete(p.counters, k)
		} else {
			v.Collect(c)
		}
	}
}

var forbiddenChars = regexp.MustCompile("[ .=\\-/]")

func (p *PrometheusSink) flattenKey(parts []string, labels []Tag) (string, string) {
	key := strings.Join(parts, "_")
	key = forbiddenChars.ReplaceAllString(key, "_")

	hash := key
	for _, label := range labels {
		hash += fmt.Sprintf(";%s=%s", label.Name, label.Value)
	}

	return key, hash
}

func prometheusLabels(labels []Tag) prometheus.Labels {
	l := make(prometheus.Labels)
	for _, label := range labels {
		l[label.Name] = label.Value
	}
	return l
}

// SetGauge should retain the last value it is set to
func (p *PrometheusSink) SetGauge(parts []string, val float32, tags []Tag) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key, hash := p.flattenKey(parts, tags)
	g, ok := p.gauges[hash]
	if !ok {
		g = prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        key,
			Help:        key,
			ConstLabels: prometheusLabels(tags),
		})
		p.gauges[hash] = g
	}
	g.Set(float64(val))
	p.updates[hash] = time.Now()
}

// AddSample is for timing information, where quantiles are used
func (p *PrometheusSink) AddSample(parts []string, val float32, tags []Tag) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key, hash := p.flattenKey(parts, tags)
	g, ok := p.summaries[hash]
	if !ok {
		g = prometheus.NewSummary(prometheus.SummaryOpts{
			Name:        key,
			Help:        key,
			MaxAge:      10 * time.Second,
			ConstLabels: prometheusLabels(tags),
		})
		p.summaries[hash] = g
	}
	g.Observe(float64(val))
	p.updates[hash] = time.Now()
}

// EmitKey is not implemented. Prometheus doesn’t offer a type for which an
// arbitrary number of values is retained, as Prometheus works with a pull
// model, rather than a push model.
func (p *PrometheusSink) EmitKey(key []string, val float32) {
}

// IncrCounter should accumulate values
func (p *PrometheusSink) IncrCounter(parts []string, val float32, tags []Tag) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key, hash := p.flattenKey(parts, tags)
	g, ok := p.counters[hash]
	if !ok {
		g = prometheus.NewCounter(prometheus.CounterOpts{
			Name:        key,
			Help:        key,
			ConstLabels: prometheusLabels(tags),
		})
		p.counters[hash] = g
	}
	g.Add(float64(val))
	p.updates[hash] = time.Now()
}
