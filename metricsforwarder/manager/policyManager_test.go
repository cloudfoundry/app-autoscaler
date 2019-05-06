package manager_test

import (
	"autoscaler/fakes"
	. "autoscaler/metricsforwarder/manager"
	"autoscaler/models"
	"errors"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cache "github.com/patrickmn/go-cache"
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
				Eventually(policyManager.GetPolicies).Should(Equal(map[string]*models.AppPolicy{
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

			Context("when retrieving policies from database fails", func() {
				BeforeEach(func() {
					database.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						return nil, errors.New("error when retrieve policies from database")
					}
				})
				It("should not call the consumer as there is no trigger", func() {
					clock.Increment(2 * testPolicyPollerInterval)
					policyMap := policyManager.GetPolicies()
					Expect(len(policyMap)).To(Equal(0))
				})
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

		Context("when allowedMetricCache has already filled with metricstype details of the same appilication", func() {

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

				database.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
					return []*models.PolicyJson{{AppId: testAppId, PolicyStr: policyStr}}, nil
				}

			})
			It("should be able refresh allowed metrics cache", func() {
				Eventually(database.RetrievePoliciesCallCount).Should(Equal(1))
				clock.Increment(1 * testPolicyPollerInterval)
				Eventually(database.RetrievePoliciesCallCount).Should(Equal(2))

				Expect(policyManager.RefreshAllowedMetricCache(policyMap)).To(BeTrue())
				res, found := allowedMetricCache.Get(testAppId)
				maps := res.(map[string]struct{})

				Expect(found).To(BeTrue())
				Expect(maps).Should(HaveKey("test-metric-name"))
				Expect(maps).ShouldNot(HaveKey("queuelength"))
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
