package aggregator

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	"context"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache"
	rpc "code.cloudfoundry.org/go-log-cache/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
)

type LogCacheClient struct {
	logger            lager.Logger
	client            LogCacheClientReader
	now               func() time.Time
	envelopeProcessor envelopeprocessor.EnvelopeProcessor
}

type LogCacheClientReader interface {
	Read(ctx context.Context, sourceID string, start time.Time, opts ...logcache.ReadOption) ([]*loggregator_v2.Envelope, error)
}

func NewLogCacheClient(logger lager.Logger, getTime func() time.Time, client LogCacheClientReader, envelopeProcessor envelopeprocessor.EnvelopeProcessor) *LogCacheClient {

	//client := client.NewClient(
	//	cfg.LogCacheAddr,
	//	client.WithViaGRPC(
	//		grpc.WithTransportCredentials(cfg.TLS.Credentials("log-cache")),
	//	),
	//)
	return &LogCacheClient{
		logger:            logger.Session("LogCacheClient"),
		client:            client,
		envelopeProcessor: envelopeProcessor,
		now:               getTime,
	}
}
func (c *LogCacheClient) GetMetric(appId string, metricType string, startTime time.Time, endTime time.Time) ([]models.AppInstanceMetric, error) {

	metricName := rpc.EnvelopeType_TIMER
	switch metricType {
	case models.MetricNameMemoryUsed:
		metricName = rpc.EnvelopeType_GAUGE
	}

	envelopes, _ := c.client.Read(context.Background(), appId, startTime, logcache.WithEndTime(endTime), logcache.WithEnvelopeTypes(metricName))
	var err error
	var metrics []models.AppInstanceMetric

	collectedAt := c.now().UnixNano()
	if metricType == models.MetricNameThroughput {
		metrics = c.envelopeProcessor.GetHttpStartStopInstanceMetrics(envelopes, appId, collectedAt, 30*time.Second)
	} else {
		for _, envelope := range envelopes {
			var metrics_temp []models.AppInstanceMetric
			metrics_temp, err = c.envelopeProcessor.GetGaugeInstanceMetrics(envelope, collectedAt)
			metrics = append(metrics, metrics_temp...)
		}
	}
	return metrics, err
}
