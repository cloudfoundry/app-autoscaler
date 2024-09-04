package forwarder

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
)

const maxRetries int = 22

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

type DummyCounter struct{}

func (c *DummyCounter) Add(delta float64) {
}

func NewSyslogEmitter(logger lager.Logger, conf *config.Config) (MetricForwarder, error) {
	var writer egress.WriteCloser
	var protocol string

	tlsConfig, _ := conf.SyslogConfig.TLS.CreateClientConfig()

	netConf := syslog.NetworkTimeoutConfig{
		WriteTimeout: time.Second,
		DialTimeout:  100 * time.Millisecond,
	}

	if conf.SyslogConfig.TLS.CACertFile != "" {
		protocol = "syslog-tls"
	} else {
		protocol = "syslog"
	}

	syslogUrl, _ := url.Parse(fmt.Sprintf("%s://%s:%d", protocol, conf.SyslogConfig.ServerAddress, conf.SyslogConfig.Port))

	logger.Info("using-syslog-url", lager.Data{"url": syslogUrl})

	hostname, _ := os.Hostname()

	binding := &syslog.URLBinding{
		URL:      syslogUrl,
		Hostname: hostname,
		Context:  context.Background(),
	}

	switch binding.URL.Scheme {
	case "syslog":
		writer = syslog.NewTCPWriter(
			binding,
			netConf,
			&DummyCounter{},
			syslog.NewConverter(),
		)
	case "syslog-tls":
		writer = syslog.NewTLSWriter(
			binding,
			netConf,
			tlsConfig,
			&DummyCounter{},
			syslog.NewConverter(),
		)
	}

	retryWriter, err := syslog.NewRetryWriter(
		binding,
		syslog.ExponentialDuration,
		maxRetries,
		writer,
	)

	if err != nil {
		return nil, err
	}

	return &SyslogEmitter{
		writer: retryWriter,
		logger: logger,
	}, nil
}

func EnvelopeForMetric(metric *models.CustomMetric) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		InstanceId: fmt.Sprintf("%d", metric.InstanceIndex),
		Timestamp:  time.Now().UnixNano(),
		SourceId:   metric.AppGUID,
		Message: &loggregator_v2.Envelope_Gauge{
			Gauge: &loggregator_v2.Gauge{
				Metrics: map[string]*loggregator_v2.GaugeValue{
					metric.Name: {
						Unit:  metric.Unit,
						Value: metric.Value,
					},
				},
			},
		},
	}
}
func (mf *SyslogEmitter) EmitMetric(metric *models.CustomMetric) {
	err := mf.writer.Write(EnvelopeForMetric(metric))

	if err != nil {
		mf.logger.Error("failed-to-write-metric-to-syslog", err)
	}
}
