package pruner_test

import (
	"errors"
	"time"

	"autoscaler/metricscollector/fakes"
	. "autoscaler/pruner"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("MetricdsDB Prune", func() {
	var (
		metricsDB  *fakes.FakeMetricsDB
		pruner     *MetricsDBPruner
		fclock     *fakeclock.FakeClock
		cutoffDays int
		buffer     *gbytes.Buffer
	)

	BeforeEach(func() {

		cutoffDays = 20
		logger := lagertest.NewTestLogger("prune-test")
		buffer = logger.Buffer()

		metricsDB = &fakes.FakeMetricsDB{}
		fclock = fakeclock.NewFakeClock(time.Now())

		pruner = NewMetricsDBPruner(logger, metricsDB, TestIntervalInHours, cutoffDays, fclock)
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			pruner.Start()
		})

		AfterEach(func() {
			pruner.Stop()
		})

		Context("when pruning metrics records from metrics db", func() {
			It("prunes at given interval and cutoff days", func() {
				Eventually(metricsDB.PruneMetricsCallCount).Should(Equal(1))
				Expect(metricsDB.PruneMetricsArgsForCall(0)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))

				fclock.Increment(TestRefreshInterval)
				Eventually(metricsDB.PruneMetricsCallCount).Should(Equal(2))
				Expect(metricsDB.PruneMetricsArgsForCall(1)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))

				fclock.Increment(TestRefreshInterval)
				Eventually(metricsDB.PruneMetricsCallCount).Should(Equal(3))
				Expect(metricsDB.PruneMetricsArgsForCall(2)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))
			})
		})

		Context("when pruning records from metrics db fails", func() {
			BeforeEach(func() {
				metricsDB.PruneMetricsReturns(errors.New("test pruner error"))
			})

			It("should error", func() {
				Eventually(metricsDB.PruneMetricsCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("test pruner error"))

				fclock.Increment(TestRefreshInterval)
				Eventually(metricsDB.PruneMetricsCallCount).Should(Equal(2))
				Eventually(buffer).Should(gbytes.Say("test pruner error"))
			})
		})
	})

	Describe("Stop", func() {
		JustBeforeEach(func() {
			pruner.Start()
		})

		It("Stop the pruner", func() {
			fclock.Increment(TestRefreshInterval)
			Eventually(metricsDB.PruneMetricsCallCount).Should(Equal(2))

			pruner.Stop()
			Eventually(buffer).Should(gbytes.Say("metrics-db-pruner-stopped"))

			fclock.Increment(TestRefreshInterval)
			Consistently(metricsDB.PruneMetricsCallCount).Should(Equal(2))
		})
	})
})
