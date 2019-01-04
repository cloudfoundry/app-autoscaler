package metricsgateway_test

import (
	"autoscaler/fakes"
	. "autoscaler/metricsgateway"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Emitter", func() {
	var (
		logger                     *lagertest.TestLogger
		metricServerAddress        string
		envelopChan                chan *loggregator_v2.Envelope
		wsMessageChan              chan int
		bufferSize                 int64 = 500
		fakeWSHelper               *fakes.FakeWSHelper
		emitter                    *EnvelopeEmitter
		testAppId                  = "test-app-id"
		testHandshakeTimeout       = 5 * time.Second
		fclock                     *fakeclock.FakeClock
		verifyWSConnectionInterval = 5 * time.Second

		testEnvelope = loggregator_v2.Envelope{
			SourceId: testAppId,
			Message: &loggregator_v2.Envelope_Gauge{
				Gauge: &loggregator_v2.Gauge{
					Metrics: map[string]*loggregator_v2.GaugeValue{
						"cpu": &loggregator_v2.GaugeValue{
							Unit:  "percentage",
							Value: 20.5,
						},
						"disk": &loggregator_v2.GaugeValue{
							Unit:  "bytes",
							Value: 3000000000,
						},
						"memory": &loggregator_v2.GaugeValue{
							Unit:  "bytes",
							Value: 1000000000,
						},
						"memory_quota": &loggregator_v2.GaugeValue{
							Unit:  "bytes",
							Value: 2000000000,
						},
					},
				},
			},
		}
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("emitter")
		fclock = fakeclock.NewFakeClock(time.Now())
		envelopChan = make(chan *loggregator_v2.Envelope, 10)
		wsMessageChan = make(chan int, 10)
		fakeWSHelper = &fakes.FakeWSHelper{}
		fakeWSHelper.WriteStub = func(envelope *loggregator_v2.Envelope) error {
			envelopChan <- envelope
			return nil
		}
		fakeWSHelper.PingStub = func() error {
			wsMessageChan <- websocket.PingMessage
			return nil
		}
		fakeWSHelper.CloseConnStub = func() error {
			wsMessageChan <- websocket.CloseMessage
			return nil
		}

	})
	Context("Start", func() {
		JustBeforeEach(func() {
			emitter = NewEnvelopeEmitter(logger, bufferSize, metricServerAddress, nil, testHandshakeTimeout, fclock, verifyWSConnectionInterval, fakeWSHelper)
			emitter.Start()
			emitter.Accept(&testEnvelope)
		})
		It("should emit envelops to metricServer", func() {
			Eventually(envelopChan).Should(Receive())
		})

		It("should send ping message to metricServer periodically", func() {
			fclock.Increment(1 * verifyWSConnectionInterval)
			Eventually(wsMessageChan).Should(Receive(Equal(websocket.PingMessage)))
			fclock.Increment(1 * verifyWSConnectionInterval)
			Eventually(wsMessageChan).Should(Receive(Equal(websocket.PingMessage)))
		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			emitter = NewEnvelopeEmitter(logger, bufferSize, metricServerAddress, nil, testHandshakeTimeout, fclock, verifyWSConnectionInterval, fakeWSHelper)
			emitter.Start()
			Eventually(logger.Buffer).Should(Say("started"))
			emitter.Accept(&testEnvelope)
			Eventually(envelopChan).Should(Receive())
			emitter.Stop()
			By("should close ws connection")
			Eventually(wsMessageChan).Should(Receive(Equal(websocket.CloseMessage)))

			emitter.Accept(&testEnvelope)
		})
		It("should stop emitting envelope", func() {
			Consistently(envelopChan).ShouldNot(Receive())
		})
	})
})
