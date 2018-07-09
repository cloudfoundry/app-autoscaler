package healthendpoint_test

import (
	"autoscaler/healthendpoint"

	"code.cloudfoundry.org/lager"
	"github.com/prometheus/client_golang/prometheus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health", func() {

	var (
		h      *healthendpoint.Registrar
		logger lager.Logger = lager.NewLogger("test")
	)

	Describe("Health UT with fake Registrar & gauge", func() {

		var (
			registrar   *spyRegistrar
			gaugeCount1 *spyGauge
			gaugeCount2 *spyGauge
		)

		Context("Default collectors register", func() {
			BeforeEach(func() {
				gaugeCount1 = newSpyGauge()
				gaugeCount2 = newSpyGauge()
				registrar = newSpyRegistrar()
				h = healthendpoint.New(registrar, map[string]prometheus.Gauge{
					"count-1": gaugeCount1,
					"count-2": gaugeCount2,
				}, true, logger)
			})

			It("registers the default collectors with the registrar", func() {
				Expect(registrar.collectors).To(HaveLen(4))
			})
		})

		Context("Custom metric handling", func() {
			BeforeEach(func() {
				gaugeCount1 = newSpyGauge()
				gaugeCount2 = newSpyGauge()
				registrar = newSpyRegistrar()
				h = healthendpoint.New(registrar, map[string]prometheus.Gauge{
					"count-1": gaugeCount1,
					"count-2": gaugeCount2,
				}, false, logger)
			})

			Describe("Registering gauge values", func() {
				It("registers the gauges with the registrar", func() {
					Expect(registrar.collectors).To(HaveLen(2))
				})
			})

			Describe("Set()", func() {
				It("sets the value on the gauge", func() {
					h.Set("count-1", 30.0)
					Expect(gaugeCount1.value).To(Equal(30.0))

					h.Set("count-2", 60.0)
					Expect(gaugeCount2.value).To(Equal(60.0))
				})
			})

			Describe("Inc()", func() {
				It("Increments the gauge", func() {
					h.Inc("count-1")
					Expect(gaugeCount1.inc).To(Equal(1))

					h.Inc("count-2")
					Expect(gaugeCount2.inc).To(Equal(1))
				})
			})

			Describe("Dec()", func() {
				It("Decrements the gauge", func() {
					h.Dec("count-1")
					Expect(gaugeCount1.dec).To(Equal(1))

					h.Dec("count-2")
					Expect(gaugeCount2.dec).To(Equal(1))
				})
			})
		})
	})

	Describe("Recover from panic", func() {

		var (
			testGauge    prometheus.Gauge
			promRegistry *prometheus.Registry
		)

		BeforeEach(func() {
			testGauge = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace: "autoscaler",
					Subsystem: "scalingengine",
					Name:      "testGauge",
					Help:      "It is a test Gauge",
				},
			)
			promRegistry = prometheus.NewRegistry()
			h = healthendpoint.New(promRegistry, map[string]prometheus.Gauge{
				"testGauge": testGauge,
			}, true, logger)
		})

		It("It won't panic with multiple prometheus registry", func() {
			Expect(func() {
				healthendpoint.New(promRegistry, map[string]prometheus.Gauge{
					"testGauge": testGauge,
				}, true, logger)
			}).NotTo(Panic())
		})

		It("It won't panic when set value for non-exist gauge", func() {
			Expect(func() {
				h.Set("non-exist-Gauge", 0)
			}).NotTo(Panic())
		})

		It("It won't panic when inc value for non-exist gauge", func() {
			Expect(func() {
				h.Inc("non-exist-Gauge")
			}).NotTo(Panic())
		})

		It("It won't panic when dec value for non-exist gauge", func() {
			Expect(func() {
				h.Dec("non-exist-Gauge")
			}).NotTo(Panic())
		})
	})

})

type spyRegistrar struct {
	prometheus.Registerer
	collectors []prometheus.Collector
}

func newSpyRegistrar() *spyRegistrar {
	return &spyRegistrar{}
}

func (s *spyRegistrar) MustRegister(c ...prometheus.Collector) {
	s.collectors = append(s.collectors, c...)
}

type spyGauge struct {
	prometheus.Gauge
	value float64
	inc   int
	dec   int
}

func newSpyGauge() *spyGauge {
	return &spyGauge{}
}

func (s *spyGauge) Set(val float64) {
	s.value = val
}

func (s *spyGauge) Inc() {
	s.inc++
}

func (s *spyGauge) Dec() {
	s.dec++
}
