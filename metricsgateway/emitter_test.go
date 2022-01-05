package metricsgateway_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Emitter", func() {
	var (
		logger        *lagertest.TestLogger
		envelopChan   = make(chan *loggregator_v2.Envelope, 10)
		wsMessageChan = make(chan int, 10)

		bufferSize                 = 500
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
						"cpu": {
							Unit:  "percentage",
							Value: 20.5,
						},
						"disk": {
							Unit:  "bytes",
							Value: 3000000000,
						},
						"memory": {
							Unit:  "bytes",
							Value: 1000000000,
						},
						"memory_quota": {
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
			err := emitter.Start()
			Expect(err).NotTo(HaveOccurred())
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
