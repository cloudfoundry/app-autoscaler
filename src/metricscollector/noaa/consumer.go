package noaa

import (
	"github.com/cloudfoundry/sonde-go/events"
)

type NoaaConsumer interface {
	ContainerMetrics(appGuid string, authToken string) ([]*events.ContainerMetric, error)
}
