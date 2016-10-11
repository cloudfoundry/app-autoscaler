package pruner_test

import (
	"time"

	"autoscaler/pruner"
	"autoscaler/pruner/fakes"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("DbPruner", func() {
	var (
		proc         ifrit.Process
		fclock       *fakeclock.FakeClock
		buffer       *gbytes.Buffer
		fakeDbPruner *fakes.FakeDbPruner
		prunerRunner *pruner.DbPrunerRunner
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("pruner-test")
		buffer = logger.Buffer()
		fclock = fakeclock.NewFakeClock(time.Now())

		fakeDbPruner = &fakes.FakeDbPruner{}
		prunerRunner = pruner.NewDbPrunerRunner(fakeDbPruner, TestRefreshInterval, fclock, logger)

	})

	JustBeforeEach(func() {
		proc = ifrit.Invoke(prunerRunner)
		Eventually(buffer).Should(gbytes.Say("started"))
	})

	AfterEach(func() {
		ginkgomon.Kill(proc)
		Eventually(proc.Wait()).Should(Receive(BeNil()))
	})

	Context("when pruning", func() {
		It("prunes after given interval", func() {
			Eventually(fakeDbPruner.PruneCallCount).Should(Equal(1))

			fclock.Increment(TestRefreshInterval)
			Eventually(fakeDbPruner.PruneCallCount).Should(Equal(2))

			fclock.Increment(TestRefreshInterval)
			Eventually(fakeDbPruner.PruneCallCount).Should(Equal(3))
		})
	})

	Context("when an interrupt is sent", func() {
		It("should stop", func() {
			fclock.Increment(TestRefreshInterval)
			Eventually(fakeDbPruner.PruneCallCount).Should(Equal(2))

			ginkgomon.Kill(proc)
			Eventually(proc.Wait()).Should(Receive(BeNil()))

			Eventually(buffer).Should(gbytes.Say("stopped"))

			fclock.Increment(TestRefreshInterval)
			Consistently(fakeDbPruner.PruneCallCount).Should(Equal(2))
		})
	})
})
