package aggregator

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache"
	"code.cloudfoundry.org/lager"
	"time"
)

type LogCacheClient struct {
	logger lager.Logger
	client logcache.Client
}

func NewLogCacheClient(logger lager.Logger, client logcache.Client) *LogCacheClient {
	return &LogCacheClient{
		logger: logger.Session("LogCacheClient"),
		client: client,
	}
}
func (c *LogCacheClient) GetMetric(appId string, metricType string, startTime time.Time, endTime time.Time) ([]*models.AppInstanceMetric, error) {
	c.logger.Debug("GetMetric")
	return nil, nil
}
