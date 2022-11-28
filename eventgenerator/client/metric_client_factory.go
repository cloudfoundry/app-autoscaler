package client

import (
	"crypto/tls"
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

var GoLogCacheNewClient = logcache.NewClient
var GoLogCacheNewOauth2HTTPClient = logcache.NewOauth2HTTPClient
var GoLogCacheWithViaGRPC = logcache.WithViaGRPC
var GoLogCacheWithHTTPClient = logcache.WithHTTPClient
var GoLogCacheWithOauth2HTTPClient = logcache.WithOauth2HTTPClient

var NewProcessor = envelopeprocessor.NewProcessor
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
	if conf.MetricCollector.UseLogCache {
		return mc.createLogCacheMetricClient(logger, conf)
	} else {
		return mc.createMetricServerMetricClient(logger, conf)
	}
}

func (mc *MetricClientFactory) createLogCacheMetricClient(logger lager.Logger, conf *config.Config) MetricClient {
	var logCacheClient LogCacheClientReader

	if hasUAACreds(conf) {
		logCacheClient = createOauth2HTTPLogCacheClient(conf)
	} else {
		logCacheClient = createGRPCLogCacheClient(conf)
	}

	envelopeProcessor := NewProcessor(logger, conf.Aggregator.AggregatorExecuteInterval)
	return mc.newLogCacheClient(logger, time.Now, logCacheClient, envelopeProcessor)
}

func (mc *MetricClientFactory) createMetricServerMetricClient(logger lager.Logger, conf *config.Config) MetricClient {
	httpClient, err := helpers.CreateHTTPClient(&conf.MetricCollector.TLSClientCerts)

	if err != nil {
		logger.Error("failed to create http client for MetricCollector", err, lager.Data{"metriccollectorTLS": httpClient})
	}
	return mc.newMetricServerClient(logger, conf.MetricCollector.MetricCollectorURL, httpClient)
}

func createOauth2HTTPLogCacheClient(conf *config.Config) *logcache.Client {
	httpClient := createHttpClient(conf.MetricCollector.UAACreds.SkipSSLValidation)
	oauthHTTPOpt := GoLogCacheWithOauth2HTTPClient(httpClient)

	clientOpt := GoLogCacheNewOauth2HTTPClient(conf.MetricCollector.UAACreds.URL,
		conf.MetricCollector.UAACreds.ClientID, conf.MetricCollector.UAACreds.ClientSecret,
		oauthHTTPOpt)

	return GoLogCacheNewClient(conf.MetricCollector.MetricCollectorURL, GoLogCacheWithHTTPClient(clientOpt))
}

func createHttpClient(skipSSLValidation bool) *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			//nolint:gosec
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSLValidation},
		},
	}
}
func createGRPCLogCacheClient(conf *config.Config) *logcache.Client {
	creds, err := newTLSCredentials(conf.MetricCollector.TLSClientCerts.CACertFile,
		conf.MetricCollector.TLSClientCerts.CertFile, conf.MetricCollector.TLSClientCerts.KeyFile)
	if err != nil {
		log.Fatalf("failed to load TLS config: %s", err)
	}
	return GoLogCacheNewClient(conf.MetricCollector.MetricCollectorURL, GoLogCacheWithViaGRPC(new(grpcCreds).WithTransportCredentials(creds)))
}

func hasUAACreds(conf *config.Config) bool {
	return conf.MetricCollector.UAACreds.URL != "" && conf.MetricCollector.UAACreds.ClientSecret != "" &&
		conf.MetricCollector.UAACreds.ClientID != ""
}
