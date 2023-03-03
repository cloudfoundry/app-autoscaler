package healthendpoint_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"

	"code.cloudfoundry.org/lager/v3"
	"github.com/prometheus/client_golang/prometheus"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health", func() {

	var (
		logger = lager.NewLogger("test")
	)

	Describe("Health UT with fake Registrar", func() {

		var (
			registrar *spyRegistrar
		)

		Context("Default collectors register", func() {
			BeforeEach(func() {
				registrar = newSpyRegistrar()
				healthendpoint.RegisterCollectors(registrar, []prometheus.Collector{
					&simpleCollector{},
					&simpleCollector{},
				}, true, logger)
			})

			It("registers the default collectors with the registrar", func() {
				Expect(registrar.collectors).To(HaveLen(4))
			})
		})

		Context("Custom collectors register", func() {
			BeforeEach(func() {
				registrar = newSpyRegistrar()
				healthendpoint.RegisterCollectors(registrar, []prometheus.Collector{
					&simpleCollector{},
					&simpleCollector{},
				}, false, logger)
			})

			Describe("Registering custom collectors", func() {
				It("registers the custom collectors with the registrar", func() {
					Expect(registrar.collectors).To(HaveLen(2))
				})
			})
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

func (s *spyRegistrar) Register(c prometheus.Collector) error {
	s.collectors = append(s.collectors, c)
	return nil
}

type simpleCollector struct{}

func (c *simpleCollector) Describe(chan<- *prometheus.Desc) {}
func (c *simpleCollector) Collect(chan<- prometheus.Metric) {}
