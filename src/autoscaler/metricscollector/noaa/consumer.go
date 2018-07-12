package noaa

import (
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
)

type NoaaConsumer interface {
	ContainerEnvelopes(appGuid string, authToken string) ([]*events.Envelope, error)
	Stream(appGuid string, authToken string) (outputChan <-chan *events.Envelope, errorChan <-chan error)
	FilteredFirehose(subscriptionId string, authToken string, filter consumer.EnvelopeFilter) (<-chan *events.Envelope, <-chan error)
	Close() error
}
