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
		metrics []*models.Metric = []*models.Metric{
			&models.Metric{
				Name:      metricType,
				Unit:      unit,
				AppId:     testAppId,
				Timestamp: timestamp,
				Instances: []models.InstanceMetric{models.InstanceMetric{
					Timestamp: timestamp,
					Index:     0,
					Value:     "100",
				}, models.InstanceMetric{
					Timestamp: timestamp,
					Index:     1,
					Value:     "200",
				}},
			},
			&models.Metric{
				Name:      metricType,
				Unit:      unit,
				AppId:     testAppId,
				Timestamp: timestamp,
				Instances: []models.InstanceMetric{models.InstanceMetric{
					Timestamp: timestamp,
					Index:     0,
					Value:     "300",
				}, models.InstanceMetric{
					Timestamp: timestamp,
					Index:     1,
					Value:     "400",
				}},
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
			Expect(appMetric.AppId).To(Equal("testAppId"))
			Expect(appMetric.MetricType).To(Equal(metricType))
			Expect(appMetric.Unit).To(Equal(unit))
			Expect(appMetric.Value).To(Equal(int64(250)))
			return nil
		}
		appMetricDatabase.RetrieveAppMetricsStub = func(appId string, metricType string, start int64, end int64) ([]*AppMetric, error) {
			return []*AppMetric{}, nil
		}
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("Aggregator-test")
		metricServer = ghttp.NewServer()
		regPath := regexp.MustCompile(`^/v1/apps/.*/metrics_history/memory$`)
		metricServer.RouteToHandler("GET", regPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
			&metrics))
		triggerChan = make(chan []*Trigger, 10)
		appMonitorChan = make(chan *AppMonitor, 10)
		evaluationManager = NewAppEvaluationManager(testEvaluateInteval, logger, clock, triggerChan, evaluatorCount, appMetricDatabase, "")
		if testEvaluateInteval > testAggregatorExecuteInterval {
			fakeWaitDuration = testEvaluateInteval
		} else {
			fakeWaitDuration = testAggregatorExecuteInterval
		}
	})
	Context("ConsumePolicy", func() {
		var policyMap map[string]*Policy
		var appChan chan *AppMonitor
		var appMonitor *AppMonitor
		var triggerArray []*Trigger
		BeforeEach(func() {
			appChan = make(chan *AppMonitor, 1)

			aggregator = NewAggregator(logger, clock, testAggregatorExecuteInterval, testPolicyPollerInterval, policyDatabase, appMetricDatabase, metricServer.URL(), 0, evaluationManager, appMonitorChan)
			policyMap = map[string]*Policy{testAppId: &Policy{
				AppId: testAppId,
				TriggerRecord: &TriggerRecord{
					InstanceMaxCount: 5,
					InstanceMinCount: 1,
					ScalingRules: []*ScalingRule{&ScalingRule{
						MetricType:       "MemoryUsage",
						StatWindow:       300,
						BreachDuration:   300,
						CoolDownDuration: 300,
						Threshold:        30,
						Operator:         "<",
						Adjustment:       "-1",
					}}},
			}}
		})
		Context("when there are data in triggerMap", func() {
			JustBeforeEach(func() {
				aggregator.ConsumePolicy(policyMap, appMonitorChan)
				aggregator.Start()
				evaluationManager.Start()
				Eventually(clock.WatcherCount).Should(Equal(3)) //policyPoller:1,aggregator:1,evaluationManager:1
				clock.Increment(1 * fakeWaitDuration)

			})
			It("should parse the triggers to appmonitor and put them in appChan", func() {

				Eventually(appMonitorChan).Should(Receive(&appMonitor))
				Expect(appMonitor).To(Equal(&AppMonitor{
					AppId:      testAppId,
					MetricType: "MemoryUsage",
					StatWindow: 300,
				}))

				Eventually(triggerChan).Should(Receive(&triggerArray))
				Expect(triggerArray).To(Equal([]*Trigger{&Trigger{
					AppId:            testAppId,
					MetricType:       "MemoryUsage",
					BreachDuration:   300,
					CoolDownDuration: 300,
					Threshold:        30,
					Operator:         "<",
					Adjustment:       "-1",
				}}))
			})
		})
		Context("when there is no data in policyMap", func() {
			BeforeEach(func() {
				policyMap = map[string]*Policy{}
			})
			JustBeforeEach(func() {
				aggregator.ConsumePolicy(policyMap, appChan)
				evaluationManager.Start()
				Eventually(clock.WatcherCount).Should(Equal(1))
				clock.Increment(1 * fakeWaitDuration)
			})
			It("should not receive any data from the appChan", func() {
				Consistently(appChan).ShouldNot(Receive())
				Consistently(triggerChan).ShouldNot(Receive())
			})
		})
		Context("when the policyMap is nil", func() {
			BeforeEach(func() {
				policyMap = nil
			})
			JustBeforeEach(func() {
				aggregator.ConsumePolicy(policyMap, appChan)
				evaluationManager.Start()
				Eventually(clock.WatcherCount).Should(Equal(1))
				clock.Increment(1 * fakeWaitDuration)
			})
			It("should not receive any data from the appChan", func() {
				Consistently(appChan).ShouldNot(Receive())
				Consistently(triggerChan).ShouldNot(Receive())
			})
		})

	})
	Context("ConsumeAppMetric", func() {
		var appmetric *AppMetric
		BeforeEach(func() {
			aggregator = NewAggregator(logger, clock, testAggregatorExecuteInterval, testPolicyPollerInterval, policyDatabase, appMetricDatabase, metricServer.URL(), metricPollerCount, evaluationManager, appMonitorChan)
			appmetric = &AppMetric{
				AppId:      testAppId,
				MetricType: metricType,
				Value:      250,
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
	Context("Start", func() {
		JustBeforeEach(func() {
			aggregator = NewAggregator(logger, clock, testAggregatorExecuteInterval, testPolicyPollerInterval, policyDatabase, appMetricDatabase, metricServer.URL(), metricPollerCount, evaluationManager, appMonitorChan)
			aggregator.Start()
			evaluationManager.Start()
			Eventually(clock.WatcherCount).Should(Equal(3)) //policyPoller:1,aggregator:1,evaluationManager:1
		})
		AfterEach(func() {
			aggregator.Stop()
		})
		It("should save the appmetric to database", func() {
			clock.Increment(1 * fakeWaitDuration)
			Eventually(policyDatabase.RetrievePoliciesCallCount).Should(BeNumerically(">=", 1))
			Eventually(appMetricDatabase.SaveAppMetricCallCount).Should(BeNumerically(">=", 1))
		})

		Context("MetricPoller", func() {
			var unBlockChan chan bool
			var calledChan chan string
			BeforeEach(func() {
				metricPollerCount = 4
				unBlockChan = make(chan bool)
				calledChan = make(chan string)
				policyDatabase.RetrievePoliciesStub = func() ([]*PolicyJson, error) {
					return []*PolicyJson{&PolicyJson{AppId: testAppId, PolicyStr: policyStr}, &PolicyJson{AppId: testAppId2, PolicyStr: policyStr}, &PolicyJson{AppId: testAppId3, PolicyStr: policyStr}, &PolicyJson{AppId: testAppId4, PolicyStr: policyStr}}, nil
				}
				appMetricDatabase.SaveAppMetricStub = func(appMetric *AppMetric) error {
					defer GinkgoRecover()
					calledChan <- appMetric.AppId
					<-unBlockChan
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
	Context("Stop", func() {
		var retrievePoliciesCallCount, saveAppMetricCallCount int
		JustBeforeEach(func() {
			aggregator = NewAggregator(logger, clock, testAggregatorExecuteInterval, testPolicyPollerInterval, policyDatabase, appMetricDatabase, metricServer.URL(), metricPollerCount, evaluationManager, appMonitorChan)
			aggregator.Start()
			Eventually(clock.WatcherCount).Should(Equal(2)) //policyPoller:1,aggregator:1
			aggregator.Stop()
			retrievePoliciesCallCount = policyDatabase.RetrievePoliciesCallCount()
			saveAppMetricCallCount = appMetricDatabase.SaveAppMetricCallCount()

		})
		It("should not retrieve or save", func() {
			clock.Increment(1 * fakeWaitDuration)
			Eventually(policyDatabase.RetrievePoliciesCallCount).Should(Equal(retrievePoliciesCallCount))
			Eventually(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(saveAppMetricCallCount))
		})
	})
})
