package forwarder

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/syslogutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress"
)

type SyslogEmitter struct {
	logger lager.Logger
	writer egress.WriteCloser
}

func (mf *SyslogEmitter) SetWriter(writer egress.WriteCloser) {
	mf.writer = writer
}

func (mf *SyslogEmitter) GetWriter() egress.WriteCloser {
	return mf.writer
}

func NewSyslogEmitter(logger lager.Logger, conf *config.Config) (MetricForwarder, error) {
	writer, err := syslogutil.NewSyslogWriter(syslogutil.SyslogConfig{
		ServerAddress: conf.SyslogConfig.ServerAddress,
		Port:          conf.SyslogConfig.Port,
		TLS:           conf.SyslogConfig.TLS,
	})
	if err != nil {
		return nil, err
	}

	return &SyslogEmitter{
		writer: writer,
		logger: logger,
	}, nil
}

func EnvelopeForMetric(metric *models.CustomMetric) *loggregator_v2.Envelope {
	return syslogutil.EnvelopeForMetric(metric)
}

func (mf *SyslogEmitter) EmitMetric(metric *models.CustomMetric) {
	if err := mf.writer.Write(syslogutil.EnvelopeForMetric(metric)); err != nil {
		mf.logger.Error("failed-to-write-metric-to-syslog", err)
	}
}
