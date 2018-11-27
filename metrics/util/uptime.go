package util

import (
	"time"

	"github.com/go-phorce/dolly/metrics"
)

var (
	keyForHeartbeat = []string{"heartbeat"}
	keyForUptime    = []string{"uptime", "seconds"}
)

// PublishHeartbeat publishes heartbeat of the service
func PublishHeartbeat(service string) {
	metrics.IncrCounter(keyForHeartbeat, 1, metrics.Tag{Name: "service", Value: service})
}

// PublishUptime publishes uptime of the service
func PublishUptime(service string, uptime time.Duration) {
	metrics.SetGauge(keyForUptime, float32(uptime/time.Second), metrics.Tag{Name: "service", Value: service})
}
