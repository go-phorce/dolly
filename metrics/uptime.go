package metrics

import (
	"time"

	"github.com/go-phorce/dolly/metrics/tags"
)

// PublishHeartbeat publishes heartbeat of the service
func PublishHeartbeat(service string) {
	IncrCounter(
		[]string{"heartbeat", tags.Separator, "service", service},
		1,
	)
}

// PublishUptime publishes uptime of the service
func PublishUptime(service string, uptime time.Duration) {
	SetGauge(
		[]string{"uptime", "seconds", tags.Separator, "service", service},
		float32(uptime/time.Second),
	)
}
