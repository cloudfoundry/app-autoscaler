package aggregator_test

import (
	. "autoscaler/eventgenerator/aggregator"
	"autoscaler/eventgenerator/aggregator/fakes"
	"autoscaler/models"
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("PolicyPoller", func() {
	var (
		database  *fakes.FakePolicyDB
		clock     *fakeclock.FakeClock
		poller    *PolicyPoller
		logger    lager.Logger
		testAppId = "testAppId"
		policyStr = `
		{
		   "instance_min_count":1,
		   "instance_max_count":5,
		   "scaling_rules":[
		      {
		         "metric_type":"test-metric-name",
		         "stat_window_secs":300,
		         "breach_duration_secs":300,
		         "threshold":30,
		         "operator":"<",
		         "cool_down_secs":300,
		         "adjustment":"-1"
		      }
		   ]
		}`
	)

	BeforeEach(func() {
		database = &fakes.FakePolicyDB{}
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("PolicyPoller-test")

	})
	Context("Start", func() {
		JustBeforeEach(func() {
			poller = NewPolicyPoller(logger, clock, testPolicyPollerInterval, database)
			poller.Start()

		})

		AfterEach(func() {
			poller.Stop()
		})

		Context("when the poller is started", func() {
			BeforeEach(func() {
				database.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
					return []*models.PolicyJson{{AppId: testAppId, PolicyStr: policyStr}}, nil
				}

			})
			It("should retrieve policies for every interval", func() {
				Eventually(database.RetrievePoliciesCallCount).Should(Equal(1))
				clock.Increment(2 * testPolicyPollerInterval * time.Second)
				Eventually(database.RetrievePoliciesCallCount).Should(BeNumerically(">=", 2))
			})

			Context("when retrieve policies and compute triggers successfully", func() {
				BeforeEach(func() {
					database.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						return []*models.PolicyJson{{AppId: testAppId, PolicyStr: policyStr}}, nil
					}
				})
				It("should call the consumer with the new triggers for every interval", func() {
					Eventually(clock.WatcherCount).Should(Equal(1))
					clock.Increment(1 * testPolicyPollerInterval)
					clock.Increment(1 * testPolicyPollerInterval)
					Eventually(database.RetrievePoliciesCallCount).Should(BeNumerically(">=", 2))
					policyMap := poller.GetPolicies()
					Expect(policyMap[testAppId]).To(Equal(&models.AppPolicy{
						AppId: testAppId,
						ScalingPolicy: &models.ScalingPolicy{
							InstanceMax: 5,
							InstanceMin: 1,
							ScalingRules: []*models.ScalingRule{{
								MetricType:            "test-metric-name",
								StatWindowSeconds:     300,
								BreachDurationSeconds: 300,
								CoolDownSeconds:       300,
								Threshold:             30,
								Operator:              "<",
								Adjustment:            "-1",
							}}},
					}))
				})
			})
			Context("when return error when retrieve policies from database", func() {
				BeforeEach(func() {
					database.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						return nil, errors.New("error when retrieve policies from database")
					}
				})
				It("should not call the consumer as there is no trigger", func() {
					clock.Increment(2 * testPolicyPollerInterval)
					policyMap := poller.GetPolicies()
					Expect(len(policyMap)).To(Equal(0))
				})
			})
		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			poller = NewPolicyPoller(logger, clock, testPolicyPollerInterval, database)
			poller.Start()
			Eventually(database.RetrievePoliciesCallCount).Should(Equal(1))

			poller.Stop()
		})

		It("stops the polling", func() {
			clock.Increment(5 * testPolicyPollerInterval)
			Consistently(database.RetrievePoliciesCallCount).Should(Or(Equal(1), Equal(2)))
		})
	})
})
