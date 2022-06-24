package aggregator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/lager"
)

type MetricServerClient struct {
	metricCollectorUrl string
	httpClient         *http.Client
	logger             lager.Logger
}

func NewMetricServerClient(logger lager.Logger, metricCollectorUrl string, tlsClientCerts *models.TLSCerts) *MetricServerClient {

	httpClient, err := helpers.CreateHTTPClient(tlsClientCerts)

	if err != nil {
		logger.Error("failed to create http client for MetricCollector", err, lager.Data{"metriccollectorTLS": tlsClientCerts})
	}
	httpClient.Transport.(*http.Transport).MaxIdleConnsPerHost = 1
	return &MetricServerClient{
		logger:             logger.Session("MetricServerClient"),
		metricCollectorUrl: metricCollectorUrl,
		httpClient:         httpClient,
	}
}
func (c *MetricServerClient) GetMetric(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {
	c.logger.Debug("GetMetric")
	var url string
	path, _ := routes.MetricsCollectorRoutes().Get(routes.GetMetricHistoriesRouteName).URLPath("appid", appId, "metrictype", metricType)
	parameters := path.Query()
	parameters.Add("start", strconv.FormatInt(startTime.UnixNano(), 10))
	parameters.Add("end", strconv.FormatInt(endTime.UnixNano(), 10))
	parameters.Add("order", db.ASCSTR)

	url = c.metricCollectorUrl + path.RequestURI() + "?" + parameters.Encode()
	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.logger.Error("Failed to retrieve metric from metrics collector. Request failed", err, lager.Data{"appId": appId, "metricType": metricType, "url": url})
		return nil, fmt.Errorf("failed to retrieve metric from metrics collector: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Failed to retrieve metric from metrics collector", nil,
			lager.Data{"appId": appId, "metricType": metricType, "statusCode": resp.StatusCode})
		return nil, errors.New("Failed to retrieve metric from metrics collector")
	}

	var metrics []models.AppInstanceMetric
	err = json.NewDecoder(resp.Body).Decode(&metrics)
	if err != nil {
		c.logger.Error("Failed to parse response", err, lager.Data{"appId": appId, "metricType": metricType})
		return nil, errors.New("Failed to parse response")
	}

	return metrics, nil
}
