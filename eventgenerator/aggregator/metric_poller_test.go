package aggregator_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/client"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"
	cfhttp "code.cloudfoundry.org/cfhttp/v2"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"

	"net/http"
	"time"
)

var _ = Describe("MetricPoller", func() {
	var (
		testAppId       = "testAppId"
		timestamp       = time.Now().UnixNano()
		testMetricType  = "a-metric-type"
		testMetricUnit  = "a-metric-unit"
		logger          *lagertest.TestLogger
		appMonitorsChan chan *models.AppMonitor
		appMetricChan   chan *models.AppMetric
		metricPoller    *MetricPoller
		metricClient    MetricClient
		metricServer    *ghttp.Server
		metrics         = []*models.AppInstanceMetric{
			{
				AppId:         testAppId,
				InstanceIndex: 0,
				CollectedAt:   111111,
				Name:          testMetricType,
				Unit:          testMetricUnit,
				Value:         "100",
				Timestamp:     111100,
			},
			{
				AppId:         testAppId,
				InstanceIndex: 1,
				CollectedAt:   111111,
				Name:          testMetricType,
				Unit:          testMetricUnit,
				Value:         "200",
				Timestamp:     110000,
			},

			{
				AppId:         testAppId,
				InstanceIndex: 0,
				CollectedAt:   222222,
				Name:          testMetricType,
				Unit:          testMetricUnit,
				Value:         "300",
				Timestamp:     222200,
			},
			{
				AppId:         testAppId,
				InstanceIndex: 1,
				CollectedAt:   222222,
				Name:          testMetricType,
				Unit:          testMetricUnit,
				Value:         "401",
				Timestamp:     220000,
			},
		}
		urlPath    string
		appMonitor *models.AppMonitor
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("MetricPoller-test")
		//TODO https://github.com/cloudfoundry/app-autoscaler-release/issues/549

		appMonitorsChan = make(chan *models.AppMonitor, 1)
		appMetricChan = make(chan *models.AppMetric, 1)
		metricServer = nil

		path, err := routes.MetricsCollectorRoutes().Get(routes.GetMetricHistoriesRouteName).URLPath("appid", testAppId, "metrictype", testMetricType)
		Expect(err).NotTo(HaveOccurred())
		urlPath = path.Path
		appMonitor = &models.AppMonitor{
			AppId:      testAppId,
			MetricType: testMetricType,
			StatWindow: 10,
		}

	})

	Context("When metric-collector is not running", func() {

		BeforeEach(func() {
			metricServer = ghttp.NewUnstartedServer()

			httpClient := cfhttp.NewClient()
			metricClient = NewMetricServerClient(logger, metricServer.URL(), httpClient)
			metricPoller = NewMetricPoller(logger, metricClient, appMonitorsChan, appMetricChan)
			metricPoller.Start()

			Expect(appMonitorsChan).Should(BeSent(appMonitor))
		})

		AfterEach(func() {
			metricPoller.Stop()
			metricServer.Close()
		})

		It("logs an error", func() {
			Eventually(logger.Buffer).Should(Say("Failed to retrieve metric"))
		})

		It("does not save any metrics", func() {
			Consistently(appMetricChan).ShouldNot(Receive())
		})
	})

	Context("Start", func() {
		var appMetric *models.AppMetric

		BeforeEach(func() {

			metricServer = ghttp.NewServer()
			metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
				&metrics))

			httpClient := cfhttp.NewClient()
			metricClient = NewMetricServerClient(logger, metricServer.URL(), httpClient)
		})

		JustBeforeEach(func() {
			metricPoller = NewMetricPoller(logger, metricClient, appMonitorsChan, appMetricChan)
			metricPoller.Start()

			Expect(appMonitorsChan).Should(BeSent(appMonitor))
		})

		AfterEach(func() {
			metricPoller.Stop()
			metricServer.Close()
		})

		Context("when metrics are successfully retrieved", func() {
			It("send the average metrics to appMetric channel", func() {
				appMetric = <-appMetricChan
				appMetric.Timestamp = timestamp

				Expect(appMetric).To(Equal(&models.AppMetric{
					AppId:      testAppId,
					MetricType: testMetricType,
					Value:      "251",
					Unit:       testMetricUnit,
					Timestamp:  timestamp}))
			})
		})

		Context("when the metrics are not valid JSON", func() {
			BeforeEach(func() {
				metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWith(http.StatusOK,
					"{[}"))
			})

			It("logs an error", func() {
				Eventually(logger.Buffer).Should(Say("Failed to parse response"))
			})

			It("should not send any metrics to appmetric channel", func() {
				Consistently(appMetricChan).ShouldNot(Receive())
			})
		})

		Context("when empty metrics are retrieved", func() {
			BeforeEach(func() {
				metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
					&[]*models.AppInstanceMetric{}))
			})

			It("send the average metrics with no value to appmetric channel", func() {
				appMetric = <-appMetricChan
				appMetric.Timestamp = timestamp

				Expect(appMetric).To(Equal(&models.AppMetric{
					AppId:      testAppId,
					MetricType: testMetricType,
					Value:      "",
					Unit:       "",
					Timestamp:  timestamp}))
			})
		})

		Context("when an error ocurrs retrieving metrics", func() {
			BeforeEach(func() {
				metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest,
					models.ErrorResponse{
						Code:    "Internal-Server-Error",
						Message: "Error"}))
			})

			It("should not send any metrics to appmetric channel", func() {
				Consistently(appMetricChan).ShouldNot(Receive())
			})
		})

	})

	Context("Stop", func() {
		BeforeEach(func() {
			metricServer = ghttp.NewServer()
			metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
				&metrics))

			metricPoller = NewMetricPoller(logger, metricClient, appMonitorsChan, appMetricChan)
			metricPoller.Start()
			metricPoller.Stop()
			Eventually(logger.Buffer).Should(Say("stopped"))
			Expect(appMonitorsChan).Should(BeSent(appMonitor))
		})

		It("stops the aggregating", func() {
			Consistently(appMetricChan).ShouldNot(Receive())
		})
	})
})
