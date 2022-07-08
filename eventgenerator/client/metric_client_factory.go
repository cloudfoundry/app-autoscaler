package client

import (
	"log"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache"
	"code.cloudfoundry.org/lager"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type MetricClient interface {
	GetMetric(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error)
}

type newLogCacheClient func(logger lager.Logger, getTime func() time.Time, client LogCacheClientReader, envelopeProcessor envelopeprocessor.EnvelopeProcessor) *LogCacheClient
type newMetricServerClient func(logger lager.Logger, metricCollectorUrl string, httpClient *http.Client) *MetricServerClient

var NewGoLogCacheClient = logcache.NewClient
var NewProcessor = envelopeprocessor.NewProcessor
var LogCacheClientWithGRPC = logcache.WithViaGRPC
var GRPCWithTransportCredentials = gogrpc.WithTransportCredentials

type grpcDialOptions interface {
	WithTransportCredentials(creds credentials.TransportCredentials) gogrpc.DialOption
}

type grpcCreds struct {
	grpcDialOptions
}

func (g grpcCreds) WithTransportCredentials(creds credentials.TransportCredentials) gogrpc.DialOption {
	return GRPCWithTransportCredentials(creds)
}

type MetricClientFactory struct {
	newLogCacheClient     newLogCacheClient
	newMetricServerClient newMetricServerClient
}

func NewMetricClientFactory(newMetricLogCacheClient newLogCacheClient, newMetricServerClient newMetricServerClient) *MetricClientFactory {
	return &MetricClientFactory{
		newMetricServerClient: newMetricServerClient,
		newLogCacheClient:     newMetricLogCacheClient,
	}
}

func (mc *MetricClientFactory) GetMetricClient(logger lager.Logger, conf *config.Config) MetricClient {
	var metricClient MetricClient

	if conf.MetricCollector.UseLogCache {
		creds, err := NewTLSCredentials(conf.MetricCollector.TLSClientCerts.CACertFile, conf.MetricCollector.TLSClientCerts.CertFile, conf.MetricCollector.TLSClientCerts.KeyFile)

		logCacheClient := NewGoLogCacheClient(conf.MetricCollector.MetricCollectorURL, LogCacheClientWithGRPC(new(grpcCreds).WithTransportCredentials(creds)))
		if err != nil {
			log.Fatalf("failed to load TLS config: %s", err)
		}

		envelopeProcessor := NewProcessor(logger, conf.Aggregator.AggregatorExecuteInterval)
		metricClient = mc.newLogCacheClient(logger, time.Now, logCacheClient, envelopeProcessor)
	} else {
		httpClient, err := helpers.CreateHTTPClient(&conf.MetricCollector.TLSClientCerts)

		if err != nil {
			logger.Error("failed to create http client for MetricCollector", err, lager.Data{"metriccollectorTLS": httpClient})
		}
		metricClient = mc.newMetricServerClient(logger, conf.MetricCollector.MetricCollectorURL, httpClient)
	}
	return metricClient
}
