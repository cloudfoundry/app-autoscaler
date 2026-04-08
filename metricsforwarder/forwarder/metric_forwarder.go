package forwarder

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"

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

func NewMetricForwarder(logger lager.Logger, conf *config.Config) (MetricForwarder, error) {
	if conf.UsingLogCache() {
		logger.Info("using-logcache-emitter")
		return NewLogCacheEmitter(logger, conf)
	}
	logger.Info("using-metron-emitter")
	return NewMetronEmitter(logger, conf)
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
