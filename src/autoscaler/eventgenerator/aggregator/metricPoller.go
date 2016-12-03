package aggregator

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/model"
	"autoscaler/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager"
)

type MetricPoller struct {
	logger             lager.Logger
	metricCollectorUrl string
	doneChan           chan bool
	appChan            chan *model.AppMonitor
	httpClient         *http.Client
	appMetricDB        db.AppMetricDB
}

func NewMetricPoller(logger lager.Logger, metricCollectorUrl string, appChan chan *model.AppMonitor, httpClient *http.Client, appMetricDB db.AppMetricDB) *MetricPoller {
	return &MetricPoller{
		metricCollectorUrl: metricCollectorUrl,
		logger:             logger.Session("MetricPoller"),
		appChan:            appChan,
		doneChan:           make(chan bool),
		httpClient:         httpClient,
		appMetricDB:        appMetricDB,
	}
}

func (m *MetricPoller) Start() {
	go m.startMetricRetrieve()
	m.logger.Info("started")
}

func (m *MetricPoller) Stop() {
	close(m.doneChan)
	m.logger.Info("stopped")
}

func (m *MetricPoller) startMetricRetrieve() {
	for {
		select {
		case <-m.doneChan:
			return
		case app := <-m.appChan:
			m.retrieveMetric(app)
		}
	}
}

func (m *MetricPoller) retrieveMetric(app *model.AppMonitor) {
	appId := app.AppId
	metricType := app.MetricType
	endTime := time.Now()
	startTime := endTime.Add(0 - app.StatWindow)
	if metricType != "MemoryUsage" {
		m.logger.Error("Unsupported metric type", fmt.Errorf("%s is not supported", metricType))
		return
	}

	var url string
	url = m.metricCollectorUrl + "/v1/apps/" + app.AppId + "/metrics_history/memory?start=" + strconv.FormatInt(startTime.UnixNano(), 10) + "&end=" + strconv.FormatInt(endTime.UnixNano(), 10)
	resp, err := m.httpClient.Get(url)
	if err != nil {
		m.logger.Error("Failed to retrieve metric from memory-collector. Request failed", err, lager.Data{"appId": appId, "metricType": metricType, "err": err})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Failed to retrieve metric from memory-collector", nil,
			lager.Data{"appId": appId, "metricType": metricType, "statusCode": resp.StatusCode})
		return
	}

	var metrics []*models.AppInstanceMetric
	err = json.NewDecoder(resp.Body).Decode(&metrics)
	if err != nil {
		m.logger.Error("Failed to parse response", err, lager.Data{"appId": appId, "metricType": metricType})
		return
	}

	avgMetric := m.aggregate(appId, metricType, metrics)
	if avgMetric == nil {
		return
	}

	err = m.appMetricDB.SaveAppMetric(avgMetric)
	if err != nil {
		m.logger.Error("Failed to save appmetric", err, lager.Data{"appmetric": avgMetric})
	}
}

func (m *MetricPoller) aggregate(appId string, metricType string, metrics []*models.AppInstanceMetric) *model.AppMetric {
	var count int64 = 0
	var sum int64 = 0
	var unit string
	var timestamp int64 = time.Now().UnixNano()
	for _, metric := range metrics {
		unit = metric.Unit
		intValue, err := strconv.ParseInt(metric.Value, 10, 64)
		if err != nil {
			m.logger.Error("failed-to-aggregate", err, lager.Data{"value": metric.Value})
		} else {
			count++
			sum += intValue
		}
	}

	if count == 0 {
		return &model.AppMetric{
			AppId:      appId,
			MetricType: metricType,
			Value:      nil,
			Unit:       "",
			Timestamp:  timestamp,
		}
	}

	avgValue := sum / count
	return &model.AppMetric{
		AppId:      appId,
		MetricType: metricType,
		Value:      &avgValue,
		Unit:       unit,
		Timestamp:  timestamp,
	}
}
