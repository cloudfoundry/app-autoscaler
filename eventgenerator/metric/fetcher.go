package metric

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache/v3"
	"code.cloudfoundry.org/go-log-cache/v3/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
)

type Fetcher interface {
	FetchMetrics(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error)
}

type LogCacheFetcherCreator interface {
	NewLogCacheFetcher(logger lager.Logger, client LogCacheClient, envelopeProcessor envelopeprocessor.EnvelopeProcessor, collectionInterval time.Duration) Fetcher
}

type LogCacheClient interface {
	Read(ctx context.Context, sourceID string, start time.Time, opts ...logcache.ReadOption) ([]*loggregator_v2.Envelope, error)
	PromQL(ctx context.Context, query string, opts ...logcache.PromQLOption) (*logcache_v1.PromQL_InstantQueryResult, error)
}

var StandardLogCacheFetcherCreator = &logCacheFetcherCreator{}

type logCacheFetcherCreator struct{}

type logCacheFetcher struct {
	logger             lager.Logger
	logCacheClient     LogCacheClient
	envelopeProcessor  envelopeprocessor.EnvelopeProcessor
	collectionInterval time.Duration
}

func (s *logCacheFetcherCreator) NewLogCacheFetcher(logger lager.Logger, client LogCacheClient, envelopeProcessor envelopeprocessor.EnvelopeProcessor, collectionInterval time.Duration) Fetcher {
	return &logCacheFetcher{
		logCacheClient:     client,
		logger:             logger,
		envelopeProcessor:  envelopeProcessor,
		collectionInterval: collectionInterval,
	}
}

