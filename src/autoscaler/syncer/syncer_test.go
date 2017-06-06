package syncer_test

import (
	"time"

	"autoscaler/syncer"
	"autoscaler/syncer/fakes"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("Syncer", func() {
	var (
		proc         ifrit.Process
		fclock       *fakeclock.FakeClock
		buffer       *gbytes.Buffer
		fakeSyncer   *fakes.FakeSyncer
		syncerRunner *syncer.SyncerRunner
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("syncer-test")
		buffer = logger.Buffer()
		fclock = fakeclock.NewFakeClock(time.Now())

		fakeSyncer = &fakes.FakeSyncer{}
		syncerRunner = syncer.NewSyncerRunner(fakeSyncer, TestSynchronizeInterval, fclock, logger)

	})

	JustBeforeEach(func() {
		proc = ifrit.Invoke(syncerRunner)
		Eventually(buffer).Should(gbytes.Say("started"))
	})

	AfterEach(func() {
		ginkgomon.Kill(proc)
		Eventually(proc.Wait()).Should(Receive(BeNil()))
	})

	Context("when synchronizing", func() {
		It("synchronizes after synchronize interval", func() {
			Eventually(fakeSyncer.SynchronizeCallCount).Should(Equal(1))

			fclock.Increment(TestSynchronizeInterval)
			Eventually(fakeSyncer.SynchronizeCallCount).Should(Equal(2))

			fclock.Increment(TestSynchronizeInterval)
			Eventually(fakeSyncer.SynchronizeCallCount).Should(Equal(3))
		})
	})

	Context("when an interrupt is sent", func() {
		It("should stop", func() {
			fclock.Increment(TestSynchronizeInterval)
			Eventually(fakeSyncer.SynchronizeCallCount).Should(Equal(2))

			ginkgomon.Kill(proc)
			Eventually(proc.Wait()).Should(Receive(BeNil()))

			Eventually(buffer).Should(gbytes.Say("stopped"))

			fclock.Increment(TestSynchronizeInterval)
			Consistently(fakeSyncer.SynchronizeCallCount).Should(Equal(2))
		})
	})

})
