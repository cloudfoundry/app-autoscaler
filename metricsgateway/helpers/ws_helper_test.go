package helpers_test

import (
	"fmt"
	"strings"
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("WsHelper", func() {
	var (
		fakeMetricServer       *ghttp.Server
		metricServerAddress    string
		testHandshakeTimeout   = 5 * time.Millisecond
		testMaxSetupRetryCount = 3
		testMaxCloseRetryCount = 3
		testRetryDelay         = 1 * time.Millisecond
		messageChan            chan []byte
		pingPongChan           chan int
		wsHelper               *WsHelper
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
				fakeMetricServer.RouteToHandler("GET", "/v1/envelopes",
					testhelpers.RespondWithMultiple(
						ghttp.RespondWith(500, ""),
						ghttp.RespondWith(500, ""),
						ghttp.RespondWith(500, ""),
						ghttp.RespondWith(500, ""),
					),
				)
			})
			It("fails to setup websocket connection", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(fmt.Sprintf(".*failed after %d retries:.*", testMaxCloseRetryCount))))
				Expect(err).To(MatchError(MatchRegexp(".*bad handshake.*")))
				Expect(len(fakeMetricServer.ReceivedRequests())).To(Equal(4))
			})
		})
		Context("when retries then connects correctly", func() {
			BeforeEach(func() {
				fakeMetricServer.RouteToHandler("GET", "/v1/envelopes",
					testhelpers.RespondWithMultiple(
						ghttp.RespondWith(500, ""),
						ghttp.RespondWith(500, ""),
						ghttp.RespondWith(500, ""),
						wsh.ServeWebsocket,
					),
				)
			})
			It("successfully connects", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(len(fakeMetricServer.ReceivedRequests())).To(Equal(4))
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
			wsHelper.CloseWaitTime = 100 * time.Millisecond
			err = wsHelper.CloseConn()
			Expect(err).NotTo(HaveOccurred())
			Eventually(wsHelper.IsClosed, 200*time.Millisecond, 50*time.Millisecond).Should(BeTrue())
		})
		Context("when maximum number of close retries reached", func() {
			BeforeEach(func() {
				fakeMetricServer.Close()
				wsh.CloseWSConnection()
			})
			It("fails to close websocket connection", func() {
				var err error
				Eventually(func() error {
					err = wsHelper.CloseConn()
					return err
				}).Should(HaveOccurred())

				Expect(err).To(MatchError(MatchRegexp(fmt.Sprintf("failed to close correctly after %d retries:.*", testMaxCloseRetryCount))))
				Expect(err).To(MatchError(MatchRegexp(".*close sent.*")))
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
			Eventually(pingPongChan, 5*time.Second, 100*time.Millisecond).Should(Receive(Equal(1)))
		})
		Context("when server is down", func() {
			BeforeEach(func() {
				fakeMetricServer.Close()
				wsh.CloseWSConnection()

			})
			It("fails to ping and reconnect", func() {
				var err error
				Eventually(func() error {
					err = wsHelper.Ping()
					return err
				}, "1s").Should(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(fmt.Sprintf(".*failed after %d retries:.*", testMaxCloseRetryCount))))
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
		Context("when server is down", Pending, func() {
			//TODO disabling due to flakeynes https://github.com/cloudfoundry/app-autoscaler-release/issues/1013
			BeforeEach(func() {
				fakeMetricServer.Close()
				wsh.CloseWSConnection()
			})
			It("fails to write envelops", func() {
				Eventually(func() error {
					err = wsHelper.Write(&testEnvelope)
					return err
				}).Should(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(fmt.Sprintf(".*failed after %d retries:.*", testMaxCloseRetryCount))))
			})
		})
	})
})
