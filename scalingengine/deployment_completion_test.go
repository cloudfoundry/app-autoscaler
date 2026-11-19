package scalingengine_test

import (
	"context"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Deployment completion worker lifecycle", func() {
	It("checks at startup and periodically and synchronizes fresh schedules", func() {
		logger := lagertest.NewTestLogger("completion-runner")
		clk := fakeclock.NewFakeClock(time.Now())
		engineDB := &fakes.FakeScalingEngineDB{}
		engine := NewScalingEngine(logger, &fakes.FakeCFClient{}, &fakes.FakePolicyDB{}, engineDB, clk, 300, 32)
		synchronized := make(chan struct{}, 10)
		runner := NewDeploymentCompletionRunner(logger, engine, clk, 15*time.Second, func() { synchronized <- struct{}{} })
		process := ifrit.Background(runner)
		Eventually(process.Ready()).Should(BeClosed())
		Eventually(synchronized).Should(Receive())
		Expect(engineDB.GetOpenScalingHistoriesCallCount()).To(Equal(1))
		clk.Increment(15 * time.Second)
		Eventually(synchronized).Should(Receive())
		Expect(engineDB.GetOpenScalingHistoriesCallCount()).To(Equal(2))
		process.Signal(os.Interrupt)
		Eventually(process.Wait()).Should(Receive(BeNil()))
	})

	It("cancels in-flight completion queries on shutdown", func() {
		logger := lagertest.NewTestLogger("completion-runner")
		clk := fakeclock.NewFakeClock(time.Now())
		engineDB, cfc := &fakes.FakeScalingEngineDB{}, &fakes.FakeCFClient{}
		engineDB.GetOpenScalingHistoriesReturns([]*models.AppScalingHistory{{AppId: "app", DeploymentGUID: "deployment",
			ScalingDimension: models.ScalingDimensionVertical, CorrelationID: "correlation", Status: models.ScalingStatusPending}}, nil)
		entered := make(chan struct{})
		cfc.GetDeploymentStub = func(ctx context.Context, _ string) (*cf.Deployment, error) {
			close(entered)
			<-ctx.Done()
			return nil, ctx.Err()
		}
		engine := NewScalingEngine(logger, cfc, &fakes.FakePolicyDB{}, engineDB, clk, 300, 32)
		process := ifrit.Background(NewDeploymentCompletionRunner(logger, engine, clk, 15*time.Second, nil))
		Eventually(entered).Should(BeClosed())
		process.Signal(os.Interrupt)
		Eventually(process.Wait()).Should(Receive(BeNil()))
		Expect(engineDB.CompleteScalingHistoryCallCount()).To(BeZero())
	})
})
