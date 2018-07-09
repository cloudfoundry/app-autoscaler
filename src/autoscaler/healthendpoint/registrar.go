package healthendpoint

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/prometheus/client_golang/prometheus"
)

// Registrar maintains a list of metrics to be served by the health endpoint
// server.
type Registrar struct {
	gauges map[string]prometheus.Gauge
	logger lager.Logger
}

// New returns an initialized health endpoint registrar configured with the
// given prmetheus.Registerer and map of prometheus.Gauges.
// use includeDefault to add collector for Process and GoRuntime
func New(registrar prometheus.Registerer, gauges map[string]prometheus.Gauge, includeDefault bool, logger lager.Logger) *Registrar {

	defer func() {
		if err := recover(); err != nil {
			logger.Info("recover from panic", lager.Data{"error": err})
		}
	}()

	if includeDefault {
		registrar.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
		registrar.MustRegister(prometheus.NewGoCollector())
	}

	for _, c := range gauges {
		registrar.MustRegister(c)
	}

	return &Registrar{
		gauges: gauges,
		logger: logger,
	}
}

// Set will set the given value on the gauge metric with the given name. If
// the gauge metric is not found the process will exit with a status code of
// 1.
func (h *Registrar) Set(name string, value float64) {
	g, ok := h.gauges[name]
	if !ok {
		h.logger.Info(fmt.Sprintf("Ingore Set() for unknown health metric %s", name))
	} else {
		g.Set(value)
	}
}

// Inc will increment the gauge metric with the given name by 1. If the gauge
// metric is not found the process will exit with a status code of 1.
func (h *Registrar) Inc(name string) {
	g, ok := h.gauges[name]
	if !ok {
		h.logger.Info(fmt.Sprintf("Ingore Inc() for unknown health metric %s", name))
	} else {
		g.Inc()
	}
}

// Dec will decrement the gauge metric with the given name by 1. If the gauge
// metric is not found the process will exit with a status code of 1.
func (h *Registrar) Dec(name string) {
	g, ok := h.gauges[name]
	if !ok {
		h.logger.Info(fmt.Sprintf("Ingore Dec() for unknown health metric %s", name))
	} else {
		g.Dec()
	}
}
