package client

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
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
	return mc.createLogCacheMetricClient(logger, conf)
}

func (mc *MetricClientFactory) createLogCacheMetricClient(logger lager.Logger, conf *config.Config) MetricClient {
	envelopeProcessor := NewProcessor(logger)
	c := NewLogCacheClient(logger, time.Now, conf.Aggregator.AggregatorExecuteInterval, envelopeProcessor, conf.MetricCollector.MetricCollectorURL)

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
	c.Configure()
	return c
}

func hasUAACreds(conf *config.Config) bool {
	return conf.MetricCollector.UAACreds.URL != "" && conf.MetricCollector.UAACreds.ClientSecret != "" &&
		conf.MetricCollector.UAACreds.ClientID != ""
}
