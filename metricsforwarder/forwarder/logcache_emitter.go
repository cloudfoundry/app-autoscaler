package forwarder

import (
	"context"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	rpc "code.cloudfoundry.org/go-log-cache/v3/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type LogCacheEmitter struct {
	logger lager.Logger
	client rpc.IngressClient
	conn   *grpc.ClientConn
}

func NewLogCacheEmitter(logger lager.Logger, conf *config.Config) (MetricForwarder, error) {
	address := conf.LogCacheConfig.Address

	var dialOpt grpc.DialOption

	tlsConfig, err := conf.LogCacheConfig.TLS.CreateClientConfig()
	if err != nil || tlsConfig == nil {
		logger.Info("using-logcache-grpc-insecure", lager.Data{"address": address})
		dialOpt = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		logger.Info("using-logcache-grpc-tls", lager.Data{"address": address})
		dialOpt = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	}

	conn, err := grpc.NewClient(address, dialOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to create logcache grpc client: %w", err)
	}

	return &LogCacheEmitter{
		logger: logger,
		client: rpc.NewIngressClient(conn),
		conn:   conn,
	}, nil
}

func (e *LogCacheEmitter) EmitMetric(metric *models.CustomMetric) {
	envelope := EnvelopeForMetric(metric)

	_, err := e.client.Send(context.Background(), &rpc.SendRequest{
		Envelopes: &loggregator_v2.EnvelopeBatch{
			Batch: []*loggregator_v2.Envelope{envelope},
		},
	})
	if err != nil {
		e.logger.Error("failed-to-send-metric-to-logcache", err)
	}
}
