package operator

import (
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"
	"github.com/tedsuo/ifrit"
)

const OperatorLockSchemaKey = "operator_lock"

func OperatorLockSchemaPath() string {
	return locket.LockSchemaPath(OperatorLockSchemaKey)
}

type ServiceClient interface {
	NewOperatorLockRunner(logger lager.Logger, OperatorID string, retryInterval, lockTTL time.Duration) ifrit.Runner
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

func (c serviceClient) NewOperatorLockRunner(logger lager.Logger, OperatorID string, retryInterval, lockTTL time.Duration) ifrit.Runner {
	return locket.NewLock(logger, c.consulClient, OperatorLockSchemaPath(), []byte(OperatorID), c.clock, retryInterval, lockTTL)
}
