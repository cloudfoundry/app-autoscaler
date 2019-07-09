package healthendpoint

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type CounterCollector interface {
	prometheus.Collector
	AddCounters(counterOps ...prometheus.CounterOpts)
	Add(counterOps prometheus.CounterOpts, count int64)
}

func NewCounterCollector() CounterCollector {
	return &counterCollector{
		counterMap: map[string]prometheus.Counter{},
	}
}

type counterCollector struct {
	counterMap map[string]prometheus.Counter
	sync.RWMutex
}

func (c *counterCollector) AddCounters(counterOps ...prometheus.CounterOpts) {
	c.Lock()
	defer c.Unlock()
	for _, ops := range counterOps {
		counterFullName := getCounterFullName(ops)
		if _, exists := c.counterMap[counterFullName]; exists {
			continue
		}
		c.counterMap[counterFullName] = prometheus.NewCounter(ops)
	}

}
func (c *counterCollector) Describe(ch chan<- *prometheus.Desc) {
	c.RLock()
	defer c.RUnlock()
	for _, counter := range c.counterMap {
		ch <- counter.Desc()
	}
}

func (c *counterCollector) Collect(ch chan<- prometheus.Metric) {
	c.RLock()
	defer c.RUnlock()
	for _, counter := range c.counterMap {
		ch <- counter
	}
}

func (c *counterCollector) Add(counterOps prometheus.CounterOpts, count int64) {
	c.RLock()
	defer c.RUnlock()
	if counter, exists := c.counterMap[getCounterFullName(counterOps)]; exists {
		counter.Add(float64(count))
	}

}
func getCounterFullName(counterOps prometheus.CounterOpts) string {
	return counterOps.Namespace + "_" + counterOps.Subsystem + "_" + counterOps.Name
}
