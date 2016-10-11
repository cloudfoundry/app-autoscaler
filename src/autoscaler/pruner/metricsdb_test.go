package pruner_test

import (
	"errors"
	"time"

	"autoscaler/metricscollector/fakes"
	"autoscaler/pruner"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("MetricdsDB Prune", func() {
	var (
		metricsDb       *fakes.FakeMetricsDB
		fclock          *fakeclock.FakeClock
		cutoffDays      int
		buffer          *gbytes.Buffer
		metricsDbPruner *pruner.MetricsDbPruner
	)

	BeforeEach(func() {

		cutoffDays = 20
		logger := lagertest.NewTestLogger("prune-test")
		buffer = logger.Buffer()

		metricsDb = &fakes.FakeMetricsDB{}
		fclock = fakeclock.NewFakeClock(time.Now())

		metricsDbPruner = pruner.NewMetricsDbPruner(metricsDb, cutoffDays, fclock, logger)

	})

	Describe("Prune", func() {
		JustBeforeEach(func() {
			metricsDbPruner.Prune()
		})

		Context("when pruning metrics records from metrics db", func() {
			It("prunes as per given cutoff days", func() {
				Eventually(metricsDb.PruneMetricsCallCount).Should(Equal(1))
				Expect(metricsDb.PruneMetricsArgsForCall(0)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))
			})
		})

		Context("when pruning records from metrics db fails", func() {
			BeforeEach(func() {
				metricsDb.PruneMetricsReturns(errors.New("test pruner error"))
			})

			It("should error", func() {
				Eventually(metricsDb.PruneMetricsCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("test pruner error"))
			})
		})
	})
})
