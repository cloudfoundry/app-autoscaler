package aggregator_test

import (
	. "autoscaler/eventgenerator/aggregator"
	"autoscaler/eventgenerator/aggregator/fakes"
	. "autoscaler/eventgenerator/model"
	"autoscaler/models"
	"errors"
	"net/http"
	"time"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("MetricPoller", func() {
	var testAppId string = "testAppId"
	var timestamp int64 = time.Now().UnixNano()
	var metricType string = "MemoryUsage"
	var logger *lagertest.TestLogger
	var appMonitorsChan chan *AppMonitor
	var appMetricDatabase *fakes.FakeAppMetricDB
	var metricPoller *MetricPoller
	var httpClient *http.Client
	var metricServer *ghttp.Server
	var metrics []*models.AppInstanceMetric = []*models.AppInstanceMetric{
		&models.AppInstanceMetric{
			AppId:         testAppId,
			InstanceIndex: 0,
			CollectedAt:   111111,
			Name:          metricType,
			Unit:          models.UnitBytes,
			Value:         "100",
			Timestamp:     111100,
		},
		&models.AppInstanceMetric{
			AppId:         testAppId,
			InstanceIndex: 1,
			CollectedAt:   111111,
			Name:          metricType,
			Unit:          models.UnitBytes,
			Value:         "200",
			Timestamp:     110000,
		},

		&models.AppInstanceMetric{
			AppId:         testAppId,
			InstanceIndex: 0,
			CollectedAt:   222222,
			Name:          metricType,
			Unit:          models.UnitBytes,
			Value:         "300",
			Timestamp:     222200,
		},
		&models.AppInstanceMetric{
			AppId:         testAppId,
			InstanceIndex: 1,
			CollectedAt:   222222,
			Name:          metricType,
			Unit:          models.UnitBytes,
			Value:         "400",
			Timestamp:     220000,
		},
	}

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("MetricPoller-test")
		httpClient = cfhttp.NewClient()
		appMonitorsChan = make(chan *AppMonitor, 1)
		appMetricDatabase = &fakes.FakeAppMetricDB{}
		metricServer = nil
	})

	Context("Start", func() {
		var appMonitor *AppMonitor

		BeforeEach(func() {
			appMonitor = &AppMonitor{
				AppId:      testAppId,
				MetricType: metricType,
				StatWindow: 10,
			}

			metricServer = ghttp.NewServer()
			metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusOK,
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

		Context("with a non-MemoryUsage type", func() {
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

				var value int64 = 250
				Expect(actualAppMetric).To(Equal(&AppMetric{
					AppId:      testAppId,
					MetricType: metricType,
					Value:      &value,
					Unit:       "bytes",
					Timestamp:  timestamp}))
			})
		})

		Context("when the metrics are not valid JSON", func() {
			BeforeEach(func() {
				metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWith(http.StatusOK,
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
				metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusOK,
					&[]*models.AppInstanceMetric{}))
			})

			It("saves the average metrics with no value", func() {
				Eventually(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(1))
				Expect(appMetricDatabase.SaveAppMetricArgsForCall(0).Value).To(BeNil())
			})
		})

		Context("when an error ocurrs retrieving metrics", func() {
			BeforeEach(func() {
				metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusBadRequest,
					models.ErrorResponse{
						Code:    "Interal-Server-Error",
						Message: "Error"}))
			})

			It("does not save any metrics", func() {
				Consistently(appMetricDatabase.SaveAppMetricCallCount).Should(Equal(0))
			})
		})

		Context("when metric-collector is not running", func() {
			JustBeforeEach(func() {
				metricServer.Close()
			})

			It("logs an error", func() {
				Eventually(logger.Buffer).Should(Say("Failed to retrieve metric"))
			})

			It("does not save any metrics", func() {
				Consistently(appMetricDatabase.SaveAppMetricCallCount).Should(BeZero())
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
			metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusOK,
				&metrics))

			metricPoller = NewMetricPoller(logger, metricServer.URL(), appMonitorsChan, httpClient, appMetricDatabase)
			metricPoller.Start()
			metricPoller.Stop()

			appMonitorsChan <- &AppMonitor{
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
