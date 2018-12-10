package metricsgateway

import (
	"autoscaler/models"

	"code.cloudfoundry.org/lager"
)

const SHARD_ID = "CF_AUTOSCALER"

type Nozzle struct {
	logger      lager.Logger
	rlpAddr     string
	tls         *models.TLSCerts
	metricsChan chan<- *models.AppInstanceMetric
	doneChan    chan bool
	index       int
}

func NewNozzle(logger lager.Logger, index int, rlpAddr string, tls *models.TLSCerts, metricsChan chan<- *models.AppInstanceMetric) *Nozzle {
	return &Nozzle{
		logger:      logger,
		index:       index,
		rlpAddr:     rlpAddr,
		tls:         tls,
		metricsChan: metricsChan,
		doneChan:    make(chan bool),
	}
}

func (n *Nozzle) Start() {

}

func (n *Nozzle) Stop() {
	n.doneChan <- true
	n.logger.Info("stopped", lager.Data{"index": n.index})
}
