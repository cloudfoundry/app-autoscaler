package manager_test

import (
	"encoding/json"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	mfcache "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/cache"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/manager"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyManager", func() {
	var (
		database                 *fakes.FakePolicyDB
		clock                    *fakeclock.FakeClock
		policyManager            *PolicyManager
		testPolicyPollerInterval time.Duration
		allowedMetricCache       *mfcache.AllowedMetricCache
		allowedMetricTypeSet     map[string]struct{}
		policyMap                map[string]*models.AppPolicy
		logger                   lager.Logger
		scalingPolicy            *models.PolicyDefinition
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
		allowedMetricCache = mfcache.New(10*time.Minute, -1)
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
				scalingPolicy = &models.PolicyDefinition{
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
				allowedMetrics := res
				Expect(found).To(BeTrue())
				Expect(allowedMetrics).Should(HaveKey("queuelength"))

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
					allowedMetrics := res

					Expect(found).To(BeTrue())
					Expect(allowedMetrics).Should(HaveKey("test-metric-name"))
					Expect(allowedMetrics).ShouldNot(HaveKey("queuelength"))
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

	Context("RefreshAllowedMetricCache concurrency", func() {
		// Regression: RefreshAllowedMetricCache built a single allowedMetricTypeSet
		// map and Replace'd that same instance into the cache for every app, so
		// all entries aliased one shared map that the refresh loop kept mutating.
		// A concurrent reader (e.g. json.Marshal of the set for logging) then
		// iterated a map being written, crashing the process with
		// "fatal error: concurrent map iteration and map write".
		var (
			appA = "app-a"
			appB = "app-b"
		)

		BeforeEach(func() {
			policyManager = NewPolicyManager(logger, clock, testPolicyPollerInterval, database, allowedMetricCache, 10*time.Minute)
			// Seed two apps so the refresh loop iterates more than one entry.
			allowedMetricCache.Set(appA, make(map[string]struct{}), 10*time.Minute)
			allowedMetricCache.Set(appB, make(map[string]struct{}), 10*time.Minute)
		})

		It("gives each app its own metric set", func() {
			policies := map[string]*models.AppPolicy{
				appA: {AppId: appA, ScalingPolicy: &models.PolicyDefinition{
					ScalingRules: []*models.ScalingRule{{MetricType: "metric-a"}}}},
				appB: {AppId: appB, ScalingPolicy: &models.PolicyDefinition{
					ScalingRules: []*models.ScalingRule{{MetricType: "metric-b"}}}},
			}
			Expect(policyManager.RefreshAllowedMetricCache(policies)).To(Succeed())

			resA, foundA := allowedMetricCache.Get(appA)
			Expect(foundA).To(BeTrue())
			resB, foundB := allowedMetricCache.Get(appB)
			Expect(foundB).To(BeTrue())

			// Each entry must contain only its own app's metric, not a shared
			// union of both.
			Expect(resA).To(HaveKey("metric-a"))
			Expect(resA).NotTo(HaveKey("metric-b"))
			Expect(resB).To(HaveKey("metric-b"))
			Expect(resB).NotTo(HaveKey("metric-a"))
		})

		It("does not race with a concurrent reader of the cached set", func() {
			// Under `go test -race` this trips the detector (and the fatal
			// "concurrent map iteration and map write") when the refresh loop
			// mutates a map another goroutine is marshalling.
			policies := map[string]*models.AppPolicy{
				appA: {AppId: appA, ScalingPolicy: &models.PolicyDefinition{
					ScalingRules: []*models.ScalingRule{{MetricType: "metric-a"}}}},
				appB: {AppId: appB, ScalingPolicy: &models.PolicyDefinition{
					ScalingRules: []*models.ScalingRule{{MetricType: "metric-b"}}}},
			}

			stop := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
						if res, found := allowedMetricCache.Get(appA); found {
							_, _ = json.Marshal(res)
						}
					}
				}
			}()

			for i := 0; i < 1000; i++ {
				Expect(policyManager.RefreshAllowedMetricCache(policies)).To(Succeed())
			}
			close(stop)
			wg.Wait()
		})
	})
})
