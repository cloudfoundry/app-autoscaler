package operator

import (
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"
	"github.com/tedsuo/ifrit"
)

const PrunerLockSchemaKey = "pruner_lock"

func PrunerLockSchemaPath() string {
	return locket.LockSchemaPath(PrunerLockSchemaKey)
}

type ServiceClient interface {
	NewPrunerLockRunner(logger lager.Logger, PrunerID string, retryInterval, lockTTL time.Duration) ifrit.Runner
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

func (c serviceClient) NewPrunerLockRunner(logger lager.Logger, PrunerID string, retryInterval, lockTTL time.Duration) ifrit.Runner {
	return locket.NewLock(logger, c.consulClient, PrunerLockSchemaPath(), []byte(PrunerID), c.clock, retryInterval, lockTTL)
}
