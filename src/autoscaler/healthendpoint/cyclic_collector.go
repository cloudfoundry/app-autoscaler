package healthendpoint

import (
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/prometheus/client_golang/prometheus"
)

type CyclicCollector struct {
	logger   lager.Logger
	interval time.Duration
	clock    clock.Clock
	guage    prometheus.Gauge
	value    int64
	result   int64
	doneChan chan bool
}

func NewCyclicCollector(clock clock.Clock, interval time.Duration, logger lager.Logger, gaugeOpts prometheus.GaugeOpts) *CyclicCollector {
	return &CyclicCollector{
		logger:   logger.Session(gaugeOpts.Name + "_CyclicCollector"),
		interval: interval,
		clock:    clock,
		guage:    prometheus.NewGauge(gaugeOpts),
		doneChan: make(chan bool),
	}
}
func (c *CyclicCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.guage.Desc()
}

func (c *CyclicCollector) Collect(ch chan<- prometheus.Metric) {
	m, _ := prometheus.NewConstMetric(c.guage.Desc(), prometheus.GaugeValue, float64(c.result))
	ch <- m
}

func (c *CyclicCollector) Inc(count int64) {
	atomic.AddInt64(&c.value, count)
}
func (c *CyclicCollector) reset() {
	atomic.StoreInt64(&c.result, c.value)
	atomic.StoreInt64(&c.value, 0)
	c.logger.Debug("reset", lager.Data{"result": c.result})

}
func (c *CyclicCollector) Start() {
	c.logger.Info("started")
	go c.start()
}
func (c *CyclicCollector) start() {
	ticker := c.clock.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-c.doneChan:
			return
		case <-ticker.C():
			c.reset()
		}
	}
}
func (c *CyclicCollector) Stop() {
	close(c.doneChan)
	c.logger.Info("stopped")
}
