package metricsgateway

import (
	"context"
	"crypto/tls"
	"time"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"autoscaler/healthendpoint"
	"autoscaler/helpers"
)

const METRICS_FORWARDER_ORIGIN = "autoscaler_metrics_forwarder"

var selectors = []*loggregator_v2.Selector{
	{
		Message: &loggregator_v2.Selector_Gauge{
			Gauge: &loggregator_v2.GaugeSelector{},
		},
	},
	{
		Message: &loggregator_v2.Selector_Timer{
			Timer: &loggregator_v2.TimerSelector{},
		},
	},
}
var envelopeCounter = prometheus.CounterOpts{
	Namespace: "autoscaler",
	Subsystem: "metricsgateway",
	Name:      "envelope_number_from_rlp",
	Help:      "the total envelopes number got from rlp",
}

type Nozzle struct {
	logger                   lager.Logger
	rlpAddr                  string
	tls                      *tls.Config
	envelopChan              chan *loggregator_v2.Envelope
	index                    int
	shardID                  string
	appIDs                   map[string]string
	getAppIDsFunc            GetAppIDsFunc
	context                  context.Context
	cancelFunc               context.CancelFunc
	envelopeCounterCollector healthendpoint.CounterCollector
}

func NewNozzle(logger lager.Logger, index int, shardID string, rlpAddr string, tls *tls.Config, envelopChan chan *loggregator_v2.Envelope, getAppIDsFunc GetAppIDsFunc, envelopeCounterCollector healthendpoint.CounterCollector) *Nozzle {
	ctx, cancelFunc := context.WithCancel(context.Background())
	envelopeCounterCollector.AddCounters(envelopeCounter)
	return &Nozzle{
		logger:                   logger.Session("Nozzle"),
		index:                    index,
		shardID:                  shardID,
		rlpAddr:                  rlpAddr,
		tls:                      tls,
		envelopChan:              envelopChan,
		getAppIDsFunc:            getAppIDsFunc,
		context:                  ctx,
		cancelFunc:               cancelFunc,
		envelopeCounterCollector: envelopeCounterCollector,
	}
}

func (n *Nozzle) Start() {
	n.envelopeCounterCollector.AddCounters(envelopeCounter)
	go n.streamMetrics()
	n.logger.Info("started", lager.Data{"index": n.index})
}

func (n *Nozzle) Stop() {
	n.cancelFunc()
}

func (n *Nozzle) streamMetrics() {
	streamConnector := loggregator.NewEnvelopeStreamConnector(n.rlpAddr, n.tls,
		loggregator.WithEnvelopeStreamLogger(helpers.NewLoggregatorGRPCLogger(n.logger.Session("envelope_streamer"))),
		loggregator.WithEnvelopeStreamConnectorDialOptions(grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             30 * time.Second,
			PermitWithoutStream: true,
		})),
	)
	rx := streamConnector.Stream(n.context, &loggregator_v2.EgressBatchRequest{
		ShardId:   n.shardID,
		Selectors: selectors,
	})
	for {
		select {
		case <-n.context.Done():
			n.logger.Info("nozzle-stopped", lager.Data{"index": n.index})
			return
		default:
		}
		envelops := rx()
		if envelops != nil {
			n.envelopeCounterCollector.Add(envelopeCounter, int64(len(envelops)))
			n.filterEnvelopes(envelops)
		}

	}
}

func (n *Nozzle) filterEnvelopes(envelops []*loggregator_v2.Envelope) {
	for _, e := range envelops {
		_, exist := n.getAppIDsFunc()[e.SourceId]
		if exist {
			switch e.GetMessage().(type) {
			case *loggregator_v2.Envelope_Gauge:
				if e.GetGauge().GetMetrics()["memory_quota"] != nil {
					n.logger.Debug("filter-envelopes-get-container-metrics", lager.Data{"index": n.index, "appID": e.SourceId, "message": e.Message})
					n.envelopChan <- e
				} else if e.GetDeprecatedTags()["origin"].GetText() == METRICS_FORWARDER_ORIGIN {
					n.logger.Debug("filter-envelopes-get-custom-metrics", lager.Data{"index": n.index, "appID": e.SourceId, "message": e.Message})
					n.envelopChan <- e
				}
			case *loggregator_v2.Envelope_Timer:
				peerType := e.GetTags()["peer_type"]
				if peerTypeFromDeprecatedTags := e.GetDeprecatedTags()["peer_type"]; peerType == "" && peerTypeFromDeprecatedTags != nil {
					peerType = peerTypeFromDeprecatedTags.GetText()
				}

				if e.GetTimer().GetName() == "http" && (peerType == "" || peerType == "Client") {
					n.logger.Debug("filter-envelopes-get-httpstartstop", lager.Data{"index": n.index, "appID": e.SourceId, "message": e.Message})
					n.envelopChan <- e
				}
			}
		}
	}
}
