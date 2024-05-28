package forwarder

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
	"github.com/prometheus/client_golang/prometheus"
)

type SyslogEmitter struct {
	logger lager.Logger
	Writer egress.WriteCloser
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
	}

	syslogEgressCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "autoscaler",
		Subsystem: "metricsforwarder",
		Name:      "syslog_metrics_egress_counter",
		Help:      "count of metrics sent to syslog",
	})

	switch binding.URL.Scheme {
	case "syslog":
		writer = syslog.NewTCPWriter(
			binding,
			netConf,
			syslogEgressCounter,
			syslog.NewConverter(),
		)
	case "syslog-tls":
		writer = syslog.NewTLSWriter(
			binding,
			netConf,
			tlsConfig,
			syslogEgressCounter,
			syslog.NewConverter(),
		)
	}

	return &SyslogEmitter{
		Writer: writer,
		logger: logger,
	}, nil
}

func (mf *SyslogEmitter) EmitMetric(metric *models.CustomMetric) {
	e := &loggregator_v2.Envelope{
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

	err := mf.Writer.Write(e)
	if err != nil {
		mf.logger.Error("failed-to-write-metric-to-syslog", err)
	}
}
