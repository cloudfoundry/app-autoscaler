package manager_test

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/manager"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
)

var _ = Describe("PolicyManager", func() {
	var (
		database                 *fakes.FakePolicyDB
		clock                    *fakeclock.FakeClock
		policyManager            *PolicyManager
		testPolicyPollerInterval time.Duration
		allowedMetricCache       cache.Cache
		allowedMetricTypeSet     map[string]struct{}
		policyMap                map[string]*models.AppPolicy
		logger                   lager.Logger
		scalingPolicy            *models.ScalingPolicy
		appPolicy                *models.AppPolicy
		testAppId                = "testAppId"
		policyStr                = `
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
	)

	BeforeEach(func() {
		database = &fakes.FakePolicyDB{}
		clock = fakeclock.NewFakeClock(time.Now())
		testPolicyPollerInterval = 1 * time.Second
		allowedMetricCache = *cache.New(10*time.Minute, -1)
		logger = lager.NewLogger("policyManager-test")
		policyMap = make(map[string]*models.AppPolicy)
		allowedMetricTypeSet = make(map[string]struct{})
		allowedMetricTypeSet["queuelength"] = struct{}{}
	})
	Context("Start", func() {
		JustBeforeEach(func() {
			policyManager = NewPolicyManager(logger, clock, testPolicyPollerInterval, database, allowedMetricCache, 10*time.Minute)
			policyManager.Start()

		})

		AfterEach(func() {
			policyManager.Stop()
		})

		Context("when the policyManager is started", func() {
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
			})
		})
	})

	Context("Refresh AllowedMetric Cache", func() {
		JustBeforeEach(func() {
			policyManager = NewPolicyManager(logger, clock, testPolicyPollerInterval, database, allowedMetricCache, 10*time.Minute)
			policyManager.Start()
		})

		AfterEach(func() {
			policyManager.Stop()
		})

		When("allowedMetricCache has already filled with metricstype details of the same appilication", func() {

			BeforeEach(func() {
				scalingPolicy = &models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 6,
					ScalingRules: []*models.ScalingRule{{
						MetricType:            "queuelength",
						BreachDurationSeconds: 60,
						Threshold:             10,
						Operator:              ">",
						CoolDownSeconds:       60,
						Adjustment:            "+1"}}}
				appPolicy = &models.AppPolicy{AppId: testAppId, ScalingPolicy: scalingPolicy}
				policyMap[testAppId] = appPolicy
				allowedMetricCache.Set(testAppId, allowedMetricTypeSet, 10*time.Minute)

				res, found := allowedMetricCache.Get(testAppId)
				maps := res.(map[string]struct{})
				Expect(found).To(BeTrue())
				Expect(maps).Should(HaveKey("queuelength"))

			})

			When("the policy is updated", func() {
				BeforeEach(func() {
					database.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						return []*models.PolicyJson{{AppId: testAppId, PolicyStr: policyStr}}, nil
					}
				})
				It("should refresh the allowed metrics cache", func() {
					Eventually(database.RetrievePoliciesCallCount).Should(Equal(1))
					clock.Increment(1 * testPolicyPollerInterval)
					Eventually(database.RetrievePoliciesCallCount).Should(Equal(2))

					res, found := allowedMetricCache.Get(testAppId)
					maps := res.(map[string]struct{})

					Expect(found).To(BeTrue())
					Expect(maps).Should(HaveKey("test-metric-name"))
					Expect(maps).ShouldNot(HaveKey("queuelength"))
				})
			})
			When("the policy is deleted", func() {
				BeforeEach(func() {
					database.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						return []*models.PolicyJson{}, nil
					}
				})
				It("should refresh the allowed metrics cache", func() {
					Eventually(database.RetrievePoliciesCallCount).Should(Equal(1))
					clock.Increment(1 * testPolicyPollerInterval)
					Eventually(database.RetrievePoliciesCallCount).Should(Equal(2))

					_, found := allowedMetricCache.Get(testAppId)
					Expect(found).To(BeFalse())
				})
			})
		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			policyManager = NewPolicyManager(logger, clock, testPolicyPollerInterval, database, allowedMetricCache, 10*time.Minute)
			policyManager.Start()
			Eventually(database.RetrievePoliciesCallCount).Should(Equal(1))

			policyManager.Stop()
		})

		It("stops the polling", func() {
			clock.Increment(5 * testPolicyPollerInterval)
			Consistently(database.RetrievePoliciesCallCount).Should(Or(Equal(1), Equal(2)))
		})
	})
})
