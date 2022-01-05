package aggregator_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppManager", func() {
	var (
		policyDB    *fakes.FakePolicyDB
		appMetricDB *fakes.FakeAppMetricDB
		clock       *fakeclock.FakeClock
		appManager  *AppManager
		logger      lager.Logger
		testAppId   = "testAppId"
		policyStr   = `
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
		nodeNum         int
		nodeIndex       int
		cacheSizePerApp int
	)

	BeforeEach(func() {
		policyDB = &fakes.FakePolicyDB{}
		appMetricDB = &fakes.FakeAppMetricDB{}
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("AppManager-test")
		nodeNum = 1
		nodeIndex = 0
		cacheSizePerApp = 5
	})
	Context("Start", func() {
		JustBeforeEach(func() {
			appManager = NewAppManager(logger, clock, testPolicyPollerInterval, nodeNum, nodeIndex, cacheSizePerApp, policyDB, appMetricDB)
			appManager.Start()

		})

		AfterEach(func() {
			appManager.Stop()
		})

		Context("when the AppManager is started", func() {
			BeforeEach(func() {
				policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
					return []*models.PolicyJson{{AppId: testAppId, PolicyStr: policyStr}}, nil
				}

			})
			It("should retrieve and get policies successfully for every interval", func() {
				Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(1))
				clock.Increment(1 * testPolicyPollerInterval)
				Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(2))
				clock.Increment(1 * testPolicyPollerInterval)
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

			Context("when running with 3 nodes", func() {
				BeforeEach(func() {
					nodeNum = 3
					var i int
					policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
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

						Eventually(appManager.GetPolicies).Should(HaveLen(2))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-1"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-2"))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-3"))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-4"))

						clock.Increment(1 * testPolicyPollerInterval)
						Consistently(appManager.GetPolicies).Should(HaveLen(2))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-3"))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-4"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-5"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-6"))

						clock.Increment(1 * testPolicyPollerInterval)
						Eventually(appManager.GetPolicies).Should(HaveLen(1))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-8"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-5"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-6"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-7"))

					})
				})
				Context("when current index is 1", func() {
					BeforeEach(func() {
						nodeIndex = 1
					})

					It("retrieves app shard 1", func() {
						Consistently(appManager.GetPolicies).Should(BeEmpty())

						clock.Increment(1 * testPolicyPollerInterval)
						Eventually(appManager.GetPolicies).Should(HaveLen(2))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-3"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-4"))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-5"))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-6"))

						clock.Increment(1 * testPolicyPollerInterval)
						Consistently(appManager.GetPolicies).Should(HaveLen(2))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-5"))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-6"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-7"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-8"))
					})
				})

				Context("when current index is 2", func() {
					BeforeEach(func() {
						nodeIndex = 2
					})

					It("retrieves app shard 2", func() {
						Eventually(appManager.GetPolicies).Should(HaveLen(2))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-1"))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-2"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-3"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-4"))

						clock.Increment(1 * testPolicyPollerInterval)
						Eventually(appManager.GetPolicies).Should(BeEmpty())

						clock.Increment(1 * testPolicyPollerInterval)
						Eventually(appManager.GetPolicies).Should(HaveLen(1))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-5"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-6"))
						Consistently(appManager.GetPolicies).Should(HaveKey("app-id-7"))
						Consistently(appManager.GetPolicies).ShouldNot(HaveKey("app-id-8"))
					})
				})

			})

			Context("when retrieving policies from database fails", func() {
				BeforeEach(func() {
					policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
						return nil, errors.New("error when retrieve policies from database")
					}
				})
				It("should not call the consumer as there is no trigger", func() {
					clock.Increment(2 * testPolicyPollerInterval)
					policyMap := appManager.GetPolicies()
					Expect(len(policyMap)).To(Equal(0))
				})
			})
		})
	})

	Context("Save and query metrics", func() {
		JustBeforeEach(func() {
			appManager = NewAppManager(logger, clock, testPolicyPollerInterval, nodeNum, nodeIndex, cacheSizePerApp, policyDB, appMetricDB)
			appManager.Start()

		})

		AfterEach(func() {
			appManager.Stop()
		})

		Context("running with 1 node", func() {
			BeforeEach(func() {
				policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
					return []*models.PolicyJson{{AppId: testAppId, PolicyStr: policyStr}}, nil
				}
				cacheSizePerApp = 3
			})
			It("should be able to save and query metrics", func() {
				Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(1))
				clock.Increment(1 * testPolicyPollerInterval)
				Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(2))

				appMetric1 := &models.AppMetric{
					AppId:      testAppId,
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  100,
				}

				appMetric2 := &models.AppMetric{
					AppId:      testAppId,
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  200,
				}

				appMetric3 := &models.AppMetric{
					AppId:      testAppId,
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  300,
				}

				appMetric4 := &models.AppMetric{
					AppId:      testAppId,
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  400,
				}

				anotheAppMetric1 := &models.AppMetric{
					AppId:      "another-app-id",
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  100,
				}

				anotheAppMetric2 := &models.AppMetric{
					AppId:      "another-app-id",
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  200,
				}

				Expect(appManager.SaveMetricToCache(appMetric1)).To(BeTrue())
				Expect(appManager.SaveMetricToCache(appMetric2)).To(BeTrue())
				Expect(appManager.SaveMetricToCache(appMetric3)).To(BeTrue())
				Expect(appManager.SaveMetricToCache(appMetric4)).To(BeTrue())
				Expect(appManager.SaveMetricToCache(anotheAppMetric1)).To(BeFalse())
				Expect(appManager.SaveMetricToCache(anotheAppMetric2)).To(BeFalse())

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

		Context("when running with 3 nodes and current node index is 0", func() {
			BeforeEach(func() {
				nodeNum = 3
				nodeIndex = 0
				var i int
				policyDB.RetrievePoliciesStub = func() ([]*models.PolicyJson, error) {
					i++
					switch i {
					case 1:
						return []*models.PolicyJson{
							{AppId: "app-id-1", PolicyStr: policyStr},
							{AppId: "app-id-3", PolicyStr: policyStr},
						}, nil
					case 2:
						return []*models.PolicyJson{
							{AppId: "app-id-2", PolicyStr: policyStr},
							{AppId: "app-id-4", PolicyStr: policyStr},
						}, nil
					}
					return []*models.PolicyJson{}, nil
				}

			})
			It("caches app shard 0", func() {
				appMetric1 := &models.AppMetric{
					AppId:      "app-id-1",
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  100,
				}

				appMetric2 := &models.AppMetric{
					AppId:      "app-id-2",
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  100,
				}

				appMetric3 := &models.AppMetric{
					AppId:      "app-id-3",
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  100,
				}

				appMetric4 := &models.AppMetric{
					AppId:      "app-id-4",
					MetricType: "test-metric-type",
					Value:      "100",
					Unit:       "test-unit",
					Timestamp:  100,
				}

				Eventually(appManager.GetPolicies).Should(HaveLen(1))

				Expect(appManager.SaveMetricToCache(appMetric1)).To(BeFalse())
				Expect(appManager.SaveMetricToCache(appMetric2)).To(BeFalse())
				Expect(appManager.SaveMetricToCache(appMetric3)).To(BeTrue())
				Expect(appManager.SaveMetricToCache(appMetric4)).To(BeFalse())

				clock.Increment(1 * testPolicyPollerInterval)
				Consistently(appManager.GetPolicies).Should(HaveLen(1))
				Expect(appManager.SaveMetricToCache(appMetric1)).To(BeFalse())
				Expect(appManager.SaveMetricToCache(appMetric2)).To(BeFalse())
				Expect(appManager.SaveMetricToCache(appMetric3)).To(BeFalse())
				Expect(appManager.SaveMetricToCache(appMetric4)).To(BeTrue())
			})
		})

	})

	Context("Stop", func() {
		BeforeEach(func() {
			appManager = NewAppManager(logger, clock, testPolicyPollerInterval, nodeNum, nodeIndex, cacheSizePerApp, policyDB, appMetricDB)
			appManager.Start()
			Eventually(policyDB.RetrievePoliciesCallCount).Should(Equal(1))

			appManager.Stop()
		})

		It("stops the polling", func() {
			clock.Increment(5 * testPolicyPollerInterval)
			Consistently(policyDB.RetrievePoliciesCallCount).Should(Or(Equal(1), Equal(2)))
		})
	})
})
