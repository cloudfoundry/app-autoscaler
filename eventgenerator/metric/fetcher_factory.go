package metric

import (
	"crypto/tls"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	logcache "code.cloudfoundry.org/go-log-cache/v3"
	"code.cloudfoundry.org/lager/v3"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type FetcherFactory interface {
	CreateFetcher(logger lager.Logger, conf config.Config) (Fetcher, error)
}

type logCacheFetcherFactory struct {
	fetcherCreator LogCacheFetcherCreator
}

func NewLogCacheFetcherFactory(fetcherCreator LogCacheFetcherCreator) FetcherFactory {
	return &logCacheFetcherFactory{
		fetcherCreator: fetcherCreator,
	}
}

func (l *logCacheFetcherFactory) CreateFetcher(logger lager.Logger, conf config.Config) (Fetcher, error) {
	var options []logcache.ClientOption

	metricsCollectorConfig := conf.MetricCollector
	uaaCredsConfig := metricsCollectorConfig.UAACreds
	if uaaCredsConfig.IsNotEmpty() {
		oauth2HTTPClient := logcache.NewOauth2HTTPClient(uaaCredsConfig.URL, uaaCredsConfig.ClientID, uaaCredsConfig.ClientSecret, logcache.WithOauth2HTTPClient(&http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// #nosec G402
					InsecureSkipVerify: uaaCredsConfig.SkipSSLValidation,
				},
			},
		}))
		options = append(options, logcache.WithHTTPClient(oauth2HTTPClient))
	} else {
		tlsConfig, err := metricsCollectorConfig.TLSClientCerts.CreateClientConfig()
		if err != nil {
			return nil, err
		}
		options = append(options, logcache.WithViaGRPC(gogrpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))))
	}

	return l.fetcherCreator.NewLogCacheFetcher(
		logger,
		logcache.NewClient(metricsCollectorConfig.MetricCollectorURL, options...),
		envelopeprocessor.NewProcessor(logger),
		conf.Aggregator.AggregatorExecuteInterval,
	), nil
}
