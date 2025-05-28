package aggregator_test

import (
	"errors"
	"sort"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

const testPolicyStr = `
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

var _ = Describe("AppManager", func() {
	var (
		policyDB       *fakes.FakePolicyDB
		appMetricDB    *fakes.FakeAppMetricDB
		clock          *fakeclock.FakeClock
		appManager     *AppManager
		logger         lager.Logger
		testAppId      = "testAppId"
		testPool       = config.PoolConfig{}
		testAggregator = config.AggregatorConfig{}
	)

	BeforeEach(func() {
		policyDB = &fakes.FakePolicyDB{}
		appMetricDB = &fakes.FakeAppMetricDB{}

		testAggregator.PolicyPollerInterval = testPolicyPollerInterval

		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("AppManager-test")
		testPool.TotalInstances = 1
		testPool.InstanceIndex = 0
		testAggregator.MetricCacheSizePerApp = 5
	})
	Context("Start", func() {
		JustBeforeEach(func() {
			appManager = NewAppManager(logger, clock, testAggregator, testPool, policyDB, appMetricDB)
			appManager.Start()

		})

		AfterEach(func() {
			appManager.Stop()
		})

		Context("Save and query metrics", func() {
			Context("running with 1 node", func() {
				BeforeEach(func() {
					policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						return []*models.PolicyJson{{AppId: testAppId, PolicyStr: testPolicyStr}}, nil
					}
					testAggregator.MetricCacheSizePerApp = 3
				})

				It("should be able to save and query metrics", func() {
					Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(1))
					clock.Increment(1 * testAggregator.PolicyPollerInterval)
					Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(2))

					appMetric1 := newAppMetric(testAppId, 100)
					appMetric2 := newAppMetric(testAppId, 200)
					appMetric3 := newAppMetric(testAppId, 300)
					appMetric4 := newAppMetric(testAppId, 400)

					anotherAppMetric1 := newAppMetric("another-app-id", 100)
					anotherAppMetric2 := newAppMetric("another-app-id", 200)

					Expect(appManager.SaveMetricToCache(appMetric1)).To(BeTrue())
					Expect(appManager.SaveMetricToCache(appMetric2)).To(BeTrue())
					Expect(appManager.SaveMetricToCache(appMetric3)).To(BeTrue())
					Expect(appManager.SaveMetricToCache(appMetric4)).To(BeTrue())
					Expect(appManager.SaveMetricToCache(anotherAppMetric1)).To(BeFalse())
					Expect(appManager.SaveMetricToCache(anotherAppMetric2)).To(BeFalse())

					By("cache hit")
					data, err := appManager.QueryAppMetrics(testAppId, "test-metric-type", 300, 500, db.ASC)
					Expect(err).NotTo(HaveOccurred())
					Expect(data).To(Equal([]*models.AppMetric{appMetric3, appMetric4}))

					By("cache miss")
					appMetricDB.RetrieveAppMetricsReturns([]*models.AppMetric{appMetric1, appMetric2}, nil)
					data, err = appManager.QueryAppMetrics(testAppId, "test-metric-type", 100, 200, db.ASC)
					Expect(err).NotTo(HaveOccurred())
					Expect(appMetricDB.RetrieveAppMetricsCallCount()).To(Equal(1))
					Expect(data).To(Equal([]*models.AppMetric{appMetric1, appMetric2}))

				})
			})

			When("running with 3 nodes and current node index is 0", func() {
				BeforeEach(func() {
					testPool.TotalInstances = 3
					testPool.InstanceIndex = 0
					var i int
					policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						i++
						switch i {
						case 1:
							return []*models.PolicyJson{
								{AppId: "app-id-1", PolicyStr: testPolicyStr},
								{AppId: "app-id-3", PolicyStr: testPolicyStr},
							}, nil
						case 2:
							return []*models.PolicyJson{
								{AppId: "app-id-2", PolicyStr: testPolicyStr},
								{AppId: "app-id-4", PolicyStr: testPolicyStr},
							}, nil
						}
						return []*models.PolicyJson{}, nil
					}

				})
				It("caches app shard 0", func() {
					appMetric1 := newAppMetric("app-id-1", 100)
					appMetric2 := newAppMetric("app-id-2", 100)
					appMetric3 := newAppMetric("app-id-3", 100)
					appMetric4 := newAppMetric("app-id-4", 100)

					Eventually(appManager.GetPolicies).Should(HaveLen(1))

					Expect(appManager.SaveMetricToCache(appMetric1)).To(BeFalse())
					Expect(appManager.SaveMetricToCache(appMetric2)).To(BeFalse())
					Expect(appManager.SaveMetricToCache(appMetric3)).To(BeTrue())
					Expect(appManager.SaveMetricToCache(appMetric4)).To(BeFalse())

					clock.Increment(1 * testAggregator.PolicyPollerInterval)
					Consistently(appManager.GetPolicies).Should(HaveLen(1))
					Expect(appManager.SaveMetricToCache(appMetric1)).To(BeFalse())
					Expect(appManager.SaveMetricToCache(appMetric2)).To(BeFalse())
					Expect(appManager.SaveMetricToCache(appMetric3)).To(BeFalse())
					Expect(appManager.SaveMetricToCache(appMetric4)).To(BeTrue())
				})
			})
		})

		When("the AppManager is started", func() {
			BeforeEach(func() {
				policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
					return []*models.PolicyJson{{AppId: testAppId, PolicyStr: testPolicyStr}}, nil
				}

			})
			It("should retrieve and get policies successfully for every interval", func() {
				Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(1))
				clock.Increment(1 * testAggregator.PolicyPollerInterval)
				Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(2))
				clock.Increment(1 * testAggregator.PolicyPollerInterval)
				Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(3))
				Eventually(appManager.GetPolicies).Should(Equal(map[string]*models.AppPolicy{
					testAppId: {
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

			When("running with 3 nodes", func() {
				BeforeEach(func() {
					testPool.TotalInstances = 3

					var i int
					policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						i++
						switch i {
						case 1:
							return []*models.PolicyJson{
								{AppId: "app-id-1", PolicyStr: testPolicyStr},
								{AppId: "app-id-2", PolicyStr: testPolicyStr},
								{AppId: "app-id-3", PolicyStr: testPolicyStr},
								{AppId: "app-id-4", PolicyStr: testPolicyStr},
							}, nil
						case 2:
							return []*models.PolicyJson{
								{AppId: "app-id-3", PolicyStr: testPolicyStr},
								{AppId: "app-id-4", PolicyStr: testPolicyStr},
								{AppId: "app-id-5", PolicyStr: testPolicyStr},
								{AppId: "app-id-6", PolicyStr: testPolicyStr},
							}, nil
						case 3:
							return []*models.PolicyJson{
								{AppId: "app-id-5", PolicyStr: testPolicyStr},
								{AppId: "app-id-6", PolicyStr: testPolicyStr},
								{AppId: "app-id-7", PolicyStr: testPolicyStr},
								{AppId: "app-id-8", PolicyStr: testPolicyStr},
							}, nil

						}
						return []*models.PolicyJson{}, nil
					}

				})

				When("current index is 0", func() {
					BeforeEach(func() {
						testPool.InstanceIndex = 0
					})

					It("retrieves app shard 0", func() {
						Eventually(appManager.GetPolicies).Should(HaveExactKeys("app-id-3", "app-id-4"))
						Consistently(appManager.GetPolicies).Should(HaveExactKeys("app-id-3", "app-id-4"))

						clock.Increment(testAggregator.PolicyPollerInterval)
						Consistently(appManager.GetPolicies).Should(HaveExactKeys("app-id-3", "app-id-4"))

						clock.Increment(testAggregator.PolicyPollerInterval)
						Eventually(appManager.GetPolicies).Should(HaveExactKeys("app-id-8"))
						Consistently(appManager.GetPolicies).Should(HaveExactKeys("app-id-8"))
					})
				})

				When("current index is 1", func() {
					BeforeEach(func() {
						testPool.InstanceIndex = 1
					})

					It("retrieves app shard 1", func() {
						Consistently(appManager.GetPolicies).Should(BeEmpty())

						clock.Increment(testAggregator.PolicyPollerInterval)
						Eventually(appManager.GetPolicies).Should(HaveExactKeys("app-id-5", "app-id-6"))
						Consistently(appManager.GetPolicies).Should(HaveExactKeys("app-id-5", "app-id-6"))

						clock.Increment(testAggregator.PolicyPollerInterval)
						Consistently(appManager.GetPolicies).Should(HaveExactKeys("app-id-5", "app-id-6"))
					})
				})

				When("current index is 2", func() {
					BeforeEach(func() {
						testPool.InstanceIndex = 2
					})

					It("retrieves app shard 2", func() {
						Eventually(appManager.GetPolicies).Should(HaveExactKeys("app-id-1", "app-id-2"))
						Consistently(appManager.GetPolicies).Should(HaveExactKeys("app-id-1", "app-id-2"))

						clock.Increment(testAggregator.PolicyPollerInterval)
						Eventually(appManager.GetPolicies).Should(BeEmpty())

						clock.Increment(testAggregator.PolicyPollerInterval)
						Eventually(appManager.GetPolicies).Should(HaveExactKeys("app-id-7"))
						Consistently(appManager.GetPolicies).Should(HaveExactKeys("app-id-7"))
					})
				})

			})

			When("retrieving policies from database fails", func() {
				BeforeEach(func() {
					policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						return nil, errors.New("error when retrieve policies from database")
					}
				})
				It("should not call the consumer as there is no trigger", func() {
					clock.Increment(2 * testAggregator.PolicyPollerInterval)
					policyMap := appManager.GetPolicies()
					Expect(len(policyMap)).To(Equal(0))
				})
			})
		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			appManager = NewAppManager(logger, clock, testAggregator, testPool, policyDB, appMetricDB)
			appManager.Start()
			Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(1))

			appManager.Stop()
		})

		It("stops the polling", func() {
			clock.Increment(5 * testAggregator.PolicyPollerInterval)
			Consistently(policyDB.RetrievePoliciesCallCount).Should(Or(Equal(1), Equal(2)))
		})
	})
})

func newAppMetric(appId string, ts int64) *models.AppMetric {
	return &models.AppMetric{
		AppId:      appId,
		MetricType: "test-metric-type",
		Value:      "100",
		Unit:       "test-unit",
		Timestamp:  ts,
	}
}

func HaveExactKeys(keys ...string) types.GomegaMatcher {
	return WithTransform(func(m map[string]*models.AppPolicy) []string {
		actualKeys := make([]string, 0, len(m))
		for k := range m {
			actualKeys = append(actualKeys, k)
		}
		sort.Strings(actualKeys)
		sort.Strings(keys)
		return actualKeys
	}, Equal(keys))
}
