package aggregator_test

import (
	. "eventgenerator/aggregator"
	. "eventgenerator/appmetric"
	. "eventgenerator/model"
	. "models"
	"net/http"
	"time"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("MetricPoller", func() {
	var testAppId string = "testAppId"
	var timestamp int64 = time.Now().UnixNano()
	var metricType string = "MemoryUsage"
	var logger lager.Logger
	var appChan chan *AppMonitor
	var metricPoller *MetricPoller
	var httpClient *http.Client
	var metricConsumer MetricConsumer
	var metricServer *ghttp.Server
	var metrics []*Metric = []*Metric{
		&Metric{
			Name:      metricType,
			Unit:      "bytes",
			AppId:     testAppId,
			TimeStamp: timestamp,
			Instances: []InstanceMetric{InstanceMetric{
				Timestamp: timestamp,
				Index:     0,
				Value:     "100",
			}, InstanceMetric{
				Timestamp: timestamp,
				Index:     1,
				Value:     "200",
			}},
		},
		&Metric{
			Name:      metricType,
			Unit:      "bytes",
			AppId:     testAppId,
			TimeStamp: timestamp,
			Instances: []InstanceMetric{InstanceMetric{
				Timestamp: timestamp,
				Index:     0,
				Value:     "300",
			}, InstanceMetric{
				Timestamp: timestamp,
				Index:     1,
				Value:     "400",
			}},
		},
	}

	BeforeEach(func() {
		logger = lager.NewLogger("MetricPoller-test")
		httpClient = cfhttp.NewClient()
		appChan = make(chan *AppMonitor, 1)
		metricConsumer = func(appMetric *AppMetric) {}
		metricServer = nil
	})
	Context("Start", func() {
		JustBeforeEach(func() {
			metricPoller = NewMetricPoller(metricServer.URL(), logger, appChan, metricConsumer, httpClient)
			metricPoller.Start()
			appChan <- &AppMonitor{
				AppId:      testAppId,
				MetricType: metricType,
				StatWindow: 10,
			}

		})
		AfterEach(func() {
			metricPoller.Stop()
			metricServer.Close()
		})
		Context("when the poller is started", func() {
			var consumed chan *AppMetric
			BeforeEach(func() {
				consumed = make(chan *AppMetric, 1)
				metricConsumer = func(appMetric *AppMetric) {
					consumed <- appMetric
				}
			})
			Context("when retrieve metrics from metric-collector successfully", func() {
				BeforeEach(func() {
					metricServer = ghttp.NewServer()
					metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusOK,
						&metrics))
				})
				It("should get the app's average metric", func() {
					var appMetric *AppMetric
					Eventually(consumed).Should(Receive(&appMetric))
					Expect(appMetric).To(Equal(&AppMetric{
						AppId:      testAppId,
						MetricType: metricType,
						Value:      250,
						Unit:       "bytes",
						Timestamp:  timestamp}))
				})
			})
			//the too long response will cause ioutil.ReadAll() to panic
			//slow test, will take a lot of seconds
			PContext("when the response body from metric-collector is too long", func() {
				BeforeEach(func() {
					var tooLargeMetrics, template []*Metric
					for i := 0; i < 9999; i++ {
						template = append(template, metrics...)
					}
					for i := 0; i < 999; i++ {
						tooLargeMetrics = append(tooLargeMetrics, template...)
					}
					metricServer = ghttp.NewServer()
					metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusOK,
						&tooLargeMetrics))
				})
				It("should cause io read error and should not do aggregation", func() {
					Consistently(consumed).ShouldNot(Receive())
				})
			})
			Context("when metric-collector returns an empty result", func() {
				BeforeEach(func() {
					metricServer = ghttp.NewServer()
					metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusOK,
						&[]*Metric{}))
				})
				It("should not do aggregation as there is no metric", func() {
					var appMetric *AppMetric
					Eventually(consumed).Should(Receive(&appMetric))
					Expect(appMetric).To(Equal(&AppMetric{
						AppId:      testAppId,
						MetricType: metricType,
						Value:      0,
						Unit:       "",
						Timestamp:  0,
					}))
				})
			})
			Context("when metric-collector returns error", func() {
				BeforeEach(func() {
					metricServer = ghttp.NewServer()
					metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusBadRequest,
						ErrorResponse{
							Code:    "Interal-Server-Error",
							Message: "Error"}))
				})
				It("should not do aggregation as there is no metric", func() {
					Consistently(consumed).ShouldNot(Receive())
				})
			})
			Context("when metric-collector is not running", func() {
				JustBeforeEach(func() {
					metricServer.Close()
				})
				BeforeEach(func() {
					metricServer = ghttp.NewServer()

				})
				It("should not do aggregation as there is no metric", func() {
					Consistently(consumed).ShouldNot(Receive())
				})
			})

		})
	})
	Context("Stop", func() {
		var consumed chan *AppMetric
		BeforeEach(func() {
			consumed = make(chan *AppMetric, 1)
			metricConsumer = func(appMetric *AppMetric) {
				consumed <- appMetric
			}
			metricServer = ghttp.NewServer()
			metricServer.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metrics_history/memory", ghttp.RespondWithJSONEncoded(http.StatusOK,
				&metrics))
			metricPoller = NewMetricPoller(metricServer.URL(), logger, appChan, metricConsumer, httpClient)
			metricPoller.Start()
			metricPoller.Stop()
			appChan <- &AppMonitor{
				AppId:      testAppId,
				MetricType: metricType,
				StatWindow: 10,
			}

		})
		It("stops the aggregating", func() {
			Consistently(consumed).ShouldNot(Receive())
		})
	})
})
