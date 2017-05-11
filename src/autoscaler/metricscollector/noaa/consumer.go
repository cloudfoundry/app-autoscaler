package noaa

import (
	"github.com/cloudfoundry/sonde-go/events"
)

type NoaaConsumer interface {
	ContainerEnvelopes(appGuid string, authToken string) ([]*events.Envelope, error)
	Stream(appGuid string, authToken string) (outputChan <-chan *events.Envelope, errorChan <-chan error)
	Close() error
}
