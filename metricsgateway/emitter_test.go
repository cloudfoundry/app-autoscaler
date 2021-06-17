package metricsgateway_test

import (
	"autoscaler/fakes"
	. "autoscaler/metricsgateway"
	"errors"
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
		logger        *lagertest.TestLogger
		envelopChan   chan *loggregator_v2.Envelope = make(chan *loggregator_v2.Envelope, 10)
		wsMessageChan chan int                      = make(chan int, 10)

		bufferSize                 int = 500
		fakeWSHelper               *fakes.FakeWSHelper
		emitter                    Emitter
		testAppId                  = "test-app-id"
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
		var startError error
		JustBeforeEach(func() {
			emitter = NewEnvelopeEmitter(logger, bufferSize, fclock, verifyWSConnectionInterval, fakeWSHelper)
			startError = emitter.Start()
		})
		It("should emit envelops to metricServer", func() {
			Expect(startError).NotTo(HaveOccurred())
			emitter.Accept(&testEnvelope)
			Eventually(envelopChan).Should(Receive())
		})

		It("should send ping message to metricServer periodically", func() {
			Expect(startError).NotTo(HaveOccurred())
			fclock.WaitForWatcherAndIncrement(1 * verifyWSConnectionInterval)
			Eventually(wsMessageChan).Should(Receive(Equal(websocket.PingMessage)))
			fclock.WaitForWatcherAndIncrement(1 * verifyWSConnectionInterval)
			Eventually(wsMessageChan).Should(Receive(Equal(websocket.PingMessage)))
		})
		Context("when it fails to connect metricsserver", func() {
			BeforeEach(func() {
				fakeWSHelper.SetupConnStub = func() error {
					return errors.New("connection-error")
				}
			})
			It("failed to start", func() {
				Expect(startError).To(HaveOccurred())
				emitter.Accept(&testEnvelope)
				Consistently(envelopChan).ShouldNot(Receive())
			})
		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			emitter = NewEnvelopeEmitter(logger, bufferSize, fclock, verifyWSConnectionInterval, fakeWSHelper)
			emitter.Start()
			Eventually(logger.Buffer).Should(Say("started"))
			emitter.Accept(&testEnvelope)
			Eventually(envelopChan).Should(Receive())
			emitter.Stop()
			By("should close ws connection")
			Eventually(wsMessageChan).Should(Receive(Equal(websocket.CloseMessage)))
			Eventually(envelopChan).ShouldNot(Receive())
		})
		It("should stop emitting envelope", func() {
			emitter.Accept(&testEnvelope)
			Consistently(envelopChan).ShouldNot(Receive())
		})
	})
})
