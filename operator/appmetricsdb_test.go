package operator_test

import (
	"errors"
	"time"

	"autoscaler/fakes"
	"autoscaler/operator"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("AppMetricsDB Prune", func() {
	var (
		appMetricsDb       *fakes.FakeAppMetricDB
		fclock             *fakeclock.FakeClock
		cutoffDuration     time.Duration
		buffer             *gbytes.Buffer
		appMetricsDbPruner *operator.AppMetricsDbPruner
	)

	BeforeEach(func() {

		cutoffDuration = 20 * time.Hour
		logger := lagertest.NewTestLogger("prune-test")
		buffer = logger.Buffer()

		appMetricsDb = &fakes.FakeAppMetricDB{}
		fclock = fakeclock.NewFakeClock(time.Now())

		appMetricsDbPruner = operator.NewAppMetricsDbPruner(appMetricsDb, cutoffDuration, fclock, logger)

	})

	Describe("Prune", func() {
		JustBeforeEach(func() {
			appMetricsDbPruner.Operate()
		})

		Context("when pruning metrics records from app metrics db", func() {
			It("prunes as per given cutoff days", func() {
				Eventually(appMetricsDb.PruneAppMetricsCallCount).Should(Equal(1))
				Expect(appMetricsDb.PruneAppMetricsArgsForCall(0)).To(BeNumerically("==", fclock.Now().Add(-cutoffDuration).UnixNano()))
			})
		})

		Context("when pruning records from app metrics db fails", func() {
			BeforeEach(func() {
				appMetricsDb.PruneAppMetricsReturns(errors.New("test pruner error"))
			})

			It("should error", func() {
				Eventually(appMetricsDb.PruneAppMetricsCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("test pruner error"))
			})
		})
	})
})
