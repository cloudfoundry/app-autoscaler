package helpers_test

import (
	"fmt"
	"strings"
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("WsHelper", func() {
	var (
		fakeMetricServer       *ghttp.Server
		metricServerAddress    string
		testHandshakeTimeout   = 5 * time.Millisecond
		testMaxSetupRetryCount = 10
		testMaxCloseRetryCount = 10
		testRetryDelay         = 500 * time.Millisecond
		messageChan            chan []byte
		pingPongChan           chan int
		wsHelper               WSHelper
		logger                 *lagertest.TestLogger
		wsh                    *testhelpers.WebsocketHandler
		testAppId              = "test-app-id"
		testEnvelope           = loggregator_v2.Envelope{
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
		fakeMetricServer = ghttp.NewServer()
		metricServerAddress = strings.Replace(fakeMetricServer.URL(), "http", "ws", 1)
		messageChan = make(chan []byte, 10)
		pingPongChan = make(chan int, 10)
		wsh = testhelpers.NewWebsocketHandler(messageChan, pingPongChan, 5*time.Second)
		fakeMetricServer.RouteToHandler("GET", "/v1/envelopes", wsh.ServeWebsocket)
		logger = lagertest.NewTestLogger("ws_helper")
	})
	AfterEach(func() {
		fakeMetricServer.Close()
	})
	Describe("SetupConn", func() {
		var err error
		JustBeforeEach(func() {
			wsHelper = NewWSHelper(metricServerAddress+routes.EnvelopePath, nil, testHandshakeTimeout, logger, testMaxSetupRetryCount, testMaxCloseRetryCount, testRetryDelay)
			err = wsHelper.SetupConn()

		})
		It("set up websocket connection", func() {
			Expect(err).NotTo(HaveOccurred())
		})
		Context("when scheme of url is not ws/wss", func() {
			BeforeEach(func() {
				metricServerAddress = strings.Replace(metricServerAddress, "ws", "http", 1)
			})
			It("fails to setup websocket connection", func() {
				Expect(err).To(Equal(fmt.Errorf("Invalid scheme '%s'", "http")))
			})
		})
		Context("when maximum number of setup retries reached", func() {
			BeforeEach(func() {
				fakeMetricServer.Close()
			})
			It("fails to setup websocket connection", func() {
				Expect(err).To(HaveOccurred())
				Eventually(logger.Buffer).Should(Say("failed-to-create-websocket-connection-to-metricserver"))
				Eventually(logger.Buffer).Should(Say("maximum-number-of-setup-retries-reached"))
			})
		})
	})
	Describe("CloseConn", func() {
		var err error
		BeforeEach(func() {
			wsHelper = NewWSHelper(metricServerAddress+routes.EnvelopePath, nil, testHandshakeTimeout, logger, testMaxSetupRetryCount, testMaxCloseRetryCount, testRetryDelay)
			err = wsHelper.SetupConn()
			Expect(err).NotTo(HaveOccurred())

			err = wsHelper.Ping()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(pingPongChan, 5*time.Second, 1*time.Second).Should(Receive(Equal(1)))
		})
		It("close the websocket connection", func() {
			err = wsHelper.CloseConn()
			Expect(err).NotTo(HaveOccurred())
			Eventually(logger.Buffer, 10*time.Second, 1*time.Second).Should(Say("successfully-close-ws-connection"))
		})
		Context("when maximum number of close retries reached", func() {
			BeforeEach(func() {
				fakeMetricServer.Close()
				wsh.CloseWSConnection()
			})
			It("fails to close websocket connection", func() {
				Eventually(func() error {
					return wsHelper.CloseConn()
				}).Should(HaveOccurred())
				Eventually(logger.Buffer).Should(Say("failed-to-send-close-message-to-metricserver"))
				Eventually(logger.Buffer).Should(Say("maximum-number-of-close-retries-reached"))

			})
		})
	})
	Describe("Ping", func() {
		var err error
		BeforeEach(func() {
			wsHelper = NewWSHelper(metricServerAddress+routes.EnvelopePath, nil, testHandshakeTimeout, logger, testMaxSetupRetryCount, testMaxCloseRetryCount, testRetryDelay)
			err = wsHelper.SetupConn()
			Expect(err).NotTo(HaveOccurred())

		})
		It("send ping message to server", func() {
			err = wsHelper.Ping()
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(pingPongChan, 5*time.Second, 1*time.Second).Should(Receive(Equal(1)))
		})
		Context("when server is down", func() {
			BeforeEach(func() {
				fakeMetricServer.Close()
				wsh.CloseWSConnection()

			})
			It("fails to ping and reconnect", func() {
				Eventually(func() error { return wsHelper.Ping() }, "2s").Should(HaveOccurred())
				Eventually(logger.Buffer, 10*time.Second, 1*time.Second).Should(Say("maximum-number-of-close-retries-reached"))
			})
		})
	})
	Describe("Write", func() {
		var err error
		BeforeEach(func() {
			wsHelper = NewWSHelper(metricServerAddress+routes.EnvelopePath, nil, testHandshakeTimeout, logger, testMaxSetupRetryCount, testMaxCloseRetryCount, testRetryDelay)
			err = wsHelper.SetupConn()
			Expect(err).NotTo(HaveOccurred())

		})
		It("write envelops to server", func() {
			Consistently(messageChan).ShouldNot(Receive())
			err = wsHelper.Write(&testEnvelope)
			Expect(err).NotTo(HaveOccurred())
			Eventually(messageChan).Should(Receive())
		})
		Context("when server is down", func() {
			BeforeEach(func() {
				fakeMetricServer.Close()
				wsh.CloseWSConnection()
			})
			It("fails to write envelops", func() {
				Eventually(func() error {
					return wsHelper.Write(&testEnvelope)
				}).Should(HaveOccurred())
				Eventually(logger.Buffer, 10*time.Second, 1*time.Second).Should(Say("maximum-number-of-close-retries-reached"))
			})
		})
	})
})
