package emitter

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
)

const maxRetries int = 22

type Emitter interface {
	EmitMetric(metric *models.CustomMetric) error
}

type SyslogEmitter struct {
	logger lager.Logger
	writer egress.WriteCloser
}

type dummyCounter struct{}

func (c *dummyCounter) Add(_ float64) {}

func NewSyslogEmitter(logger lager.Logger, conf *config.Config) (Emitter, error) {
	var writer egress.WriteCloser

	tlsConfig, _ := conf.SyslogConfig.TLS.CreateClientConfig()

	netConf := syslog.NetworkTimeoutConfig{
		WriteTimeout: time.Second,
		DialTimeout:  100 * time.Millisecond,
	}

	var protocol string
	if conf.SyslogConfig.TLS.CACertFile != "" {
		protocol = "syslog-tls"
	} else {
		protocol = "syslog"
	}

	syslogURL, _ := url.Parse(fmt.Sprintf("%s://%s:%d", protocol, conf.SyslogConfig.ServerAddress, conf.SyslogConfig.Port))

	logger.Info("using-syslog-url", lager.Data{"url": syslogURL})

	hostname, _ := os.Hostname()

	binding := &syslog.URLBinding{
		URL:      syslogURL,
		Hostname: hostname,
		Context:  context.Background(),
	}

	switch binding.URL.Scheme {
	case "syslog":
		writer = syslog.NewTCPWriter(
			binding,
			netConf,
			&dummyCounter{},
			syslog.NewConverter(),
		)
	case "syslog-tls":
		writer = syslog.NewTLSWriter(
			binding,
			netConf,
			tlsConfig,
			&dummyCounter{},
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
		return nil, fmt.Errorf("failed to create syslog retry writer: %w", err)
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

func (e *SyslogEmitter) EmitMetric(metric *models.CustomMetric) error {
	if err := e.writer.Write(EnvelopeForMetric(metric)); err != nil {
		e.logger.Error("failed-to-write-metric-to-syslog", err)
		return fmt.Errorf("failed to write metric to syslog: %w", err)
	}
	return nil
}
