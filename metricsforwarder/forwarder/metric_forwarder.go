package forwarder

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v9"

	"code.cloudfoundry.org/lager/v3"
)

type MetricForwarder interface {
	EmitMetric(*models.CustomMetric)
}

type MetronEmitter struct {
	client *loggregator.IngressClient
	logger lager.Logger
}

const METRICS_FORWARDER_ORIGIN = "autoscaler_metrics_forwarder"

func hasLoggregatorConfig(conf *config.Config) bool {
	return conf.LoggregatorConfig.MetronAddress != ""
}

func NewMetricForwarder(logger lager.Logger, conf *config.Config) (MetricForwarder, error) {
	if hasLoggregatorConfig(conf) {
		logger.Info("Using metron-emitter")
		return NewMetronEmitter(logger, conf)
	} else {
		logger.Info("Using syslog-emitter")
		return NewSyslogEmitter(logger, conf)
	}
}
