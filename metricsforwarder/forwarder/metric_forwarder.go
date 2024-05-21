package forwarder

import (
	"fmt"
	"net/url"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v9"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
)

type Emitter interface {
	EmitMetric(*models.CustomMetric)
}

type SyslogEmitter struct {
	url      string
	hostname string
	netConf  syslog.NetworkTimeoutConfig
}

type MetronEmitter struct {
	client *loggregator.IngressClient
	logger lager.Logger
}

const METRICS_FORWARDER_ORIGIN = "autoscaler_metrics_forwarder"

func NewMetronEmitter(logger lager.Logger, conf *config.Config) (Emitter, error) {
	tlsConfig, err := loggregator.NewIngressTLSConfig(
		conf.LoggregatorConfig.TLS.CACertFile,
		conf.LoggregatorConfig.TLS.CertFile,
		conf.LoggregatorConfig.TLS.KeyFile,
	)
	if err != nil {
		logger.Error("could-not-create-TLS-config", err, lager.Data{"config": conf})
		return &MetronEmitter{}, err
	}

	client, err := loggregator.NewIngressClient(
		tlsConfig,
		loggregator.WithAddr(conf.LoggregatorConfig.MetronAddress),
		loggregator.WithTag("origin", METRICS_FORWARDER_ORIGIN),
		loggregator.WithLogger(helpers.NewLoggregatorGRPCLogger(logger.Session("metric_forwarder"))),
	)
	if err != nil {
		logger.Error("could-not-create-loggregator-client", err, lager.Data{"config": conf})
		return &MetronEmitter{}, err
	}

	return &MetronEmitter{
		client: client,
		logger: logger,
	}, nil

	return &MetronEmitter{}, nil
}

func hasLoggregatorConfig(conf *config.Config) bool {
	return conf.LoggregatorConfig.MetronAddress != ""
}

func NewSyslogEmitter(logger lager.Logger, conf *config.Config) (Emitter, error) {
	netConf := syslog.NetworkTimeoutConfig{
		WriteTimeout: time.Second,
		DialTimeout:  100 * time.Millisecond,
	}

	return &SyslogEmitter{
		url:      conf.SyslogConfig.ServerAddress,
		hostname: "test-hostname",
		netConf:  netConf,
	}, nil
}

func NewMetricForwarder(logger lager.Logger, conf *config.Config) (Emitter, error) {
	if hasLoggregatorConfig(conf) {
		return NewMetronEmitter(logger, conf)
	} else {
		return NewSyslogEmitter(logger, conf)
	}
}

type Counter struct{}

func (c *Counter) Add(delta float64) {
}
func (c *Counter) Set(delta float64) {
}

func (mf *SyslogEmitter) EmitMetric(metric *models.CustomMetric) {
	url, _ := url.Parse(fmt.Sprintf("syslog-tls://%s", mf.url))

	binding := &syslog.URLBinding{
		URL:      url,
		Hostname: mf.hostname,
	}

	w := syslog.NewTCPWriter(
		binding,
		mf.netConf,
		&Counter{},
		syslog.NewConverter(),
	)

	e := &loggregator_v2.Envelope{
		Timestamp: time.Now().UnixNano(),
		SourceId:  metric.AppGUID,
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

	w.Write(e)

}

func (mf *MetronEmitter) EmitMetric(metric *models.CustomMetric) {
	mf.logger.Debug("custom-metric-emit-request-received", lager.Data{"metric": metric})

	options := []loggregator.EmitGaugeOption{
		loggregator.WithGaugeAppInfo(metric.AppGUID, int(metric.InstanceIndex)),
		loggregator.WithGaugeValue(metric.Name, metric.Value, metric.Unit),
	}
	mf.client.EmitGauge(options...)
}
