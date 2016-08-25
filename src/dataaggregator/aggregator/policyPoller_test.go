package aggregator_test

import (
	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager"
	. "dataaggregator/aggregator"
	"dataaggregator/aggregator/fakes"
	. "dataaggregator/appmetric"
	. "dataaggregator/policy"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("PolicyPoller", func() {
	var (
		database   *fakes.FakeDB
		clock      *fakeclock.FakeClock
		poller     *PolicyPoller
		triggerMap map[string]*Trigger
		logger     lager.Logger
		consumer   Consumer
		appChan    chan *AppMonitor
		testAppId1 = "testAppId"
		policyStr1 = `
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
	)

	BeforeEach(func() {
		triggerMap = make(map[string]*Trigger)
		database = &fakes.FakeDB{}
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lager.NewLogger("PolicyPoller-test")
		consumer = func(triggers map[string]*Trigger, appChan chan *AppMonitor) {}
		appChan = make(chan *AppMonitor, 1)

	})
	Context("Start", func() {
		JustBeforeEach(func() {
			poller = NewPolicyPoller(logger, clock, TestPolicyPollerInterval, database, consumer, appChan)
			poller.Start()

		})
		AfterEach(func() {
			poller.Stop()
		})
		Context("when the poller is started", func() {
			BeforeEach(func() {
				database.RetrievePoliciesStub = func() ([]*PolicyJson, error) {
					return []*PolicyJson{&PolicyJson{AppId: testAppId1, PolicyStr: policyStr1}}, nil
				}

			})
			It("should retrieve policies for every interval", func() {
				clock.Increment(2 * TestPolicyPollerInterval * time.Second)
				Eventually(database.RetrievePoliciesCallCount).Should(BeNumerically(">=", 2))
			})

			Context("when retrieve policies and compute triggers successfully", func() {
				var consumed chan map[string]*Trigger
				BeforeEach(func() {
					database.RetrievePoliciesStub = func() ([]*PolicyJson, error) {
						return []*PolicyJson{&PolicyJson{AppId: testAppId1, PolicyStr: policyStr1}}, nil
					}
					consumed = make(chan map[string]*Trigger, 1)
					consumer = func(triggers map[string]*Trigger, appChan chan *AppMonitor) {
						consumed <- triggers
					}
				})
				It("should call the consumer with the new triggers for every interval", func() {
					clock.Increment(2 * TestPolicyPollerInterval * time.Second)
					var triggerMap map[string]*Trigger
					Eventually(consumed).Should(Receive(&triggerMap))
					Expect(triggerMap[testAppId1]).To(Equal(&Trigger{
						AppId: testAppId1,
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
					}))
				})
			})
			Context("when return error when retrieve policies from database", func() {
				var consumed chan bool
				BeforeEach(func() {
					database.RetrievePoliciesStub = func() ([]*PolicyJson, error) {
						return nil, errors.New("error when retrieve policies from database")
					}
					consumed = make(chan bool, 1)
					consumer = func(triggers map[string]*Trigger, appChan chan *AppMonitor) {
						consumed <- true
					}
				})
				It("should not call the consumer as there is no trigger", func() {
					clock.Increment(2 * TestPolicyPollerInterval * time.Second)
					Consistently(consumed).ShouldNot(Receive())
				})
			})
		})
	})
	Context("Stop", func() {
		BeforeEach(func() {
			poller = NewPolicyPoller(logger, clock, TestPolicyPollerInterval, database, consumer, appChan)
			poller.Start()
			poller.Stop()
		})

		It("stops the polling", func() {
			clock.Increment(5 * TestPolicyPollerInterval * time.Second)
			Eventually(database.RetrievePoliciesCallCount).Should(Equal(1))
			Consistently(database.RetrievePoliciesCallCount).Should(Equal(1))
		})
	})
})
