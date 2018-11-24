package metrics

import (
	"time"
)

var (
	keyForHeartbeat = []string{"heartbeat"}
	keyForUptime    = []string{"uptime", "seconds"}
)

// PublishHeartbeat publishes heartbeat of the service
func PublishHeartbeat(service string) {
	IncrCounter(keyForHeartbeat, 1, Tag{"service", service})
}

// PublishUptime publishes uptime of the service
func PublishUptime(service string, uptime time.Duration) {
	SetGauge(keyForUptime, float32(uptime/time.Second), Tag{"service", service})
}
