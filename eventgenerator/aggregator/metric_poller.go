package aggregator

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager"
)

type MetricPoller struct {
	logger          lager.Logger
	doneChan        chan bool
	metricClient    MetricClient
	appMonitorsChan chan *models.AppMonitor
	appMetricChan   chan *models.AppMetric
}

func NewMetricPoller(logger lager.Logger, metricClient MetricClient, appMonitorsChan chan *models.AppMonitor, appMetricChan chan *models.AppMetric) *MetricPoller {
	return &MetricPoller{
		logger:          logger.Session("MetricPoller"),
		appMonitorsChan: appMonitorsChan,
		metricClient:    metricClient,
		doneChan:        make(chan bool),
		appMetricChan:   appMetricChan,
	}
}

func (m *MetricPoller) Start() {
	go m.startMetricRetrieve()
	m.logger.Info("started")
}

func (m *MetricPoller) Stop() {
	close(m.doneChan)
}

func (m *MetricPoller) startMetricRetrieve() {
	for {
		select {
		case <-m.doneChan:
			m.logger.Info("stopped")
			return
		case appMonitor := <-m.appMonitorsChan:
			err := m.retrieveMetric(appMonitor)
			if err != nil {
				m.logger.Error("Error:", err)
			}
		}
	}
}

func (m *MetricPoller) retrieveMetric(appMonitor *models.AppMonitor) error {
	var metrics []models.AppInstanceMetric
	appId := appMonitor.AppId
	metricType := appMonitor.MetricType
	statWindow := appMonitor.StatWindow
	endTime := time.Now()
	startTime := endTime.Add(0 - statWindow)

	metrics, err := m.metricClient.GetMetric(appId, metricType, startTime, endTime)
	if err != nil {
		return fmt.Errorf("retriveMetric Failed: %w", err)
	}
	avgMetric := m.aggregate(appId, metricType, metrics)
	m.logger.Debug("save-aggregated-appmetric", lager.Data{"appMetric": avgMetric})
	m.appMetricChan <- avgMetric
	return nil
}

func (m *MetricPoller) aggregate(appId string, metricType string, metrics []models.AppInstanceMetric) *models.AppMetric {
	var count int64
	var sum int64
	var unit string
	timestamp := time.Now().UnixNano()
	for _, metric := range metrics {
		unit = metric.Unit
		metricValue, err := strconv.ParseInt(metric.Value, 10, 64)
		if err != nil {
			m.logger.Error("failed-to-aggregate", err, lager.Data{"appid": appId, "metrictype": metricType, "value": metric.Value})
		} else {
			count++
			sum += metricValue
		}
	}

	if count == 0 {
		return &models.AppMetric{
			AppId:      appId,
			MetricType: metricType,
			Value:      "",
			Unit:       "",
			Timestamp:  timestamp,
		}
	}

	avgValue := int64(math.Ceil(float64(sum) / float64(count)))
	return &models.AppMetric{
		AppId:      appId,
		MetricType: metricType,
		Value:      fmt.Sprintf("%d", avgValue),
		Unit:       unit,
		Timestamp:  timestamp,
	}
}
