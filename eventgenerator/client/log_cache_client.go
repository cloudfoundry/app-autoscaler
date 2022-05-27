package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
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
	c.logger.Debug("GetMetric")
	logMetricType := getEnvelopeType(metricType)
	envelopes, _ := c.client.Read(context.Background(), appId, startTime, logcache.WithEndTime(endTime), logcache.WithEnvelopeTypes(logMetricType))
	var err error
	var metrics []models.AppInstanceMetric

	collectedAt := c.now().UnixNano()
	if logMetricType == rpc.EnvelopeType_TIMER {
		metrics = c.envelopeProcessor.GetHttpStartStopInstanceMetrics(envelopes, appId, collectedAt, 30*time.Second)
	} else {
		metrics, err = c.envelopeProcessor.GetGaugeInstanceMetrics(envelopes, collectedAt)
	}
	return metrics, err
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
