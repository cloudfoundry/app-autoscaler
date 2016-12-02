package aggregator_test

import (
	. "autoscaler/eventgenerator/aggregator"
	"autoscaler/eventgenerator/aggregator/fakes"
	. "autoscaler/eventgenerator/generator"
	. "autoscaler/eventgenerator/model"
	"autoscaler/models"
	"net/http"
	"regexp"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Aggregator", func() {
	var (
		err               error
		getPolicies       GetPolicies
		evaluationManager *AppEvaluationManager
		aggregator        *Aggregator
		appMetricDatabase *fakes.FakeAppMetricDB
		policyDatabase    *fakes.FakePolicyDB
		clock             *fakeclock.FakeClock
		logger            lager.Logger
		metricServer      *ghttp.Server
		metricPollerCount int = 3
		evaluatorCount    int = 0
		triggerChan       chan []*Trigger
		appMonitorChan    chan *AppMonitor
		testAppId         string = "testAppId"
		testAppId2        string = "testAppId2"
		testAppId3        string = "testAppId3"
		testAppId4        string = "testAppId4"
		timestamp         int64  = time.Now().UnixNano()
		metricType        string = "MemoryUsage"
		unit              string = "bytes"
		fakeWaitDuration  time.Duration
		policyStr         = `
		{
		   "instance_min_count":1,
		   "instance_max_count":5,
		   "scaling_rules":[
		      {
		         "metric_type":"MemoryUsage",
		         "stat_window":300,
		         "breach_duration":300,
		         "threshold":30,
		         "operator":"<",
		         "cool_down_duration":300,
		         "adjustment":"-1"
		      }
		   ]
		}`
		metrics []*models.AppInstanceMetric = []*models.AppInstanceMetric{
			&models.AppInstanceMetric{
				AppId:         testAppId,
				InstanceIndex: 0,
				CollectedAt:   111111,
				Name:          metricType,
				Unit:          models.UnitBytes,
				Value:         "100",
				Timestamp:     111100,
			},
			&models.AppInstanceMetric{
				AppId:         testAppId,
				InstanceIndex: 1,
				CollectedAt:   111111,
				Name:          metricType,
				Unit:          models.UnitBytes,
				Value:         "200",
				Timestamp:     110000,
			},

			&models.AppInstanceMetric{
				AppId:         testAppId,
				InstanceIndex: 0,
				CollectedAt:   222222,
				Name:          metricType,
				Unit:          models.UnitBytes,
				Value:         "300",
				Timestamp:     222200,
			},
			&models.AppInstanceMetric{
				AppId:         testAppId,
				InstanceIndex: 1,
				CollectedAt:   222222,
				Name:          metricType,
				Unit:          models.UnitBytes,
				Value:         "400",
				Timestamp:     220000,
			},
		}
		policyMap map[string]*Policy = map[string]*Policy{
			testAppId: &Policy{
				AppId: testAppId,
				TriggerRecord: &TriggerRecord{
					InstanceMaxCount: 5,
					InstanceMinCount: 1,
					ScalingRules: []*ScalingRule{
						&ScalingRule{
							MetricType:       "MemoryUsage",
							StatWindow:       300,
							BreachDuration:   300,
							CoolDownDuration: 300,
							Threshold:        30,
							Operator:         "<",
							Adjustment:       "-1",
						},
					},
				},
			},
		}
		policyMap2 map[string]*Policy = map[string]*Policy{
			testAppId: &Policy{
				AppId: testAppId,
				TriggerRecord: &TriggerRecord{
					InstanceMaxCount: 5,
					InstanceMinCount: 1,
					ScalingRules: []*ScalingRule{
						&ScalingRule{
							MetricType:       "MemoryUsage",
							StatWindow:       300,
							BreachDuration:   300,
							CoolDownDuration: 300,
							Threshold:        30,
							Operator:         "<",
							Adjustment:       "-1",
						},
					},
				},
			},
			testAppId2: &Policy{
				AppId: testAppId2,
				TriggerRecord: &TriggerRecord{
					InstanceMaxCount: 5,
					InstanceMinCount: 1,
					ScalingRules: []*ScalingRule{
						&ScalingRule{
							MetricType:       "MemoryUsage",
							StatWindow:       300,
							BreachDuration:   300,
							CoolDownDuration: 300,
							Threshold:        30,
							Operator:         "<",
							Adjustment:       "-1",
						},
					},
				},
			},
			testAppId3: &Policy{
				AppId: testAppId3,
				TriggerRecord: &TriggerRecord{
					InstanceMaxCount: 5,
					InstanceMinCount: 1,
					ScalingRules: []*ScalingRule{
						&ScalingRule{
							MetricType:       "MemoryUsage",
							StatWindow:       300,
							BreachDuration:   300,
							CoolDownDuration: 300,
							Threshold:        30,
							Operator:         "<",
							Adjustment:       "-1",
						},
					},
				},
			},
			testAppId4: &Policy{
				AppId: testAppId4,
				TriggerRecord: &TriggerRecord{
					InstanceMaxCount: 5,
					InstanceMinCount: 1,
					ScalingRules: []*ScalingRule{
						&ScalingRule{
							MetricType:       "MemoryUsage",
							StatWindow:       300,
							BreachDuration:   300,
							CoolDownDuration: 300,
							Threshold:        30,
							Operator:         "<",
							Adjustment:       "-1",
						},
					},
				},
			},
		}
	)

	BeforeEach(func() {
		appMetricDatabase = &fakes.FakeAppMetricDB{}
		policyDatabase = &fakes.FakePolicyDB{}

		policyDatabase.RetrievePoliciesStub = func() ([]*PolicyJson, error) {
			return []*PolicyJson{&PolicyJson{AppId: testAppId, PolicyStr: policyStr}}, nil
		}

		appMetricDatabase.SaveAppMetricStub = func(appMetric *AppMetric) error {
			Expect(appMetric.AppId).To(Equal(testAppId))
			Expect(appMetric.MetricType).To(Equal(metricType))
			Expect(appMetric.Unit).To(Equal(unit))
			Expect(*appMetric.Value).To(Equal(int64(250)))
			return nil
		}
		getPolicies = func() map[string]*Policy {
			return policyMap
		}
		appMetricDatabase.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
			return []*AppMetric{}, nil
		}

		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("Aggregator-test")

		regPath := regexp.MustCompile(`^/v1/apps/.*/metrics_history/memory$`)
		metricServer = ghttp.NewServer()
		metricServer.RouteToHandler("GET", regPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
			&metrics))

		triggerChan = make(chan []*Trigger, 10)
		appMonitorChan = make(chan *AppMonitor, 10)
		evaluationManager, err = NewAppEvaluationManager(testEvaluateInteval, logger, clock, triggerChan, evaluatorCount, appMetricDatabase, "", getPolicies, nil)
		Expect(err).NotTo(HaveOccurred())
		if testEvaluateInteval > testAggregatorExecuteInterval {
			fakeWaitDuration = testEvaluateInteval
		} else {
			fakeWaitDuration = testAggregatorExecuteInterval
		}
	})

	Describe("ConsumeAppMetric", func() {
		var appmetric *AppMetric
		var value int64 = 250
		BeforeEach(func() {
			aggregator, err = NewAggregator(logger, clock, testAggregatorExecuteInterval, testPolicyPollerInterval, policyDatabase, appMetricDatabase, metricServer.URL(), metricPollerCount, evaluationManager, appMonitorChan, getPolicies, nil)
			Expect(err).NotTo(HaveOccurred())

			appmetric = &AppMetric{
				AppId:      testAppId,
				MetricType: metricType,
				Value:      &value,
				Unit:       "bytes",
				Timestamp:  timestamp}
		})

		Context("when there is data in appmetric", func() {
			JustBeforeEach(func() {
				aggregator.ConsumeAppMetric(appmetric)
			})

			It("should call database.SaveAppmetric to save the appmetric to database", func() {
				Eventually(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(1))
			})
		})

		Context("when the appmetric is nil", func() {
			BeforeEach(func() {
				appmetric = nil
			})

			JustBeforeEach(func() {
				aggregator.ConsumeAppMetric(appmetric)
			})

			It("should call database.SaveAppmetric to save the appmetric to database", func() {
				Consistently(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(0))
			})
		})
	})

	Describe("Start", func() {
		JustBeforeEach(func() {
			aggregator, err = NewAggregator(logger, clock, testAggregatorExecuteInterval, testPolicyPollerInterval, policyDatabase, appMetricDatabase, metricServer.URL(), metricPollerCount, evaluationManager, appMonitorChan, getPolicies, nil)
			Expect(err).NotTo(HaveOccurred())
			aggregator.Start()
			evaluationManager.Start()
			Eventually(clock.WatcherCount).Should(Equal(2)) //aggregator:1,evaluationManager:1
		})

		AfterEach(func() {
			aggregator.Stop()
		})

		It("should save the appmetric to database", func() {
			clock.Increment(1 * fakeWaitDuration)
			Eventually(appMetricDatabase.SaveAppMetricCallCount).Should(BeNumerically(">=", 1))
		})

		Context("MetricPoller", func() {
			var unBlockChan chan bool
			var calledChan chan string

			BeforeEach(func() {
				metricPollerCount = 4
				unBlockChan = make(chan bool)
				calledChan = make(chan string)
				getPolicies = func() map[string]*Policy {
					return policyMap2
				}
				appMetricDatabase.SaveAppMetricStub = func(appMetric *AppMetric) error {
					defer GinkgoRecover()
					calledChan <- appMetric.AppId
					Eventually(unBlockChan).Should(BeClosed())
					return nil
				}
			})

			It("should create MetricPollerCount metric-pollers", func() {
				clock.Increment(1 * fakeWaitDuration)
				for i := 0; i < metricPollerCount; i++ {
					Eventually(calledChan).Should(Receive())
				}
				Consistently(calledChan).ShouldNot(Receive())

				Eventually(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(int(metricPollerCount)))

				close(unBlockChan)
			})
		})
	})

	Describe("Stop", func() {
		var retrievePoliciesCallCount int
		var saveAppMetricCallCount int

		JustBeforeEach(func() {
			aggregator, err = NewAggregator(logger, clock, testAggregatorExecuteInterval, testPolicyPollerInterval, policyDatabase, appMetricDatabase, metricServer.URL(), metricPollerCount, evaluationManager, appMonitorChan, getPolicies, nil)
			Expect(err).NotTo(HaveOccurred())
			aggregator.Start()
			Eventually(clock.WatcherCount).Should(Equal(1)) //aggregator:1
			aggregator.Stop()

			retrievePoliciesCallCount = policyDatabase.RetrievePoliciesCallCount()
			saveAppMetricCallCount = appMetricDatabase.SaveAppMetricCallCount()
		})

		It("should not retrieve or save", func() {
			clock.Increment(1 * fakeWaitDuration)
			Eventually(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(saveAppMetricCallCount))
		})
	})
})
