package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
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
	logger            lager.Logger
	client            LogCacheClientReader
	now               func() time.Time
	envelopeProcessor envelopeprocessor.EnvelopeProcessor
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
}

type LogCacheClientCreator interface {
	NewLogCacheClient(logger lager.Logger, getTime func() time.Time, client LogCacheClientReader, envelopeProcessor envelopeprocessor.EnvelopeProcessor) *LogCacheClient
}

func NewLogCacheClient(logger lager.Logger, getTime func() time.Time, client LogCacheClientReader, envelopeProcessor envelopeprocessor.EnvelopeProcessor) *LogCacheClient {
	return &LogCacheClient{
		logger: logger.Session("LogCacheClient"),
		client: client,

		envelopeProcessor: envelopeProcessor,
		now:               getTime,
	}
}
func (c *LogCacheClient) GetMetric(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {
	var metrics []models.AppInstanceMetric

	var err error

	//TODO: filters for "throughput" and "responsetime"
	filters := logCacheFiltersFor(endTime, metricType)
	c.logger.Debug("GetMetric", lager.Data{"filters": valuesFrom(filters)})
	envelopes, err := c.client.Read(context.Background(), appId, startTime, filters...)
	//TODO: merge all envelopes with matching timestamp, source_id and instance_id

	if err != nil {
		return metrics, fmt.Errorf("fail to Read %s metric from %s GoLogCache client: %w", getEnvelopeType(metricType), appId, err)
	}

	collectedAt := c.now().UnixNano()
	if getEnvelopeType(metricType) == rpc.EnvelopeType_TIMER {
		metrics = c.envelopeProcessor.GetHttpStartStopInstanceMetrics(envelopes, appId, collectedAt, 30*time.Second)
	} else {
		c.logger.Debug("envelopes received from log-cache: ", lager.Data{"envelopes": envelopes})
		metrics, err = c.envelopeProcessor.GetGaugeInstanceMetrics(envelopes, collectedAt)
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
		readOptions = append(readOptions, logcache.WithNameFilter("memory_quota"))
		readOptions = append(readOptions, logcache.WithNameFilter("memory"))
	case models.MetricNameMemoryUsed:
		readOptions = append(readOptions, logcache.WithNameFilter("memory"))
	case models.MetricNameCPUUtil:
		readOptions = append(readOptions, logcache.WithNameFilter("cpu"))
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

func NewTLSCredentials(caPath string, certPath string, keyPath string) (credentials.TransportCredentials, error) {
	cfg, err := NewTLSConfig(caPath, certPath, keyPath)
	if err != nil {
		return nil, err
	}

	return NewTLS(cfg), nil
}

func NewTLSConfig(caPath string, certPath string, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	caCertBytes, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertBytes); !ok {
		return nil, errors.New("cannot parse ca cert")
	}

	tlsConfig.RootCAs = caCertPool

	return tlsConfig, nil
}
