package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	gogrpc "google.golang.org/grpc"

	"google.golang.org/grpc/credentials"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache/v2"
	"code.cloudfoundry.org/go-log-cache/v2/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
)

type LogCacheClient struct {
	logger lager.Logger
	Client LogCacheClientReader

	now                func() time.Time
	envelopeProcessor  envelopeprocessor.EnvelopeProcessor
	goLogCache         GoLogCache
	TLSConfig          *tls.Config
	uaaCreds           models.UAACreds
	url                string
	collectionInterval time.Duration

	grpc GRPC
}

type LogCacheClientReader interface {
	Read(ctx context.Context, sourceID string, start time.Time, opts ...logcache.ReadOption) ([]*loggregator_v2.Envelope, error)
	PromQL(ctx context.Context, query string, opts ...logcache.PromQLOption) (*logcache_v1.PromQL_InstantQueryResult, error)
}

type GRPCOptions interface {
	WithTransportCredentials(creds credentials.TransportCredentials) gogrpc.DialOption
}

type GRPC struct {
	WithTransportCredentials func(creds credentials.TransportCredentials) gogrpc.DialOption
}

type GoLogCacheClient interface {
	NewClient(addr string, opts ...logcache.ClientOption) *logcache.Client
	WithViaGRPC(opts ...gogrpc.DialOption) logcache.ClientOption
	WithHTTPClient(h logcache.HTTPClient) logcache.ClientOption
	NewOauth2HTTPClient(oauth2Addr, client, clientSecret string, opts ...logcache.Oauth2Option) *logcache.Oauth2HTTPClient
	WithOauth2HTTPClient(client logcache.HTTPClient) logcache.Oauth2Option
}

type GoLogCache struct {
	NewClient            func(addr string, opts ...logcache.ClientOption) *logcache.Client
	WithViaGRPC          func(opts ...gogrpc.DialOption) logcache.ClientOption
	WithHTTPClient       func(h logcache.HTTPClient) logcache.ClientOption
	NewOauth2HTTPClient  func(oauth2Addr string, client string, clientSecret string, opts ...logcache.Oauth2Option) *logcache.Oauth2HTTPClient
	WithOauth2HTTPClient func(client logcache.HTTPClient) logcache.Oauth2Option
}

type LogCacheClientCreator interface {
	NewLogCacheClient(logger lager.Logger, getTime func() time.Time, collectionInterval time.Duration, envelopeProcessor envelopeprocessor.EnvelopeProcessor, url string) MetricClient
}

func NewLogCacheClient(logger lager.Logger, getTime func() time.Time, collectionInterval time.Duration, envelopeProcessor envelopeprocessor.EnvelopeProcessor, url string) *LogCacheClient {
	var c = &LogCacheClient{
		logger: logger.Session("LogCacheClient"),

		envelopeProcessor: envelopeProcessor,
		now:               getTime,
		url:               url,
		goLogCache: GoLogCache{
			NewClient:            logcache.NewClient,
			WithViaGRPC:          logcache.WithViaGRPC,
			WithHTTPClient:       logcache.WithHTTPClient,
			NewOauth2HTTPClient:  logcache.NewOauth2HTTPClient,
			WithOauth2HTTPClient: logcache.WithOauth2HTTPClient,
		},
		collectionInterval: collectionInterval,

		grpc: GRPC{
			WithTransportCredentials: gogrpc.WithTransportCredentials,
		},
	}
	return c
}

