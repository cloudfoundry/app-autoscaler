package forwarder

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10"

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

func hasSyslogConfig(conf *config.Config) bool {
	return conf.SyslogConfig.ServerAddress != ""
}

func NewMetricForwarder(logger lager.Logger, conf *config.Config) (MetricForwarder, error) {
	if hasSyslogConfig(conf) {
		logger.Info("using-syslog-emitter")
		return NewSyslogEmitter(logger, conf)
	} else {
		logger.Info("using-metron-emitter")
		return NewMetronEmitter(logger, conf)
	}
}
