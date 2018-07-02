package aggregator_test

import (
	. "autoscaler/eventgenerator/aggregator"
	"autoscaler/eventgenerator/aggregator/fakes"
	"autoscaler/models"
	"errors"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		         "breach_duration_secs":300,
		         "threshold":30,
		         "operator":"<",
		         "cool_down_secs":300,
		         "adjustment":"-1"
		      }
		   ]
		}`
		nodeNum   int
		nodeIndex int
	)

	BeforeEach(func() {
		database = &fakes.FakePolicyDB{}
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("PolicyPoller-test")
		nodeNum = 1
		nodeIndex = 0
	})
	Context("Start", func() {
		JustBeforeEach(func() {
			poller = NewPolicyPoller(logger, clock, testPolicyPollerInterval, nodeNum, nodeIndex, database)
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
			It("should retrieve and get policies successfully for every interval", func() {
				Eventually(database.RetrievePoliciesCallCount).Should(Equal(1))
				clock.Increment(1 * testPolicyPollerInterval)
				Eventually(database.RetrievePoliciesCallCount).Should(Equal(2))
				clock.Increment(1 * testPolicyPollerInterval)
				Eventually(database.RetrievePoliciesCallCount).Should(Equal(3))
				Eventually(poller.GetPolicies).Should(Equal(map[string]*models.AppPolicy{
					testAppId: &models.AppPolicy{
						AppId: testAppId,
						ScalingPolicy: &models.ScalingPolicy{
							InstanceMax: 5,
							InstanceMin: 1,
							ScalingRules: []*models.ScalingRule{{
								MetricType:            "test-metric-name",
								BreachDurationSeconds: 300,
								CoolDownSeconds:       300,
								Threshold:             30,
								Operator:              "<",
								Adjustment:            "-1",
							}}}}},
				))
			})

			Context("when running with 3 nodes", func() {
				BeforeEach(func() {
					nodeNum = 3
					var i int
					database.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						i++
						switch i {
						case 1:
							return []*models.PolicyJson{
								{AppId: "app-id-1", PolicyStr: policyStr},
								{AppId: "app-id-2", PolicyStr: policyStr},
								{AppId: "app-id-3", PolicyStr: policyStr},
								{AppId: "app-id-4", PolicyStr: policyStr},
							}, nil
						case 2:
							return []*models.PolicyJson{
								{AppId: "app-id-3", PolicyStr: policyStr},
								{AppId: "app-id-4", PolicyStr: policyStr},
								{AppId: "app-id-5", PolicyStr: policyStr},
								{AppId: "app-id-6", PolicyStr: policyStr},
							}, nil
						case 3:
							return []*models.PolicyJson{
								{AppId: "app-id-5", PolicyStr: policyStr},
								{AppId: "app-id-6", PolicyStr: policyStr},
								{AppId: "app-id-7", PolicyStr: policyStr},
								{AppId: "app-id-8", PolicyStr: policyStr},
							}, nil

						}
						return []*models.PolicyJson{}, nil
					}

				})
				Context("when current index is 0", func() {
					BeforeEach(func() {
						nodeIndex = 0
					})

					It("retrieves app shard 0", func() {

						Eventually(poller.GetPolicies).Should(HaveLen(2))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-1"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-2"))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-3"))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-4"))

						clock.Increment(1 * testPolicyPollerInterval)
						Consistently(poller.GetPolicies).Should(HaveLen(2))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-3"))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-4"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-5"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-6"))

						clock.Increment(1 * testPolicyPollerInterval)
						Eventually(poller.GetPolicies).Should(HaveLen(1))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-8"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-5"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-6"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-7"))

					})
				})
				Context("when current index is 1", func() {
					BeforeEach(func() {
						nodeIndex = 1
					})

					It("retrieves app shard 1", func() {
						Consistently(poller.GetPolicies).Should(BeEmpty())

						clock.Increment(1 * testPolicyPollerInterval)
						Eventually(poller.GetPolicies).Should(HaveLen(2))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-3"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-4"))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-5"))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-6"))

						clock.Increment(1 * testPolicyPollerInterval)
						Consistently(poller.GetPolicies).Should(HaveLen(2))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-5"))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-6"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-7"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-8"))
					})
				})

				Context("when current index is 2", func() {
					BeforeEach(func() {
						nodeIndex = 2
					})

					It("retrieves app shard 2", func() {
						Eventually(poller.GetPolicies).Should(HaveLen(2))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-1"))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-2"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-3"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-4"))

						clock.Increment(1 * testPolicyPollerInterval)
						Eventually(poller.GetPolicies).Should(BeEmpty())

						clock.Increment(1 * testPolicyPollerInterval)
						Eventually(poller.GetPolicies).Should(HaveLen(1))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-5"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-6"))
						Consistently(poller.GetPolicies).Should(HaveKey("app-id-7"))
						Consistently(poller.GetPolicies).ShouldNot(HaveKey("app-id-8"))
					})
				})

			})

			Context("when retrieving policies from database fails", func() {
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
			poller = NewPolicyPoller(logger, clock, testPolicyPollerInterval, nodeNum, nodeIndex, database)
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
