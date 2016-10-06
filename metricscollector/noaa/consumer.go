package noaa

import (
	"github.com/cloudfoundry/sonde-go/events"
)

type NoaaConsumer interface {
	ContainerEnvelopes(appGuid string, authToken string) ([]*events.Envelope, error)
}
