package eventgenerator

import (
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"
	"github.com/tedsuo/ifrit"
)

const EventGeneratorLockSchemaKey = "eventgenerator_lock"

func EventGeneratorLockSchemaPath() string {
	return locket.LockSchemaPath(EventGeneratorLockSchemaKey)
}

type ServiceClient interface {
	NewEventGeneratorLockRunner(logger lager.Logger, EventGeneratorID string, retryInterval, lockTTL time.Duration) ifrit.Runner
}

type serviceClient struct {
	consulClient consuladapter.Client
	clock        clock.Clock
}

func NewServiceClient(consulClient consuladapter.Client, clock clock.Clock) ServiceClient {
	return serviceClient{
		consulClient: consulClient,
		clock:        clock,
	}
}

func (c serviceClient) NewEventGeneratorLockRunner(logger lager.Logger, EventGeneratorID string, retryInterval, lockTTL time.Duration) ifrit.Runner {
	return locket.NewLock(logger, c.consulClient, EventGeneratorLockSchemaPath(), []byte(EventGeneratorID), c.clock, retryInterval, lockTTL)
}
