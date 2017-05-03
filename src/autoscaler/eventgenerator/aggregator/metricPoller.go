package aggregator

import (
	"autoscaler/db"
	"autoscaler/models"
	"autoscaler/routes"
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
	appChan            chan *models.AppMonitor
	httpClient         *http.Client
	appMetricDB        db.AppMetricDB
}

func NewMetricPoller(logger lager.Logger, metricCollectorUrl string, appChan chan *models.AppMonitor, httpClient *http.Client, appMetricDB db.AppMetricDB) *MetricPoller {
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

func (m *MetricPoller) retrieveMetric(app *models.AppMonitor) {
	appId := app.AppId
	metricType := app.MetricType
	endTime := time.Now()
	startTime := endTime.Add(0 - app.StatWindow)
	if metricType != models.MetricNameMemory {
		m.logger.Error("Unsupported metric type", fmt.Errorf("%s is not supported", metricType))
		return
	}

	var url string
	path, _ := routes.MetricsCollectorRoutes().Get(routes.GetMemoryMetricHistoriesRouteName).URLPath("appid", app.AppId)
	parameters := path.Query()
	parameters.Add("start", strconv.FormatInt(startTime.UnixNano(), 10))
	parameters.Add("end", strconv.FormatInt(endTime.UnixNano(), 10))
	url = m.metricCollectorUrl + path.RequestURI() + "?" + parameters.Encode()
	resp, err := m.httpClient.Get(url)
	if err != nil {
		m.logger.Error("Failed to retrieve metric from metrics collector. Request failed", err, lager.Data{"appId": appId, "metricType": metricType, "err": err})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Failed to retrieve metric from metrics collector", nil,
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

func (m *MetricPoller) aggregate(appId string, metricType string, metrics []*models.AppInstanceMetric) *models.AppMetric {
	var count int64 = 0
	var sum int64 = 0
	var unit string
	var timestamp int64 = time.Now().UnixNano()
	for _, metric := range metrics {
		unit = metric.Unit
		metricValue, err := strconv.ParseInt(metric.Value, 10, 64)
		if err != nil {
			m.logger.Error("failed-to-aggregate", err, lager.Data{"value": metric.Value})
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

	avgValue := int64(float64(sum)/float64(count) + 0.5)
	return &models.AppMetric{
		AppId:      appId,
		MetricType: metricType,
		Value:      fmt.Sprintf("%d", avgValue),
		Unit:       unit,
		Timestamp:  timestamp,
	}
}
