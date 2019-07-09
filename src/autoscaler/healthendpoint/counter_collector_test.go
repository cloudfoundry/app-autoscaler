package healthendpoint_test

import (
	. "autoscaler/healthendpoint"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("CounterCollector", func() {
	var (
		namespace1 string = "test_name_space1"
		subSystem1 string = "test_sub_system1"
		name1      string = "test_name1"
		help1      string = "test_help1"

		namespace2 string = "test_name_space2"
		subSystem2 string = "test_sub_system2"
		name2      string = "test_name2"
		help2      string = "test_help2"

		counterOpt1 = prometheus.CounterOpts{
			Namespace: namespace1,
			Subsystem: subSystem1,
			Name:      name1,
			Help:      help1,
		}
		counterOpt2 = prometheus.CounterOpts{
			Namespace: namespace2,
			Subsystem: subSystem2,
			Name:      name2,
			Help:      help2,
		}

		descChan         chan *prometheus.Desc
		metricChan       chan prometheus.Metric
		counterCollector CounterCollector
		counterDesc1     = prometheus.NewDesc(
			prometheus.BuildFQName(namespace1, subSystem1, name1),
			help1,
			nil,
			nil,
		)
		counterDesc2 = prometheus.NewDesc(
			prometheus.BuildFQName(namespace2, subSystem2, name2),
			help2,
			nil,
			nil,
		)
		counterMetric1 = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace1,
				Subsystem: subSystem1,
				Name:      name1,
				Help:      help1,
			})
		counterMetric2 = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace2,
				Subsystem: subSystem2,
				Name:      name2,
				Help:      help2,
			})
	)
	BeforeEach(func() {
		descChan = make(chan *prometheus.Desc, 10)
		metricChan = make(chan prometheus.Metric, 10)
		counterCollector = NewCounterCollector()
		counterCollector.AddCounters(counterOpt1, counterOpt2)
	})
	Context("Describe", func() {
		BeforeEach(func() {
			counterCollector.Describe(descChan)
		})
		It("receive descriptions", func() {
			var desc1, desc2 *prometheus.Desc
			Expect(descChan).To(Receive(&desc1))
			Expect(descChan).To(Receive(&desc2))
			Expect([]prometheus.Desc{*desc1, *desc2}).To(ContainElement(*counterDesc1))
			Expect([]prometheus.Desc{*desc1, *desc2}).To(ContainElement(*counterDesc2))
		})
	})
	Context("Collect", func() {
		BeforeEach(func() {
			counterCollector.Add(counterOpt1, 10)
			counterCollector.Add(counterOpt2, 100)
			counterCollector.Collect(metricChan)
			counterMetric1.Add(float64(10))
			counterMetric2.Add(float64(100))
		})
		It("Receive metrics", func() {
			var metric1, metric2 prometheus.Metric
			Expect(metricChan).To(Receive(&metric1))
			Expect(metricChan).To(Receive(&metric2))
			Expect([]prometheus.Metric{metric1, metric2}).To(ContainElement(prometheus.Metric(counterMetric1)))
			Expect([]prometheus.Metric{metric1, metric2}).To(ContainElement(prometheus.Metric(counterMetric2)))
		})
	})

})
