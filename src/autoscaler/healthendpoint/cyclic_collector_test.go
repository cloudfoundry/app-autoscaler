package healthendpoint_test

import (
	"time"

	. "autoscaler/healthendpoint"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("CyclicCollector", func() {
	var (
		cyclicCollector *CyclicCollector
		clock           *fakeclock.FakeClock
		logger          *lagertest.TestLogger
		interval        time.Duration = 5 * time.Second

		descChan   chan *prometheus.Desc
		metricChan chan prometheus.Metric
		gaugeOpts  prometheus.GaugeOpts = prometheus.GaugeOpts{
			Namespace: "autoscaler",
			Subsystem: "nozzle",
			Name:      "envelope_number_from_rlp",
			Help:      "Number of envelopes from rlp",
		}
		desc = *prometheus.NewDesc(
			prometheus.BuildFQName("autoscaler", "nozzle", "envelope_number_from_rlp"),
			"Number of envelopes from rlp",
			nil,
			nil,
		)
		metric0, _ = prometheus.NewConstMetric(&desc, prometheus.GaugeValue, float64(0))
		metric1, _ = prometheus.NewConstMetric(&desc, prometheus.GaugeValue, float64(30))
	)
	BeforeEach(func() {
		descChan = make(chan *prometheus.Desc, 10)
		metricChan = make(chan prometheus.Metric, 10)
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lagertest.NewTestLogger("CyclicCollector-test")
		cyclicCollector = NewCyclicCollector(clock, interval, logger, gaugeOpts)
	})
	Context("Describe", func() {
		BeforeEach(func() {
			cyclicCollector.Describe(descChan)
		})
		It("Receive descs", func() {
			Eventually(descChan).Should(Receive(Equal(&desc)))
		})
	})
	Context("Collect", func() {
		BeforeEach(func() {
			cyclicCollector.Start()
			Eventually(logger.Buffer, 5*time.Second).Should(Say("started"))
		})
		It("Receive metrics", func() {
			By("get zero value at first")
			cyclicCollector.Collect(metricChan)
			Eventually(metricChan).Should(Receive(Equal(metric0)))
			By("get non-zero value after Inc was called")
			cyclicCollector.Inc(30)
			clock.Increment(1 * interval)
			Eventually(logger.Buffer, 5*time.Second).Should(Say("reset"))
			cyclicCollector.Collect(metricChan)
			Eventually(metricChan).Should(Receive(Equal(metric1)))
			By("get zero value after reset")
			clock.Increment(1 * interval)
			Eventually(logger.Buffer, 5*time.Second).Should(Say("reset"))
			cyclicCollector.Collect(metricChan)
			Eventually(metricChan).Should(Receive(Equal(metric0)))

		})
		AfterEach(func() {
			cyclicCollector.Stop()
		})
	})
})
