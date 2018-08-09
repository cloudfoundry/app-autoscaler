package operator_test

import (
	"errors"
	"time"

	"autoscaler/metricscollector/fakes"
	"autoscaler/operator"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("InstanceMetricsDB Prune", func() {
	var (
		instanceMetricsDb       *fakes.FakeInstanceMetricsDB
		fclock                  *fakeclock.FakeClock
		cutoffDays              int
		buffer                  *gbytes.Buffer
		instanceMetricsDbPruner *operator.InstanceMetricsDbPruner
	)

	BeforeEach(func() {

		cutoffDays = 20
		logger := lagertest.NewTestLogger("prune-test")
		buffer = logger.Buffer()

		instanceMetricsDb = &fakes.FakeInstanceMetricsDB{}
		fclock = fakeclock.NewFakeClock(time.Now())

		instanceMetricsDbPruner = operator.NewInstanceMetricsDbPruner(instanceMetricsDb, cutoffDays, fclock, logger)

	})

	Describe("Prune", func() {
		JustBeforeEach(func() {
			instanceMetricsDbPruner.Operate()
		})

		Context("when pruning metrics records from instancemetrics db", func() {
			It("prunes as per given cutoff days", func() {
				Eventually(instanceMetricsDb.PruneInstanceMetricsCallCount).Should(Equal(1))
				Expect(instanceMetricsDb.PruneInstanceMetricsArgsForCall(0)).To(BeNumerically("==", fclock.Now().AddDate(0, 0, -cutoffDays).UnixNano()))
			})
		})

		Context("when pruning records from instancemetrics db fails", func() {
			BeforeEach(func() {
				instanceMetricsDb.PruneInstanceMetricsReturns(errors.New("test operator. error"))
			})

			It("should error", func() {
				Eventually(instanceMetricsDb.PruneInstanceMetricsCallCount).Should(Equal(1))
				Eventually(buffer).Should(gbytes.Say("test operator. error"))
			})
		})
	})
})
