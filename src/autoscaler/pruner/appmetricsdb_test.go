package pruner_test

import (
	"errors"
	"time"

	"autoscaler/eventgenerator/aggregator/fakes"
	. "autoscaler/pruner"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Prune", func() {
	var (
		appMetricsDb *fakes.FakeAppMetricDB
		prunerRunner *DbPrunerRunner
		fclock       *fakeclock.FakeClock
		cutoffDays   int
		buffer       *gbytes.Buffer
	)

	BeforeEach(func() {

		cutoffDays = 20
		logger := lagertest.NewTestLogger("prune-test")
		buffer = logger.Buffer()

		appMetricsDb = &fakes.FakeAppMetricDB{}
		fclock = fakeclock.NewFakeClock(time.Now())

		appMetricsDbPruner := NewAppMetricsDbPruner(appMetricsDb, cutoffDays, fclock, logger)
		prunerRunner = NewDbPrunerRunner(appMetricsDbPruner, "appmetrics-db", TestRefreshInterval, fclock, logger)

	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			prunerRunner.Start()
		})

		AfterEach(func() {
			prunerRunner.Stop()
		})

		Context("when pruning metrics records from app metrics db", func() {
			It("prunes at given interval and cutoff days", func() {
				Eventually(appMetricsDb.PruneAppMetricsCallCount).Should(Equal(1))
				Expect(appMetricsDb.PruneAppMetricsArgsForCall(0)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))

				fclock.Increment(TestRefreshInterval)
				Eventually(appMetricsDb.PruneAppMetricsCallCount).Should(Equal(2))
				Expect(appMetricsDb.PruneAppMetricsArgsForCall(1)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))

				fclock.Increment(TestRefreshInterval)
				Eventually(appMetricsDb.PruneAppMetricsCallCount).Should(Equal(3))
				Expect(appMetricsDb.PruneAppMetricsArgsForCall(2)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))
			})
		})

		Context("when pruning records from app metrics db fails", func() {
			BeforeEach(func() {
				appMetricsDb.PruneAppMetricsReturns(errors.New("test pruner error"))
			})

			It("should error", func() {
				Eventually(appMetricsDb.PruneAppMetricsCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("test pruner error"))

				fclock.Increment(TestRefreshInterval)
				Eventually(appMetricsDb.PruneAppMetricsCallCount).Should(Equal(2))
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
			Eventually(appMetricsDb.PruneAppMetricsCallCount).Should(Equal(2))

			prunerRunner.Stop()
			Eventually(buffer).Should(gbytes.Say("appmetrics-db-pruner-stopped"))

			fclock.Increment(TestRefreshInterval)
			Eventually(appMetricsDb.PruneAppMetricsCallCount).Should(Equal(2))
		})
	})
})
