package syslogutil

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
)

const MaxRetries int = 22

type SyslogConfig struct {
	ServerAddress string          `yaml:"server_address" json:"server_address"`
	Port          int             `yaml:"port" json:"port"`
	TLS           models.TLSCerts `yaml:"tls" json:"tls"`
}

type noopCounter struct{}

func (c *noopCounter) Add(_ float64) {} // intentionally empty: satisfies the counter interface but discards the value

func NewSyslogWriter(conf SyslogConfig) (egress.WriteCloser, error) {
	tlsConfig, err := conf.TLS.CreateClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	netConf := syslog.NetworkTimeoutConfig{
		WriteTimeout: time.Second,
		DialTimeout:  100 * time.Millisecond,
	}

	var protocol string
	if conf.TLS.CACertFile != "" {
		protocol = "syslog-tls"
	} else {
		protocol = "syslog"
	}

	syslogURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", protocol, conf.ServerAddress, conf.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to parse syslog URL: %w", err)
	}

	hostname, _ := os.Hostname()

	binding := &syslog.URLBinding{
		URL:      syslogURL,
		Hostname: hostname,
		Context:  context.Background(),
	}

	var writer egress.WriteCloser
	switch binding.URL.Scheme {
	case "syslog":
		writer = syslog.NewTCPWriter(
			binding,
			netConf,
			&noopCounter{},
			syslog.NewConverter(),
		)
	case "syslog-tls":
		writer = syslog.NewTLSWriter(
			binding,
			netConf,
			tlsConfig,
			&noopCounter{},
			syslog.NewConverter(),
		)
	default:
		return nil, fmt.Errorf("unsupported syslog scheme: %s", binding.URL.Scheme)
	}

	retryWriter, err := syslog.NewRetryWriter(
		binding,
		syslog.ExponentialDuration,
		MaxRetries,
		writer,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create syslog retry writer: %w", err)
	}

	return retryWriter, nil
}

func EnvelopeForMetric(metric *models.CustomMetric) *loggregator_v2.Envelope {
	return &loggregator_v2.Envelope{
		InstanceId: strconv.FormatUint(uint64(metric.InstanceIndex), 10),
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
