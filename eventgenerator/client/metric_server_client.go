package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/lager/v3"
)

type MetricServerClient struct {
	httpClient *http.Client
	logger     lager.Logger
	url        string
}

type NewMetricsServerClientFunc func(logger lager.Logger, metricCollectorUrl string, httpClient *http.Client) MetricClient

type MetricServerClientCreator interface {
	NewMetricServerClient(logger lager.Logger, metricCollectorUrl string, httpClient *http.Client) MetricClient
}

func NewMetricServerClient(logger lager.Logger, url string, httpClient *http.Client) *MetricServerClient {
	return &MetricServerClient{
		logger:     logger.Session("MetricServerClient"),
		url:        url,
		httpClient: httpClient,
	}
}
func (c *MetricServerClient) GetMetrics(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {
	c.logger.Debug("GetMetrics")
	var url string
	path, err := routes.MetricsCollectorRoutes().Get(routes.GetMetricHistoriesRouteName).URLPath("appid", appId, "metrictype", metricType)
	if err != nil {
		c.logger.Error("Failed to retrieve metric from metrics collector. Request failed", err, lager.Data{"appId": appId, "metricType": metricType})
		return nil, fmt.Errorf("Failed to retrieve metric from metrics collector. Request failed: %w", err)
	}
	parameters := path.Query()
	parameters.Add("start", strconv.FormatInt(startTime.UnixNano(), 10))
	parameters.Add("end", strconv.FormatInt(endTime.UnixNano(), 10))
	parameters.Add("order", db.ASCSTR)

	url = c.url + path.RequestURI() + "?" + parameters.Encode()
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

func (c *MetricServerClient) GetUrl() string {
	return c.url
}
