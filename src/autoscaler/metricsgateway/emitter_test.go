package metricsgateway_test

import (
	. "autoscaler/metricsgateway"
	"autoscaler/metricsgateway/testhelpers"
	"strings"
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Emitter", func() {
	var (
		logger                     *lagertest.TestLogger
		fakeMetricServer           *ghttp.Server
		metricServerAddress        string
		messageChan                chan []byte
		messageTypeChan            chan int
		bufferSize                 int64 = 500
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
		fakeMetricServer = ghttp.NewServer()
		metricServerAddress = strings.Replace(fakeMetricServer.URL(), "http", "ws", 1)
		messageChan = make(chan []byte, 10)
		messageTypeChan = make(chan int, 10)
		wsh := testhelpers.NewWebsocketHandler(messageChan, messageTypeChan, 100*time.Second)

		fakeMetricServer.RouteToHandler("GET", "/v1/envelopes", wsh.ServeWebsocket)
	})
	Context("Start", func() {
		JustBeforeEach(func() {
			emitter = NewEnvelopeEmitter(logger, bufferSize, metricServerAddress, nil, testHandshakeTimeout, fclock, verifyWSConnectionInterval)
			emitter.Start()
			emitter.Accept(&testEnvelope)
		})
		It("should emit envelops to metricServer", func() {
			Eventually(messageChan).Should(Receive())
		})

		Context("when metricServer is not started", func() {
			BeforeEach(func() {
				fakeMetricServer.Close()
			})
			It("failed to start emitter", func() {
				Consistently(messageChan).ShouldNot(Receive())
				Eventually(logger.Buffer).Should(Say("failed-to-start-emimtter"))
			})
		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			emitter = NewEnvelopeEmitter(logger, bufferSize, metricServerAddress, nil, testHandshakeTimeout, fclock, verifyWSConnectionInterval)
			emitter.Start()
			Eventually(logger.Buffer).Should(Say("started"))
			emitter.Accept(&testEnvelope)
			Eventually(messageChan).Should(Receive())
			emitter.Stop()
			By("should close ws connection")
			Eventually(messageTypeChan).Should(Receive(Equal(websocket.CloseMessage)))

			emitter.Accept(&testEnvelope)
		})
		It("should stop emitting envelope", func() {
			Consistently(messageChan).ShouldNot(Receive())
		})
	})
})