func (c *LogCacheClient) emptyAppInstanceMetrics(appId string, name string, unit string, now time.Time) ([]models.AppInstanceMetric, error) {
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

func (c *LogCacheClient) getMetricsPromQLAPI(appId string, metricType string) ([]models.AppInstanceMetric, error) {
	collectionInterval := fmt.Sprintf("%.0f", c.CollectionInterval().Seconds())
	now := time.Now()

	query := ""
	metricTypeUnit := ""
	if metricType == models.MetricNameThroughput {
		query = fmt.Sprintf("sum by (instance_id) (count_over_time(http{source_id='%s'}[%ss])) / %s", appId, collectionInterval, collectionInterval)
		metricTypeUnit = models.UnitRPS
	}

	if metricType == models.MetricNameResponseTime {
		query = fmt.Sprintf("avg by (instance_id) (max_over_time(http{source_id='%s'}[%ss])) / (1000 * 1000)", appId, collectionInterval)
		metricTypeUnit = models.UnitMilliseconds
	}

	c.logger.Info("query-promql-api", lager.Data{"query": query, "appId": appId, "metricType": metricType})
	result, err := c.Client.PromQL(context.Background(), query, logcache.WithPromQLTime(now))
	if err != nil {
		return []models.AppInstanceMetric{}, fmt.Errorf("failed getting PromQL result (metricType: %s, appId: %s, collectionInterval: %s, query: %s, time: %s): %w", metricType, appId, collectionInterval, query, now.String(), err)
	}
	c.logger.Info("received-promql-api-result", lager.Data{"result": result})

	// safeguard: the query ensures that we get a vector but let's double-check
	vector := result.GetVector()
	if vector == nil {
		return []models.AppInstanceMetric{}, fmt.Errorf("result does not contain a vector")
	}

	// return empty metrics if there are no samples, this usually happens in case there were no recent http-requests towards the application
	if len(vector.GetSamples()) <= 0 {
		return c.emptyAppInstanceMetrics(appId, metricType, metricTypeUnit, now)
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

		instanceId := uint32(instanceIdUInt)
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

func (c *LogCacheClient) getMetricsRestAPI(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {
	filters := logCacheFiltersFor(endTime, metricType)
	c.logger.Info("query-rest-api-with-filters", lager.Data{"filters": valuesFrom(filters)})

	envelopes, err := c.Client.Read(context.Background(), appId, startTime, filters...)
	if err != nil {
		return []models.AppInstanceMetric{}, fmt.Errorf("fail to Read %s metric from %s GoLogCache client: %w", logcache_v1.EnvelopeType_GAUGE, appId, err)
	}
	c.logger.Info("received-rest-api-result", lager.Data{"envelopes": envelopes})

	collectedAt := c.now().UnixNano()
	metrics, err := c.envelopeProcessor.GetGaugeMetrics(envelopes, collectedAt)

	return filter(metrics, metricType), err
}

func (c *LogCacheClient) GetMetrics(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {
	c.logger.Debug("GetMetrics")

	// the log-cache REST API only return max. 1000 envelopes: https://github.com/cloudfoundry/log-cache-release/tree/main/src#get-apiv1readsource-id.
	// receiving a limited set of envelopes breaks throughput and responsetime, because all envelopes are required to calculate these metric types properly.
	// pagination via `start_time` and `end_time` could be done but is very error-prone.
	// using the PromQL API also has an advantage over REST API because it shifts the metric aggregations to log-cache.
	if metricType == models.MetricNameThroughput || metricType == models.MetricNameResponseTime {
		c.logger.Debug("get-metrics-via-promql-api", lager.Data{"metricType": metricType})
		return c.getMetricsPromQLAPI(appId, metricType)
	}

	c.logger.Debug("get-metrics-via-rest-api", lager.Data{"metricType": metricType})
	return c.getMetricsRestAPI(appId, metricType, startTime, endTime)
}

func (c *LogCacheClient) SetTLSConfig(tlsConfig *tls.Config) {
	c.TLSConfig = tlsConfig
}

func (c *LogCacheClient) GetTlsConfig() *tls.Config {
	return c.TLSConfig
}

func (c *LogCacheClient) SetUAACreds(uaaCreds models.UAACreds) {
	c.uaaCreds = uaaCreds
}

func (c *LogCacheClient) GetUAACreds() models.UAACreds {
	return c.uaaCreds
}

func (c *LogCacheClient) GetUrl() string {
	return c.url
}

func (c *LogCacheClient) SetGoLogCache(goLogCache GoLogCache) {
	c.goLogCache = goLogCache
}

func (c *LogCacheClient) SetGRPC(grpc GRPC) {
	c.grpc = grpc
}

func (c *LogCacheClient) Configure() {
	var opts []logcache.ClientOption

	if c.uaaCreds.IsEmpty() {
		opts = append(opts, c.goLogCache.WithViaGRPC(c.grpc.WithTransportCredentials(credentials.NewTLS(c.TLSConfig))))
	} else {
		oauth2HTTPOpts := c.goLogCache.WithOauth2HTTPClient(c.getUaaHttpClient())
		oauth2HTTPClient := c.goLogCache.NewOauth2HTTPClient(c.uaaCreds.URL, c.uaaCreds.ClientID, c.uaaCreds.ClientSecret, oauth2HTTPOpts)
		opts = append(opts, c.goLogCache.WithHTTPClient(oauth2HTTPClient))
	}

	c.Client = c.goLogCache.NewClient(c.url, opts...)
}

func (c *LogCacheClient) GetUaaTlsConfig() *tls.Config {
	//nolint:gosec
	return &tls.Config{InsecureSkipVerify: c.uaaCreds.SkipSSLValidation}
}

func (c *LogCacheClient) CollectionInterval() time.Duration {
	return c.collectionInterval
}

func valuesFrom(filters []logcache.ReadOption) url.Values {
	values := url.Values{}
	for _, f := range filters {
		f(nil, values)
	}
	return values
}

func filter(metrics []models.AppInstanceMetric, metricType string) []models.AppInstanceMetric {
	var result []models.AppInstanceMetric
	for i := range metrics {
		if metrics[i].Name == metricType {
			result = append(result, metrics[i])
		}
	}

	return result
}

func logCacheFiltersFor(endTime time.Time, metricType string) (readOptions []logcache.ReadOption) {
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

func (c *LogCacheClient) getUaaHttpClient() logcache.HTTPClient {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: c.GetUaaTlsConfig(),
		},
	}
}
