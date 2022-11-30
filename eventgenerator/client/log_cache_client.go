package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache"
	rpc "code.cloudfoundry.org/go-log-cache/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
)

type LogCacheClient struct {
	logger lager.Logger
	Client LogCacheClientReader

	now                  func() time.Time
	envelopeProcessor    envelopeprocessor.EnvelopeProcessor
	goLogCache           GoLogCache
	transportCredentials grpc.DialOption
}

// ClientOption configures the LogCache client.
type ClientOption interface {
	configure(client interface{})
}

// clientOptionFunc enables regular functions to be a ClientOption.
type clientOptionFunc func(client interface{})

// configure Implements clientOptionFunc.
func (f clientOptionFunc) configure(client interface{}) {
	f(client)
}

var NewTLS = credentials.NewTLS

type TLSConfig interface {
	NewTLS(c *tls.Config) credentials.TransportCredentials
}
type LogCacheClientReader interface {
	Read(ctx context.Context, sourceID string, start time.Time, opts ...logcache.ReadOption) ([]*loggregator_v2.Envelope, error)
}

type GoLogCacheClient interface {
	NewClient(addr string, opts ...logcache.ClientOption) *logcache.Client
	WithViaGRPC(opts ...grpc.DialOption) logcache.ClientOption
	WithHTTPClient(h logcache.HTTPClient) logcache.ClientOption
	NewOauth2HTTPClient(oauth2Addr, client, clientSecret string, opts ...logcache.Oauth2Option) *logcache.Oauth2HTTPClient
	WithOauth2HTTPClient(client logcache.HTTPClient) logcache.Oauth2Option
}

type LogCacheClientCreator interface {
	NewLogCacheClient(logger lager.Logger, getTime func() time.Time, client LogCacheClientReader, envelopeProcessor envelopeprocessor.EnvelopeProcessor) *LogCacheClient
}

type GoLogCache struct {
	NewClientFn            func(addr string, opts ...logcache.ClientOption) *logcache.Client
	WithViaGRPCFn          func(opts ...grpc.DialOption) logcache.ClientOption
	WithHTTPClientFn       func(h logcache.HTTPClient) logcache.ClientOption
	NewOauth2HTTPClientFn  func(oauth2Addr string, client string, clientSecret string, opts ...logcache.Oauth2Option) *logcache.Oauth2HTTPClient
	WithOauth2HTTPClientFn func(client logcache.HTTPClient) logcache.Oauth2Option
}

func NewLogCacheClient(logger lager.Logger, getTime func() time.Time, envelopeProcessor envelopeprocessor.EnvelopeProcessor, addrs string, opts ...ClientOption) *LogCacheClient {
	c := &LogCacheClient{
		logger: logger.Session("LogCacheClient"),

		envelopeProcessor: envelopeProcessor,
		now:               getTime,
		goLogCache: GoLogCache{
			NewClientFn:            logcache.NewClient,
			WithViaGRPCFn:          logcache.WithViaGRPC,
			WithHTTPClientFn:       logcache.WithHTTPClient,
			NewOauth2HTTPClientFn:  logcache.NewOauth2HTTPClient,
			WithOauth2HTTPClientFn: logcache.WithOauth2HTTPClient,
		},
	}

	for _, o := range opts {
		o.configure(c)
	}

	c.Client = c.goLogCache.NewClientFn(addrs,
		c.goLogCache.WithViaGRPCFn(c.transportCredentials),
	)

	return c
}

func WithLogCacheLibrary(l GoLogCache) ClientOption {
	return clientOptionFunc(func(c interface{}) {
		switch c := c.(type) {
		case *LogCacheClient:
			c.goLogCache = l
		default:
			panic("unknown type")
		}
	})
}

func WithGRPCTransportCredentials(d grpc.DialOption) ClientOption {
	return clientOptionFunc(func(c interface{}) {
		switch c := c.(type) {
		case *LogCacheClient:
			c.transportCredentials = d

		default:
			panic("unknown type")
		}
	})
}

func (c *LogCacheClient) GetMetric(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {
	var metrics []models.AppInstanceMetric

	var err error

	filters := logCacheFiltersFor(endTime, metricType)
	c.logger.Debug("GetMetric", lager.Data{"filters": valuesFrom(filters)})
	envelopes, err := c.Client.Read(context.Background(), appId, startTime, filters...)

	if err != nil {
		return metrics, fmt.Errorf("fail to Read %s metric from %s GoLogCache client: %w", getEnvelopeType(metricType), appId, err)
	}

	collectedAt := c.now().UnixNano()
	if getEnvelopeType(metricType) == rpc.EnvelopeType_TIMER {
		metrics = c.envelopeProcessor.GetTimerMetrics(envelopes, appId, collectedAt)
	} else {
		c.logger.Debug("envelopes received from log-cache: ", lager.Data{"envelopes": envelopes})
		metrics, err = c.envelopeProcessor.GetGaugeMetrics(envelopes, collectedAt)
	}
	return filter(metrics, metricType), err
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
	case models.MetricNameCPUUtil:
		readOptions = append(readOptions, logcache.WithNameFilter("cpu"))
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

//func newTLSCredentials(caPath string, certPath string, keyPath string) (credentials.TransportCredentials, error) {
//	cfg, err := NewTLSConfig(caPath, certPath, keyPath)
//	if err != nil {
//		return nil, err
//	}
//
//	return NewTLS(cfg), nil
//}
