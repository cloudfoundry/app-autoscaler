package aggregator_test

import (
	. "autoscaler/eventgenerator/aggregator"
	"autoscaler/eventgenerator/aggregator/fakes"
	"autoscaler/models"
	"autoscaler/routes"
	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager/lagertest"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
	"net/http"
	"time"
)

var _ = Describe("MetricPoller", func() {
	var testAppId string = "testAppId"
	var timestamp int64 = time.Now().UnixNano()
	var metricType string = models.MetricNameMemory
	var logger *lagertest.TestLogger
	var appMonitorsChan chan *models.AppMonitor
	var appMetricDatabase *fakes.FakeAppMetricDB
	var metricPoller *MetricPoller
	var httpClient *http.Client
	var metricServer *ghttp.Server
	var metrics []*models.AppInstanceMetric = []*models.AppInstanceMetric{
		{
			AppId:         testAppId,
			InstanceIndex: 0,
			CollectedAt:   111111,
			Name:          metricType,
			Unit:          models.UnitMegaBytes,
			Value:         "100",
			Timestamp:     111100,
		},
		{
			AppId:         testAppId,
			InstanceIndex: 1,
			CollectedAt:   111111,
			Name:          metricType,
			Unit:          models.UnitMegaBytes,
			Value:         "200",
			Timestamp:     110000,
		},

		{
			AppId:         testAppId,
			InstanceIndex: 0,
			CollectedAt:   222222,
			Name:          metricType,
			Unit:          models.UnitMegaBytes,
			Value:         "300",
			Timestamp:     222200,
		},
		{
			AppId:         testAppId,
			InstanceIndex: 1,
			CollectedAt:   222222,
			Name:          metricType,
			Unit:          models.UnitMegaBytes,
			Value:         "400",
			Timestamp:     220000,
		},
	}
	var urlPath string
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("MetricPoller-test")
		httpClient = cfhttp.NewClient()
		appMonitorsChan = make(chan *models.AppMonitor, 1)
		appMetricDatabase = &fakes.FakeAppMetricDB{}
		metricServer = nil

		path, err := routes.MetricsCollectorRoutes().Get(routes.MemoryMetricHistoryRoute).URLPath("appid", testAppId)
		Expect(err).NotTo(HaveOccurred())
		urlPath = path.Path
	})

	Context("When metric-collector is not running", func() {
		var appMonitor *models.AppMonitor

		BeforeEach(func() {
			appMonitor = &models.AppMonitor{
				AppId:      testAppId,
				MetricType: metricType,
				StatWindow: 10,
			}

			metricServer = ghttp.NewUnstartedServer()
			metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
				&metrics))

			metricPoller = NewMetricPoller(logger, metricServer.URL(), appMonitorsChan, httpClient, appMetricDatabase)
			metricPoller.Start()

			appMonitorsChan <- appMonitor
		})

		AfterEach(func() {
			metricPoller.Stop()
			metricServer.Close()
		})

		It("logs an error", func() {
			Eventually(logger.Buffer).Should(Say("Failed to retrieve metric"))
		})

		It("does not save any metrics", func() {
			Consistently(appMetricDatabase.SaveAppMetricCallCount).Should(BeZero())
		})
	})

	Context("Start", func() {
		var appMonitor *models.AppMonitor

		BeforeEach(func() {
			appMonitor = &models.AppMonitor{
				AppId:      testAppId,
				MetricType: metricType,
				StatWindow: 10,
			}

			metricServer = ghttp.NewServer()
			metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
				&metrics))
		})

		JustBeforeEach(func() {
			metricPoller = NewMetricPoller(logger, metricServer.URL(), appMonitorsChan, httpClient, appMetricDatabase)
			metricPoller.Start()

			appMonitorsChan <- appMonitor
		})

		AfterEach(func() {
			metricPoller.Stop()
			metricServer.Close()
		})

		Context("with a non-memoryused type", func() {
			BeforeEach(func() {
				appMonitor.MetricType = "garbage"
			})

			It("logs an error", func() {
				Eventually(logger.Buffer).Should(Say("Unsupported metric type"))
			})

			It("does not save any metrics", func() {
				Consistently(appMetricDatabase.SaveAppMetricCallCount).Should(BeZero())
			})
		})

		Context("when metrics are successfully retrieved", func() {
			It("saves the average metrics", func() {
				Eventually(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(1))
				actualAppMetric := appMetricDatabase.SaveAppMetricArgsForCall(0)
				actualAppMetric.Timestamp = timestamp

				Expect(actualAppMetric).To(Equal(&models.AppMetric{
					AppId:      testAppId,
					MetricType: metricType,
					Value:      "250",
					Unit:       models.UnitMegaBytes,
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

			It("does not save any metrics", func() {
				Consistently(appMetricDatabase.SaveAppMetricCallCount).Should(BeZero())
			})
		})

		Context("when empty metrics are retrieved", func() {
			BeforeEach(func() {
				metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
					&[]*models.AppInstanceMetric{}))
			})

			It("saves the average metrics with no value", func() {
				Eventually(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(1))
				Expect(appMetricDatabase.SaveAppMetricArgsForCall(0).Value).To(BeEmpty())
			})
		})

		Context("when an error ocurrs retrieving metrics", func() {
			BeforeEach(func() {
				metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWithJSONEncoded(http.StatusBadRequest,
					models.ErrorResponse{
						Code:    "Interal-Server-Error",
						Message: "Error"}))
			})

			It("does not save any metrics", func() {
				Consistently(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(0))
			})
		})

		Context("when the save fails", func() {
			BeforeEach(func() {
				appMetricDatabase.SaveAppMetricReturns(errors.New("db error"))
			})

			It("logs an error", func() {
				Eventually(logger.Buffer).Should(Say("Failed to save"))
			})
		})
	})

	Context("Stop", func() {
		BeforeEach(func() {
			metricServer = ghttp.NewServer()
			metricServer.RouteToHandler("GET", urlPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
				&metrics))

			metricPoller = NewMetricPoller(logger, metricServer.URL(), appMonitorsChan, httpClient, appMetricDatabase)
			metricPoller.Start()
			metricPoller.Stop()

			appMonitorsChan <- &models.AppMonitor{
				AppId:      testAppId,
				MetricType: metricType,
				StatWindow: 10,
			}
		})

		It("stops the aggregating", func() {
			Consistently(appMetricDatabase.SaveAppMetricCallCount).Should(Or(Equal(0), Equal(1)))
		})
	})
})
