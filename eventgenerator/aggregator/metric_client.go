package aggregator

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type MetricClient interface {
	GetMetric(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error)
}
