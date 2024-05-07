package forwarder

import (
	"fmt"
	"net/url"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
)

type SyslogEmitter struct {
	url      string
	hostname string
	netConf  syslog.NetworkTimeoutConfig
}

type Counter struct{}

func (c *Counter) Add(delta float64) {
}
func (c *Counter) Set(delta float64) {
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
