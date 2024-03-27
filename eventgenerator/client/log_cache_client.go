package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	gogrpc "google.golang.org/grpc"

	"google.golang.org/grpc/credentials"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache/v2"
	rpc "code.cloudfoundry.org/go-log-cache/v2/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
)

type LogCacheClient struct {
	logger lager.Logger
	Client LogCacheClientReader

	now               func() time.Time
	envelopeProcessor envelopeprocessor.EnvelopeProcessor
	goLogCache        GoLogCache
	TLSConfig         *tls.Config
	uaaCreds          models.UAACreds
	url               string

	grpc GRPC
}

type LogCacheClientReader interface {
	Read(ctx context.Context, sourceID string, start time.Time, opts ...logcache.ReadOption) ([]*loggregator_v2.Envelope, error)
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
	NewLogCacheClient(logger lager.Logger, getTime func() time.Time, envelopeProcessor envelopeprocessor.EnvelopeProcessor, addrs string) MetricClient
}

func NewLogCacheClient(logger lager.Logger, getTime func() time.Time, envelopeProcessor envelopeprocessor.EnvelopeProcessor, url string) *LogCacheClient {
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

		grpc: GRPC{
			WithTransportCredentials: gogrpc.WithTransportCredentials,
		},
	}
	return c
}

func (c *LogCacheClient) GetMetrics(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {
	var metrics []models.AppInstanceMetric

	var err error

	filters := logCacheFiltersFor(endTime, metricType)
	c.logger.Debug("GetMetrics", lager.Data{"filters": valuesFrom(filters)})
	envelopes, err := c.Client.Read(context.Background(), appId, startTime, filters...)

	if err != nil {
		return metrics, fmt.Errorf("fail to Read %s metric from %s GoLogCache client: %w", getEnvelopeType(metricType), appId, err)
	}

	collectedAt := c.now().UnixNano()
	if getEnvelopeType(metricType) == rpc.EnvelopeType_TIMER {
		metrics = c.envelopeProcessor.GetTimerMetrics(envelopes, appId, collectedAt)
	} else {
		c.logger.Debug("envelopes received from log-cache", lager.Data{"envelopes": envelopes})
		metrics, err = c.envelopeProcessor.GetGaugeMetrics(envelopes, collectedAt)
	}
	return filter(metrics, metricType), err
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
	logMetricType := getEnvelopeType(metricType)
	readOptions = append(readOptions, logcache.WithEndTime(endTime))
	readOptions = append(readOptions, logcache.WithEnvelopeTypes(logMetricType))

	switch metricType {
	case models.MetricNameMemoryUtil:
		readOptions = append(readOptions, logcache.WithNameFilter("memory|memory_quota"))
	case models.MetricNameMemoryUsed:
		readOptions = append(readOptions, logcache.WithNameFilter("memory"))
	case models.MetricNameCPU:
		readOptions = append(readOptions, logcache.WithNameFilter("cpu"))
	case models.MetricNameCPUUtil:
		readOptions = append(readOptions, logcache.WithNameFilter("cpu_entitlement"))
	case models.MetricNameResponseTime, models.MetricNameThroughput:
		readOptions = append(readOptions, logcache.WithNameFilter("http"))
	default:
		readOptions = append(readOptions, logcache.WithNameFilter(metricType))
	}

	return readOptions
}

func getEnvelopeType(metricType string) rpc.EnvelopeType {
	var metricName rpc.EnvelopeType
	switch metricType {
	case models.MetricNameThroughput, models.MetricNameResponseTime:
		metricName = rpc.EnvelopeType_TIMER
	default:
		metricName = rpc.EnvelopeType_GAUGE
	}
	return metricName
}

func (c *LogCacheClient) getUaaHttpClient() logcache.HTTPClient {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: c.GetUaaTlsConfig(),
		},
	}
}
