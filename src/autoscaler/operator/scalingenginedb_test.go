package operator_test

import (
	"errors"
	"time"

	"autoscaler/operator"
	"autoscaler/scalingengine/fakes"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("ScalingEngineDbPruner", func() {
	var (
		scalingEngineDB       *fakes.FakeScalingEngineDB
		fclock                *fakeclock.FakeClock
		cutoffDays            int
		buffer                *gbytes.Buffer
		scalingEngineDbPruner *operator.ScalingEngineDbPruner
	)

	BeforeEach(func() {
		cutoffDays = 20
		logger := lagertest.NewTestLogger("pruner-test")
		buffer = logger.Buffer()
		scalingEngineDB = &fakes.FakeScalingEngineDB{}
		fclock = fakeclock.NewFakeClock(time.Now())
		scalingEngineDbPruner = operator.NewScalingEngineDbPruner(scalingEngineDB, cutoffDays, fclock, logger)
	})

	Describe("Prune", func() {
		JustBeforeEach(func() {
			scalingEngineDbPruner.Operate()
		})

		Context("when pruning records from scalinghistory table", func() {
			It("prunes as per given cutoff days", func() {
				Eventually(scalingEngineDB.PruneScalingHistoriesCallCount).Should(Equal(1))
				Expect(scalingEngineDB.PruneScalingHistoriesArgsForCall(0)).To(Equal(fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))
			})
		})

		Context("when pruning records from scalinghistory table fails", func() {
			BeforeEach(func() {
				scalingEngineDB.PruneScalingHistoriesReturns(errors.New("test error"))
			})

			It("should error", func() {
				Eventually(scalingEngineDB.PruneScalingHistoriesCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("test error"))
			})
		})
	})
})
