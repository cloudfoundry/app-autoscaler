package metricscollector

import (
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"
	"github.com/tedsuo/ifrit"
)

const MetricsCollectorLockSchemaKey = "metricscollector_lock"

func MetricsCollectorLockSchemaPath() string {
	return locket.LockSchemaPath(MetricsCollectorLockSchemaKey)
}

type ServiceClient interface {
	NewMetricsCollectorLockRunner(logger lager.Logger, MetricsCollectorID string, retryInterval, lockTTL time.Duration) ifrit.Runner
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

func (c serviceClient) NewMetricsCollectorLockRunner(logger lager.Logger, MetricsCollectorID string, retryInterval, lockTTL time.Duration) ifrit.Runner {
	return locket.NewLock(logger, c.consulClient, MetricsCollectorLockSchemaPath(), []byte(MetricsCollectorID), c.clock, retryInterval, lockTTL)
}
