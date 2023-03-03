package operator_test

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3/lagertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon_v2"
)

var _ = Describe("Operator", func() {
	var (
		proc           ifrit.Process
		fclock         *fakeclock.FakeClock
		buffer         *gbytes.Buffer
		fakeOperator   *fakes.FakeOperator
		operatorRunner *operator.OperatorRunner
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("pruner-test")
		buffer = logger.Buffer()
		fclock = fakeclock.NewFakeClock(time.Now())

		fakeOperator = &fakes.FakeOperator{}
		operatorRunner = operator.NewOperatorRunner(fakeOperator, TestRefreshInterval, fclock, logger)

	})

	JustBeforeEach(func() {
		proc = ifrit.Invoke(operatorRunner)
		Eventually(buffer).Should(gbytes.Say("started"))
	})

	AfterEach(func() {
		ginkgomon_v2.Kill(proc)
		Eventually(proc.Wait()).Should(Receive(BeNil()))
	})

	Context("when pruning", func() {
		It("prunes after given interval", func() {
			Eventually(fakeOperator.OperateCallCount).Should(Equal(1))

			fclock.Increment(TestRefreshInterval)
			Eventually(fakeOperator.OperateCallCount).Should(Equal(2))

			fclock.Increment(TestRefreshInterval)
			Eventually(fakeOperator.OperateCallCount).Should(Equal(3))
		})
	})

	Context("when an interrupt is sent", func() {
		It("should stop", func() {
			fclock.Increment(TestRefreshInterval)
			Eventually(fakeOperator.OperateCallCount).Should(Equal(2))

			ginkgomon_v2.Kill(proc)
			Eventually(proc.Wait()).Should(Receive(BeNil()))

			Eventually(buffer).Should(gbytes.Say("stopped"))

			fclock.Increment(TestRefreshInterval)
			Consistently(fakeOperator.OperateCallCount).Should(Equal(2))
		})
	})
})
