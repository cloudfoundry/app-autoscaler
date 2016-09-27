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
		metricsDb    *fakes.FakeMetricsDB
		prunerRunner *DbPrunerRunner
		fclock       *fakeclock.FakeClock
		cutoffDays   int
		buffer       *gbytes.Buffer
	)

	BeforeEach(func() {

		cutoffDays = 20
		logger := lagertest.NewTestLogger("prune-test")
		buffer = logger.Buffer()

		metricsDb = &fakes.FakeMetricsDB{}
		fclock = fakeclock.NewFakeClock(time.Now())

		metricsDbPruner := NewMetricsDbPruner(metricsDb, cutoffDays, fclock, logger)
		prunerRunner = NewDbPrunerRunner(metricsDbPruner, "metrics-db", TestRefreshInterval, fclock, logger)

	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			prunerRunner.Start()
		})

		AfterEach(func() {
			prunerRunner.Stop()
		})

		Context("when pruning metrics records from metrics db", func() {
			It("prunes at given interval and cutoff days", func() {
				Eventually(metricsDb.PruneMetricsCallCount).Should(Equal(1))
				Expect(metricsDb.PruneMetricsArgsForCall(0)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))

				fclock.Increment(TestRefreshInterval)
				Eventually(metricsDb.PruneMetricsCallCount).Should(Equal(2))
				Expect(metricsDb.PruneMetricsArgsForCall(1)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))

				fclock.Increment(TestRefreshInterval)
				Eventually(metricsDb.PruneMetricsCallCount).Should(Equal(3))
				Expect(metricsDb.PruneMetricsArgsForCall(2)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))
			})
		})

		Context("when pruning records from metrics db fails", func() {
			BeforeEach(func() {
				metricsDb.PruneMetricsReturns(errors.New("test pruner error"))
			})

			It("should error", func() {
				Eventually(metricsDb.PruneMetricsCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("test pruner error"))

				fclock.Increment(TestRefreshInterval)
				Eventually(metricsDb.PruneMetricsCallCount).Should(Equal(2))
				Eventually(buffer).Should(gbytes.Say("test pruner error"))
			})
		})
	})

	Describe("Stop", func() {
		JustBeforeEach(func() {
			prunerRunner.Start()
			Eventually(fclock.WatcherCount).Should(Equal(1))
		})

		It("Stops the pruner", func() {
			fclock.Increment(TestRefreshInterval)
			Eventually(metricsDb.PruneMetricsCallCount).Should(Equal(2))

			prunerRunner.Stop()
			Eventually(buffer).Should(gbytes.Say("metrics-db-pruner-stopped"))

			fclock.Increment(TestRefreshInterval)
			Consistently(metricsDb.PruneMetricsCallCount).Should(Equal(2))
		})
	})
})
