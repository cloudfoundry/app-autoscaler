package client

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager"
)

type GetMetricFunc func(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error)

type MetricClient interface {
	GetMetrics(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error)
}

var NewProcessor = envelopeprocessor.NewProcessor

type MetricClientFactory struct {
}

func NewMetricClientFactory() *MetricClientFactory {
	return &MetricClientFactory{}
}

func (mc *MetricClientFactory) GetMetricClient(logger lager.Logger, conf *config.Config) MetricClient {
	if conf.MetricCollector.UseLogCache {
		return mc.createLogCacheMetricClient(logger, conf)
	} else {
		return mc.createMetricServerMetricClient(logger, conf)
	}
}

func (mc *MetricClientFactory) createLogCacheMetricClient(logger lager.Logger, conf *config.Config) MetricClient {
	envelopeProcessor := NewProcessor(logger, conf.Aggregator.AggregatorExecuteInterval)
	c := NewLogCacheClient(logger, time.Now, envelopeProcessor, conf.MetricCollector.MetricCollectorURL)

	if hasUAACreds(conf) {
		uaaCreds := models.UAACreds{
			URL:               conf.MetricCollector.UAACreds.URL,
			ClientID:          conf.MetricCollector.UAACreds.ClientID,
			ClientSecret:      conf.MetricCollector.UAACreds.ClientSecret,
			SkipSSLValidation: conf.MetricCollector.UAACreds.SkipSSLValidation,
		}
		c.SetUAACreds(uaaCreds)
	} else {
		tlsCerts := &models.TLSCerts{
			KeyFile:    conf.MetricCollector.TLSClientCerts.KeyFile,
			CertFile:   conf.MetricCollector.TLSClientCerts.CertFile,
			CACertFile: conf.MetricCollector.TLSClientCerts.CACertFile,
		}
		tlsConfig, _ := tlsCerts.CreateClientConfig()
		c.SetTLSConfig(tlsConfig)
	}
	return c
}

func (mc *MetricClientFactory) createMetricServerMetricClient(logger lager.Logger, conf *config.Config) MetricClient {
	httpClient, err := helpers.CreateHTTPClient(&conf.MetricCollector.TLSClientCerts)

	if err != nil {
		logger.Error("failed to create http client for MetricCollector", err, lager.Data{"metriccollectorTLS": httpClient})
	}
	return NewMetricServerClient(logger, conf.MetricCollector.MetricCollectorURL, httpClient)
}

func hasUAACreds(conf *config.Config) bool {
	return conf.MetricCollector.UAACreds.URL != "" && conf.MetricCollector.UAACreds.ClientSecret != "" &&
		conf.MetricCollector.UAACreds.ClientID != ""
}
