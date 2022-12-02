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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type MetricClient interface {
	GetMetric(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error)
}

type newLogCacheClient func(logger lager.Logger, getTime func() time.Time, client LogCacheClientReader, envelopeProcessor envelopeprocessor.EnvelopeProcessor) *LogCacheClient
type newMetricServerClient func(logger lager.Logger, metricCollectorUrl string, httpClient *http.Client) *MetricServerClient

type Factory struct {
	GoLogCacheNewClient            func(addr string, opts ...logcache.ClientOption) *logcache.Client
	GoLogCacheNewOauth2HTTPClient  func(oauth2Addr, client, clientSecret string, opts ...logcache.Oauth2Option) *logcache.Oauth2HTTPClient
	GoLogCacheWithViaGRPC          func(opts ...grpc.DialOption) logcache.ClientOption
	GoLogCacheWithHTTPClient       func(h logcache.HTTPClient) logcache.ClientOption
	GoLogCacheWithOauth2HTTPClient func(client logcache.HTTPClient) logcache.Oauth2Option
	NewProcessor                   func(logger lager.Logger, collectionInterval time.Duration) envelopeprocessor.Processor
	GRPCWithTransportCredentials   func(creds credentials.TransportCredentials) grpc.DialOption
	NewTLS                         func(config *tls.Config) credentials.TransportCredentials
}

type grpcDialOptions interface {
	WithTransportCredentials(creds credentials.TransportCredentials) grpc.DialOption
}

type grpcCreds struct {
	grpcDialOptions
	*Factory
}

func newFactory() *Factory {
	return &Factory{
		GoLogCacheNewClient:            logcache.NewClient,
		GoLogCacheNewOauth2HTTPClient:  logcache.NewOauth2HTTPClient,
		GoLogCacheWithViaGRPC:          logcache.WithViaGRPC,
		GoLogCacheWithHTTPClient:       logcache.WithHTTPClient,
		GoLogCacheWithOauth2HTTPClient: logcache.WithOauth2HTTPClient,
		NewProcessor:                   envelopeprocessor.NewProcessor,
		GRPCWithTransportCredentials:   grpc.WithTransportCredentials,
		NewTLS:                         credentials.NewTLS,
	}
}

func (g grpcCreds) WithTransportCredentials(creds credentials.TransportCredentials) grpc.DialOption {
	return g.Factory.GRPCWithTransportCredentials(creds)
}

type MetricClientFactory struct {
	newLogCacheClient     newLogCacheClient
	newMetricServerClient newMetricServerClient
	Factory               *Factory
}

func NewMetricClientFactory(newMetricLogCacheClient newLogCacheClient, newMetricServerClient newMetricServerClient) *MetricClientFactory {
	return &MetricClientFactory{
		newMetricServerClient: newMetricServerClient,
		newLogCacheClient:     newMetricLogCacheClient,
		Factory:               newFactory(),
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
		logCacheClient = mc.Factory.createOauth2HTTPLogCacheClient(conf)
	} else {
		logCacheClient = mc.Factory.createGRPCLogCacheClient(conf)
	}

	envelopeProcessor := mc.Factory.NewProcessor(logger, conf.Aggregator.AggregatorExecuteInterval)
	return mc.newLogCacheClient(logger, time.Now, logCacheClient, envelopeProcessor)
}

func (mc *MetricClientFactory) createMetricServerMetricClient(logger lager.Logger, conf *config.Config) MetricClient {
	httpClient, err := helpers.CreateHTTPClient(&conf.MetricCollector.TLSClientCerts)

	if err != nil {
		logger.Error("failed to create http client for MetricCollector", err, lager.Data{"metriccollectorTLS": httpClient})
	}
	return mc.newMetricServerClient(logger, conf.MetricCollector.MetricCollectorURL, httpClient)
}

func (f *Factory) createOauth2HTTPLogCacheClient(conf *config.Config) *logcache.Client {
	httpClient := createHttpClient(conf.MetricCollector.UAACreds.SkipSSLValidation)
	oauthHTTPOpt := f.GoLogCacheWithOauth2HTTPClient(httpClient)

	clientOpt := f.GoLogCacheNewOauth2HTTPClient(conf.MetricCollector.UAACreds.URL,
		conf.MetricCollector.UAACreds.ClientID, conf.MetricCollector.UAACreds.ClientSecret,
		oauthHTTPOpt)

	return f.GoLogCacheNewClient(conf.MetricCollector.MetricCollectorURL, f.GoLogCacheWithHTTPClient(clientOpt))
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
func (f *Factory) createGRPCLogCacheClient(conf *config.Config) *logcache.Client {
	creds, err := f.newTLSCredentials(conf.MetricCollector.TLSClientCerts.CACertFile,
		conf.MetricCollector.TLSClientCerts.CertFile, conf.MetricCollector.TLSClientCerts.KeyFile)
	if err != nil {
		log.Fatalf("failed to load TLS config: %s", err)
	}
	return f.GoLogCacheNewClient(conf.MetricCollector.MetricCollectorURL, f.GoLogCacheWithViaGRPC(grpcCreds{Factory: f}.WithTransportCredentials(creds)))
}

func hasUAACreds(conf *config.Config) bool {
	return conf.MetricCollector.UAACreds.URL != "" && conf.MetricCollector.UAACreds.ClientSecret != "" &&
		conf.MetricCollector.UAACreds.ClientID != ""
}
