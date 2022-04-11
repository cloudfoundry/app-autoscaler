package healthendpoint_test

import (
	"strings"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"

	"github.com/prometheus/client_golang/prometheus/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("CounterCollector", func() {
	var (
		namespace1 = "test_name_space1"
		subSystem1 = "test_sub_system1"
		name1      = "test_name1"
		help1      = "test_help1"

		namespace2 = "test_name_space2"
		subSystem2 = "test_sub_system2"
		name2      = "test_name2"
		help2      = "test_help2"

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
			numberReceived := testutil.CollectAndCount(counterCollector, "test_name_space1_test_sub_system1_test_name1", "test_name_space2_test_sub_system2_test_name2")
			Expect(numberReceived).Should(Equal(2))

			expected := `
				# HELP test_name_space1_test_sub_system1_test_name1 test_help1
				# TYPE test_name_space1_test_sub_system1_test_name1 counter
				test_name_space1_test_sub_system1_test_name1 10
        		# HELP test_name_space2_test_sub_system2_test_name2 test_help2
        		# TYPE test_name_space2_test_sub_system2_test_name2 counter
				test_name_space2_test_sub_system2_test_name2 100
`
			err := testutil.CollectAndCompare(counterCollector, strings.NewReader(expected))
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
