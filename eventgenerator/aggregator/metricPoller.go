package aggregator

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager"
)

type MetricPoller struct {
	logger             lager.Logger
	metricCollectorUrl string
	doneChan           chan bool
	appMonitorsChan    chan *models.AppMonitor
	httpClient         *http.Client
	appMetricChan      chan *models.AppMetric
}

func NewMetricPoller(logger lager.Logger, metricCollectorUrl string, appMonitorsChan chan *models.AppMonitor, httpClient *http.Client, appMetricChan chan *models.AppMetric) *MetricPoller {
	return &MetricPoller{
		metricCollectorUrl: metricCollectorUrl,
		logger:             logger.Session("MetricPoller"),
		appMonitorsChan:    appMonitorsChan,
		doneChan:           make(chan bool),
		httpClient:         httpClient,
		appMetricChan:      appMetricChan,
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
			m.retrieveMetric(appMonitor)
		}
	}
}

func (m *MetricPoller) retrieveMetric(appMonitor *models.AppMonitor) {
	appId := appMonitor.AppId
	metricType := appMonitor.MetricType
	endTime := time.Now()
	startTime := endTime.Add(0 - appMonitor.StatWindow)

	var url string
	path, _ := routes.MetricsCollectorRoutes().Get(routes.GetMetricHistoriesRouteName).URLPath("appid", appMonitor.AppId, "metrictype", metricType)
	parameters := path.Query()
	parameters.Add("start", strconv.FormatInt(startTime.UnixNano(), 10))
	parameters.Add("end", strconv.FormatInt(endTime.UnixNano(), 10))
	parameters.Add("order", db.ASCSTR)
	url = m.metricCollectorUrl + path.RequestURI() + "?" + parameters.Encode()
	resp, err := m.httpClient.Get(url)
	if err != nil {
		m.logger.Error("Failed to retrieve metric from metrics collector. Request failed", err, lager.Data{"appId": appId, "metricType": metricType, "url": url})
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
	m.logger.Debug("save-aggregated-appmetric", lager.Data{"appMetric": avgMetric})
	m.appMetricChan <- avgMetric
}

func (m *MetricPoller) aggregate(appId string, metricType string, metrics []*models.AppInstanceMetric) *models.AppMetric {
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
