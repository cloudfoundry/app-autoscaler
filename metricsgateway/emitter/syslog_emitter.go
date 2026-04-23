package emitter

import (
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/syslogutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress"
)

type Emitter interface {
	EmitMetric(metric *models.CustomMetric) error
}

type SyslogEmitter struct {
	logger lager.Logger
	writer egress.WriteCloser
}

func NewSyslogEmitter(logger lager.Logger, conf *config.Config) (Emitter, error) {
	writer, err := syslogutil.NewSyslogWriter(syslogutil.SyslogConfig{
		ServerAddress: conf.SyslogConfig.ServerAddress,
		Port:          conf.SyslogConfig.Port,
		TLS:           conf.SyslogConfig.TLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create syslog writer: %w", err)
	}

	logger.Info("using-syslog-emitter")

	return &SyslogEmitter{
		writer: writer,
		logger: logger,
	}, nil
}

func (e *SyslogEmitter) EmitMetric(metric *models.CustomMetric) error {
	if err := e.writer.Write(syslogutil.EnvelopeForMetric(metric)); err != nil {
		e.logger.Error("failed-to-write-metric-to-syslog", err)
		return fmt.Errorf("failed to write metric to syslog: %w", err)
	}
	return nil
}
