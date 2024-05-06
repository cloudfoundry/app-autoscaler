package forwarder

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v9"
	"code.cloudfoundry.org/lager/v3"
)

type Emitter interface {
	EmitMetric(*models.CustomMetric)
}

type SyslogAgentForwarder struct {
}

type MetronAgentEmitter struct {
	client *loggregator.IngressClient
	logger lager.Logger
}

const METRICS_FORWARDER_ORIGIN = "autoscaler_metrics_forwarder"

func NewMetronAgentEmitter(logger lager.Logger, conf *config.Config) (Emitter, error) {
	tlsConfig, err := loggregator.NewIngressTLSConfig(
		conf.LoggregatorConfig.TLS.CACertFile,
		conf.LoggregatorConfig.TLS.CertFile,
		conf.LoggregatorConfig.TLS.KeyFile,
	)
	if err != nil {
		logger.Error("could-not-create-TLS-config", err, lager.Data{"config": conf})
		return &MetronAgentEmitter{}, err
	}

	client, err := loggregator.NewIngressClient(
		tlsConfig,
		loggregator.WithAddr(conf.LoggregatorConfig.MetronAddress),
		loggregator.WithTag("origin", METRICS_FORWARDER_ORIGIN),
		loggregator.WithLogger(helpers.NewLoggregatorGRPCLogger(logger.Session("metric_forwarder"))),
	)
	if err != nil {
		logger.Error("could-not-create-loggregator-client", err, lager.Data{"config": conf})
		return &MetronAgentEmitter{}, err
	}

	return &MetronAgentEmitter{
		client: client,
		logger: logger,
	}, nil

	return &MetronAgentEmitter{}, nil
}

func NewMetricForwarder(logger lager.Logger, conf *config.Config) (Emitter, error) {
	if conf.LoggregatorConfig.MetronAddress == "" {
		return &SyslogAgentForwarder{}, nil
	} else {
		return NewMetronAgentEmitter(logger, conf)
	}
}

func (mf *SyslogAgentForwarder) EmitMetric(metric *models.CustomMetric) {
}

func (mf *MetronAgentEmitter) EmitMetric(metric *models.CustomMetric) {
	mf.logger.Debug("custom-metric-emit-request-received", lager.Data{"metric": metric})

	options := []loggregator.EmitGaugeOption{
		loggregator.WithGaugeAppInfo(metric.AppGUID, int(metric.InstanceIndex)),
		loggregator.WithGaugeValue(metric.Name, metric.Value, metric.Unit),
	}
	mf.client.EmitGauge(options...)
}
