package aggregator

import (
	"code.cloudfoundry.org/lager"
	"dataaggregator/appmetric"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"metricscollector/metrics"
	"metricscollector/server"
	"net/http"
	"strconv"
	"time"
)

type MetricConsumer func(appMetric *appmetric.AppMetric)
type MetricPoller struct {
	metricCollectorUrl string
	logger             lager.Logger
	doneChan           chan bool
	appChan            chan *appmetric.AppMonitor
	metricConsumer     MetricConsumer
	httpClient         *http.Client
}

func NewMetricPoller(metricCollectorUrl string, logger lager.Logger, appChan chan *appmetric.AppMonitor, metricConsumer MetricConsumer, httpClient *http.Client) *MetricPoller {
	return &MetricPoller{
		metricCollectorUrl: metricCollectorUrl,
		logger:             logger.Session("MetricPoller"),
		appChan:            appChan,
		doneChan:           make(chan bool),
		metricConsumer:     metricConsumer,
		httpClient:         httpClient,
	}
}
func (m *MetricPoller) Start() {
	go m.startMetricRetrieve()
	m.logger.Info("started")
}

func (m *MetricPoller) Stop() {
	m.doneChan <- true
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
func (m *MetricPoller) retrieveMetric(app *appmetric.AppMonitor) {
	appId := app.AppId
	metricType := app.MetricType
	endTime := time.Now().UnixNano()
	startTime := endTime - int64(app.StatWindow)
	if metricType != "MemoryUsage" {
		m.logger.Error("Unsupported metric type", fmt.Errorf("%s is not supported", metricType))
		return
	}
	var url string
	url = m.metricCollectorUrl + "/v1/apps/" + app.AppId + "/metrics_history/memory?start=" + strconv.FormatInt(startTime, 10) + "&end=" + strconv.FormatInt(endTime, 10)
	resp, err := m.httpClient.Get(url)
	if err != nil {
		m.logger.Error("Retrieve metric failed", err, lager.Data{"appId": appId, "metricType": metricType, "err": err})
		return
	}
	defer resp.Body.Close()
	var metrics []*metrics.Metric
	if resp.StatusCode == http.StatusOK {
		data, readError := ioutil.ReadAll(resp.Body)
		if readError != nil {
			m.logger.Error("Can not read data from response", readError)
			return
		}
		json.Unmarshal(data, &metrics)
		avgMetric := m.doAggregate(appId, metricType, metrics)
		if avgMetric != nil {
			m.metricConsumer(avgMetric)
		}
	} else {
		var errorResponse server.ErrorResponse
		errBody, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(errBody, &errorResponse)
		m.logger.Error("Retrieve metric failed", errors.New(errorResponse.Message), lager.Data{"appId": appId, "metricType": metricType})
	}

}
func (m *MetricPoller) doAggregate(appId string, metricType string, metrics []*metrics.Metric) *appmetric.AppMetric {
	var count int64 = 0
	var sum int64 = 0
	var unit string
	var timestamp int64
	for _, metric := range metrics {
		unit = metric.Unit
		timestamp = metric.TimeStamp
		for _, instanceMetric := range metric.Instances {
			count++
			intValue, _ := strconv.ParseInt(instanceMetric.Value, 10, 64)
			sum += intValue
		}
	}
	var avgAppMetric *appmetric.AppMetric
	if count == 0 {
		avgAppMetric = &appmetric.AppMetric{
			AppId:      appId,
			MetricType: metricType,
			Value:      0,
			Unit:       "",
			Timestamp:  0,
		}
	} else {
		avgAppMetric = &appmetric.AppMetric{
			AppId:      appId,
			MetricType: metricType,
			Value:      sum / count,
			Unit:       unit,
			Timestamp:  timestamp,
		}
	}
	return avgAppMetric
}
