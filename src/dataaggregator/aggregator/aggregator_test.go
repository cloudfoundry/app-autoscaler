package aggregator_test

import (
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "dataaggregator/aggregator"
	"dataaggregator/aggregator/fakes"
	. "dataaggregator/appmetric"
	. "dataaggregator/policy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "metricscollector/metrics"
	"net/http"
	"time"
)

var _ = Describe("Aggregator", func() {
	var (
		aggregator   *Aggregator
		database     *fakes.FakeDB
		clock        *fakeclock.FakeClock
		logger       lager.Logger
		metricServer *ghttp.Server
		testAppId    string = "testAppId"
		timestamp    int64  = time.Now().UnixNano()
		metricType   string = "MemoryUsage"
		unit         string = "bytes"
		policyStr           = `
		{
		   "instance_min_count":1,
		   "instance_max_count":5,
		   "scaling_rules":[
		      {
		         "metric_type":"MemoryUsage",
		         "stat_window_secs":300,
		         "breach_duration_secs":300,
		         "threshold":30,
		         "operator":"<",
		         "cool_down_secs":300,
		         "adjustment":"-1"
		      }
		   ]
		}`
		metrics []*Metric = []*Metric{
			&Metric{
				Name:      metricType,
				Unit:      unit,
				AppId:     testAppId,
				TimeStamp: timestamp,
				Instances: []InstanceMetric{InstanceMetric{
					Timestamp: timestamp,
					Index:     0,
					Value:     "100",
				}, InstanceMetric{
					Timestamp: timestamp,
					Index:     1,
					Value:     "200",
				}},
			},
			&Metric{
				Name:      metricType,
				Unit:      unit,
				AppId:     testAppId,
				TimeStamp: timestamp,
				Instances: []InstanceMetric{InstanceMetric{
					Timestamp: timestamp,
					Index:     0,
					Value:     "300",
				}, InstanceMetric{
					Timestamp: timestamp,
					Index:     1,
					Value:     "400",
				}},
			},
		}
	)
	BeforeEach(func() {
		database = &fakes.FakeDB{}
		database.RetrievePoliciesStub = func() ([]*PolicyJson, error) {
			return []*PolicyJson{&PolicyJson{AppId: testAppId, PolicyStr: policyStr}}, nil
		}
		database.SaveAppMetricStub = func(appMetric *AppMetric) error {
			Expect(appMetric.AppId).To(Equal("testAppId"))
			Expect(appMetric.MetricType).To(Equal(metricType))
			Expect(appMetric.Unit).To(Equal(unit))
			Expect(appMetric.Value).To(BeNumerically("==", 250))
			return nil
		}
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("Aggregator-test")
		metricServer = ghttp.NewServer()
		metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusOK,
			&metrics))
	})
	Context("ConsumeTrigger", func() {
		var triggerMap map[string]*Trigger
		var appChan chan *AppMonitor
		var appMonitor *AppMonitor
		JustBeforeEach(func() {
			appChan = make(chan *AppMonitor, 1)
			aggregator = NewAggregator(logger, clock, TestPolicyPollerInterval, database, metricServer.URL(), TestMetricPollerCount)
			triggerMap = map[string]*Trigger{testAppId: &Trigger{
				AppId: testAppId,
				TriggerRecord: &TriggerRecord{
					InstanceMaxCount: 5,
					InstanceMinCount: 1,
					ScalingRules: []*ScalingRule{&ScalingRule{
						MetricType:         "MemoryUsage",
						StatWindowSecs:     300,
						BreachDurationSecs: 300,
						CoolDownSecs:       300,
						Threshold:          30,
						Operator:           "<",
						Adjustment:         "-1",
					}}},
			}}
		})
		It("should parse the triggers to appmonitor and put them in appChan", func() {
			aggregator.ConsumeTrigger(triggerMap, appChan)
			Eventually(appChan).Should(Receive(&appMonitor))
			Expect(appMonitor).To(Equal(&AppMonitor{
				AppId:          testAppId,
				MetricType:     "MemoryUsage",
				StatWindowSecs: 300,
			}))
		})
	})
	Context("ConsumeAppMetric", func() {
		var appmetric *AppMetric
		JustBeforeEach(func() {
			aggregator = NewAggregator(logger, clock, TestPolicyPollerInterval, database, metricServer.URL(), TestMetricPollerCount)
			appmetric = &AppMetric{
				AppId:      testAppId,
				MetricType: metricType,
				Value:      250,
				Unit:       "bytes",
				Timestamp:  timestamp}
		})
		It("should save the appmetric to database", func() {
			aggregator.ConsumeAppMetric(appmetric)
			Eventually(database.SaveAppMetricCallCount).Should(BeNumerically("==", 1))
		})
	})
	Context("Start", func() {
		JustBeforeEach(func() {
			aggregator = NewAggregator(logger, clock, TestPolicyPollerInterval, database, metricServer.URL(), TestMetricPollerCount)
			aggregator.Start()
		})
		AfterEach(func() {
			aggregator.Stop()
		})
		It("should save the appmetric to database", func() {
			clock.Increment(2 * TestPolicyPollerInterval * time.Second)
			Eventually(database.RetrievePoliciesCallCount).Should(BeNumerically(">=", 2))
			Eventually(database.SaveAppMetricCallCount).Should(BeNumerically(">=", 2))

		})
	})
	Context("Stop", func() {
		var retrievePoliciesCallCount, saveAppMetricCallCount int
		JustBeforeEach(func() {
			aggregator = NewAggregator(logger, clock, TestPolicyPollerInterval, database, metricServer.URL(), TestMetricPollerCount)
			aggregator.Start()
			aggregator.Stop()
			retrievePoliciesCallCount = database.RetrievePoliciesCallCount()
			saveAppMetricCallCount = database.SaveAppMetricCallCount()

		})
		It("should return 1", func() {
			clock.Increment(10 * TestPolicyPollerInterval * time.Second)
			Eventually(database.RetrievePoliciesCallCount).Should(BeNumerically("==", retrievePoliciesCallCount))
			Eventually(database.SaveAppMetricCallCount).Should(BeNumerically("==", saveAppMetricCallCount))

		})
	})
})
