package generator_test

import (
	"autoscaler/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strconv"
	"time"

	"testing"
)

func TestGenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Generator Suite")
}

func generateTestAppMetrics(appId string, metricType string, unit string, metricValues []int64, breachDurationSecs int, enoughMetrics bool) []*models.AppMetric {
	appMetrics := []*models.AppMetric{}
	if enoughMetrics {
		appMetrics = append(appMetrics, &models.AppMetric{
			AppId:      appId,
			MetricType: metricType,
			Value:      "10",
			Unit:       unit,
			Timestamp:  time.Now().UnixNano() - int64(time.Duration(breachDurationSecs)*time.Second),
		})
	}
	for _, value := range metricValues {
		appMetrics = append(appMetrics, &models.AppMetric{
			AppId:      appId,
			MetricType: metricType,
			Value:      strconv.FormatInt(value, 10),
			Unit:       unit,
			Timestamp:  time.Now().UnixNano(),
		})
	}
	return appMetrics
}