func (l *logCacheFetcher) FetchMetrics(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {
	// the log-cache REST API only return max. 1000 envelopes: https://github.com/cloudfoundry/log-cache-release/tree/main/src#get-apiv1readsource-id.
	// receiving a limited set of envelopes breaks throughput and responsetime, because all envelopes are required to calculate these metric types properly.
	// pagination via `start_time` and `end_time` could be done but is very error-prone.
	// using the PromQL API also has an advantage over REST API because it shifts the metric aggregations to log-cache.
	if metricType == models.MetricNameThroughput || metricType == models.MetricNameResponseTime {
		return l.getMetricsPromQLAPI(appId, metricType, l.collectionInterval)
	}

	return l.getMetricsRestAPI(appId, metricType, startTime, endTime)
}

func (l *logCacheFetcher) getMetricsPromQLAPI(appId string, metricType string, collectionInterval time.Duration) ([]models.AppInstanceMetric, error) {
	l.logger.Info("get-metric-promql-api", lager.Data{"appId": appId, "metricType": metricType})
	collectionIntervalSeconds := fmt.Sprintf("%.0f", collectionInterval.Seconds())
	now := time.Now()

	query := ""
	metricTypeUnit := ""
	if metricType == models.MetricNameThroughput {
		query = fmt.Sprintf("sum by (instance_id) (count_over_time(http{source_id='%s',peer_type='Client'}[%ss])) / %s", appId, collectionIntervalSeconds, collectionIntervalSeconds)
		metricTypeUnit = models.UnitRPS
	}

	if metricType == models.MetricNameResponseTime {
		query = fmt.Sprintf("avg by (instance_id) (max_over_time(http{source_id='%s',peer_type='Client'}[%ss])) / (1000 * 1000)", appId, collectionIntervalSeconds)
		metricTypeUnit = models.UnitMilliseconds
	}

	l.logger.Info("query-promql-api", lager.Data{"query": query})
	result, err := l.logCacheClient.PromQL(context.Background(), query, logcache.WithPromQLTime(now))
	if err != nil {
		return []models.AppInstanceMetric{}, fmt.Errorf("failed getting PromQL result (metricType: %s, appId: %s, collectionInterval: %s, query: %s, time: %s): %w", metricType, appId, collectionIntervalSeconds, query, now.String(), err)
	}
	l.logger.Info("received-promql-api-result", lager.Data{"result": result, "query": query})

	// safeguard: the query ensures that we get a vector but let's double-check
	vector := result.GetVector()
	if vector == nil {
		return []models.AppInstanceMetric{}, fmt.Errorf("result does not contain a vector")
	}

	// return empty metric if there are no samples, this usually happens in case there were no recent http-requests towards the application
	if len(vector.GetSamples()) <= 0 {
		return l.emptyAppInstanceMetrics(appId, metricType, metricTypeUnit, now)
	}

	// convert result into autoscaler metric model
	var metrics []models.AppInstanceMetric
	for _, sample := range vector.GetSamples() {
		// safeguard: metric label instance_id should be always there but let's double-check
		instanceIdStr, ok := sample.GetMetric()["instance_id"]
		if !ok {
			return []models.AppInstanceMetric{}, fmt.Errorf("sample does not contain instance_id: %w", err)
		}

		instanceIdUInt, err := strconv.ParseUint(instanceIdStr, 10, 32)
		if err != nil {
			return []models.AppInstanceMetric{}, fmt.Errorf("could not convert instance_id to uint32: %w", err)
		}

		// safeguard: the query ensures that we get a point but let's double-check
		point := sample.GetPoint()
		if point == nil {
			return []models.AppInstanceMetric{}, fmt.Errorf("sample does not contain a point")
		}

		instanceId := instanceIdUInt
		valueWithoutDecimalsRoundedToCeiling := fmt.Sprintf("%.0f", math.Ceil(point.GetValue()))

		metrics = append(metrics, models.AppInstanceMetric{
			AppId:         appId,
			InstanceIndex: instanceId,
			Name:          metricType,
			Unit:          metricTypeUnit,
			Value:         valueWithoutDecimalsRoundedToCeiling,
			CollectedAt:   now.UnixNano(),
			Timestamp:     now.UnixNano(),
		})
	}
	return metrics, nil
}

func (l *logCacheFetcher) emptyAppInstanceMetrics(appId string, name string, unit string, now time.Time) ([]models.AppInstanceMetric, error) {
	return []models.AppInstanceMetric{
		{
			AppId:         appId,
			InstanceIndex: 0,
			Name:          name,
			Unit:          unit,
			Value:         "0",
			CollectedAt:   now.UnixNano(),
			Timestamp:     now.UnixNano(),
		},
	}, nil
}

func (l *logCacheFetcher) getMetricsRestAPI(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {
	filters := l.readOptions(endTime, metricType)

	l.logger.Info("query-rest-api-with-filters", lager.Data{"appId": appId, "metricType": metricType, "startTime": startTime, "endTime": endTime, "filters": l.valuesFrom(filters)})
	envelopes, err := l.logCacheClient.Read(context.Background(), appId, startTime, filters...)
	if err != nil {
		return []models.AppInstanceMetric{}, fmt.Errorf("fail to Read %s metric from %s GoLogCache client: %w", logcache_v1.EnvelopeType_GAUGE, appId, err)
	}
	l.logger.Info("received-rest-api-result", lager.Data{"numEnvelopes": len(envelopes)})

	metrics := l.envelopeProcessor.GetGaugeMetrics(envelopes, time.Now().UnixNano())

	return l.filter(metrics, metricType), nil
}

func (l *logCacheFetcher) readOptions(endTime time.Time, metricType string) (readOptions []logcache.ReadOption) {
	readOptions = append(readOptions, logcache.WithEndTime(endTime))
	readOptions = append(readOptions, logcache.WithEnvelopeTypes(logcache_v1.EnvelopeType_GAUGE))

	switch metricType {
	case models.MetricNameMemoryUtil:
		readOptions = append(readOptions, logcache.WithNameFilter("memory|memory_quota"))
	case models.MetricNameMemoryUsed:
		readOptions = append(readOptions, logcache.WithNameFilter("memory"))
	case models.MetricNameCPU:
		readOptions = append(readOptions, logcache.WithNameFilter("cpu"))
	case models.MetricNameCPUUtil:
		readOptions = append(readOptions, logcache.WithNameFilter("cpu_entitlement"))
	case models.MetricNameDisk:
		readOptions = append(readOptions, logcache.WithNameFilter("disk"))
	case models.MetricNameDiskUtil:
		readOptions = append(readOptions, logcache.WithNameFilter("disk|disk_quota"))
	default:
		readOptions = append(readOptions, logcache.WithNameFilter(metricType))
	}

	return readOptions
}

func (l *logCacheFetcher) filter(metrics []models.AppInstanceMetric, metricType string) []models.AppInstanceMetric {
	var result []models.AppInstanceMetric
	for i := range metrics {
		if metrics[i].Name == metricType {
			result = append(result, metrics[i])
		}
	}

	return result
}

func (l *logCacheFetcher) valuesFrom(filters []logcache.ReadOption) url.Values {
	values := url.Values{}
	for _, f := range filters {
		f(nil, values)
	}
	return values
}
