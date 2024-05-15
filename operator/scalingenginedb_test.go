package operator_test

import (
	"context"
	"errors"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3/lagertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("ScalingEngineDbPruner", func() {
	var (
		scalingEngineDB       *fakes.FakeScalingEngineDB
		fclock                *fakeclock.FakeClock
		cutoffDuration        time.Duration
		buffer                *gbytes.Buffer
		scalingEngineDbPruner *operator.ScalingEngineDbPruner
	)

	BeforeEach(func() {
		cutoffDuration = 20
		logger := lagertest.NewTestLogger("pruner-test")
		buffer = logger.Buffer()
		scalingEngineDB = &fakes.FakeScalingEngineDB{}
		fclock = fakeclock.NewFakeClock(time.Now())
		scalingEngineDbPruner = operator.NewScalingEngineDbPruner(scalingEngineDB, cutoffDuration, fclock, logger)
	})

	Describe("Prune", func() {
		JustBeforeEach(func() {
			scalingEngineDbPruner.Operate(context.Background())
		})

		Context("when pruning records from scalinghistory table", func() {
			It("prunes as per given cutoff days", func() {
				Eventually(scalingEngineDB.PruneScalingHistoriesCallCount).Should(Equal(1))
				_, cutoffTime := scalingEngineDB.PruneScalingHistoriesArgsForCall(0)
				Expect(cutoffTime).To(Equal(fclock.Now().Add(-cutoffDuration).UnixNano()))
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

		Context("when pruning records from scalingcooldown table", func() {
			It("prunes expired cooldowns", func() {
				Eventually(scalingEngineDB.PruneCooldownsCallCount).Should(Equal(1))
				_, cutoffTime := scalingEngineDB.PruneCooldownsArgsForCall(0)
				Expect(cutoffTime).To(Equal(fclock.Now().UnixNano()))
			})
		})

		Context("when pruning records from scalingcooldown table fails", func() {
			BeforeEach(func() {
				scalingEngineDB.PruneCooldownsReturns(errors.New("test error"))
			})

			It("should error", func() {
				Eventually(scalingEngineDB.PruneCooldownsCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("test error"))
			})
		})

	})
})
