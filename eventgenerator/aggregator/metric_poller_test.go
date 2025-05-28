package aggregator_test

import (
	"errors"
	"path/filepath"

	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/metric"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	rpc "code.cloudfoundry.org/go-log-cache/v3/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
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
		metricFetcher   metric.Fetcher
		mockLogCache    *testhelpers.MockLogCache
		appMonitor      *models.AppMonitor
	)
	BeforeEach(func() {
		logger = lagertest.NewTestLogger("MetricPoller-test")
		//TODO https://github.com/cloudfoundry/app-autoscaler-release/issues/549

		appMonitorsChan = make(chan *models.AppMonitor, 1)
		appMetricChan = make(chan *models.AppMetric, 1)

		appMonitor = &models.AppMonitor{
			AppId:      testAppId,
			MetricType: testMetricType,
			StatWindow: 10,
		}

	})

	Context("When metric-collector is not running", func() {

		BeforeEach(func() {
			metricFetcher, err := metric.NewLogCacheFetcherFactory(metric.StandardLogCacheFetcherCreator).CreateFetcher(logger, config.Config{
				Aggregator: &config.AggregatorConfig{},
				MetricCollector: config.MetricCollectorConfig{
					MetricCollectorURL: "this.endpoint.does.not.exist:1234",
				},
			})
			Expect(err).ToNot(HaveOccurred())

			metricPoller = NewMetricPoller(logger, metricFetcher, appMonitorsChan, appMetricChan)
			metricPoller.Start()

			Expect(appMonitorsChan).Should(BeSent(appMonitor))
		})

		AfterEach(func() {
			metricPoller.Stop()
		})

		It("logs an error", func() {
			//TODO this should be a prometheus counter not a log statement check
			Eventually(logger.Buffer, 2*time.Second).Should(Say("retrieveMetric Failed"))

		})

		It("does not save any metrics", func() {
			Consistently(appMetricChan).ShouldNot(Receive())
		})
	})

	Context("Start", func() {
		var appMetric *models.AppMetric

		BeforeEach(func() {
			testCertDir := testhelpers.TestCertFolder()
			tlsConfig, err := testhelpers.NewTLSConfig(
				filepath.Join(testCertDir, "autoscaler-ca.crt"),
				filepath.Join(testCertDir, "log-cache.crt"),
				filepath.Join(testCertDir, "log-cache.key"),
				"log-cache",
			)
			Expect(err).ToNot(HaveOccurred())

			mockLogCache = testhelpers.NewMockLogCache(tlsConfig)
			mockLogCache.ReadReturns(testAppId, &rpc.ReadResponse{
				Envelopes: &loggregator_v2.EnvelopeBatch{
					Batch: []*loggregator_v2.Envelope{
						{
							SourceId:   testAppId,
							InstanceId: "0",
							Timestamp:  111100,
							DeprecatedTags: map[string]*loggregator_v2.Value{
								"origin": {
									Data: &loggregator_v2.Value_Text{
										Text: "autoscaler_metrics_forwarder",
									},
								},
							},
							Message: &loggregator_v2.Envelope_Gauge{
								Gauge: &loggregator_v2.Gauge{
									Metrics: map[string]*loggregator_v2.GaugeValue{
										testMetricType: {
											Unit:  testMetricUnit,
											Value: 100,
										},
									},
								},
							},
						},
						{
							SourceId:   testAppId,
							InstanceId: "1",
							Timestamp:  110000,
							DeprecatedTags: map[string]*loggregator_v2.Value{
								"origin": {
									Data: &loggregator_v2.Value_Text{
										Text: "autoscaler_metrics_forwarder",
									},
								},
							},
							Message: &loggregator_v2.Envelope_Gauge{
								Gauge: &loggregator_v2.Gauge{
									Metrics: map[string]*loggregator_v2.GaugeValue{
										testMetricType: {
											Unit:  testMetricUnit,
											Value: 200,
										},
									},
								},
							},
						},
						{
							SourceId:   testAppId,
							InstanceId: "0",
							Timestamp:  222200,
							DeprecatedTags: map[string]*loggregator_v2.Value{
								"origin": {
									Data: &loggregator_v2.Value_Text{
										Text: "autoscaler_metrics_forwarder",
									},
								},
							},
							Message: &loggregator_v2.Envelope_Gauge{
								Gauge: &loggregator_v2.Gauge{
									Metrics: map[string]*loggregator_v2.GaugeValue{
										testMetricType: {
											Unit:  testMetricUnit,
											Value: 300,
										},
									},
								},
							},
						},
						{
							SourceId:   testAppId,
							InstanceId: "1",
							Timestamp:  220000,
							DeprecatedTags: map[string]*loggregator_v2.Value{
								"origin": {
									Data: &loggregator_v2.Value_Text{
										Text: "autoscaler_metrics_forwarder",
									},
								},
							},
							Message: &loggregator_v2.Envelope_Gauge{
								Gauge: &loggregator_v2.Gauge{
									Metrics: map[string]*loggregator_v2.GaugeValue{
										testMetricType: {
											Unit:  testMetricUnit,
											Value: 401,
										},
									},
								},
							},
						},
					},
				},
			}, nil)
			err = mockLogCache.Start(3000 + GinkgoParallelProcess())
			Expect(err).ToNot(HaveOccurred())

			metricFetcher, err = metric.NewLogCacheFetcherFactory(metric.StandardLogCacheFetcherCreator).CreateFetcher(logger, config.Config{
				Aggregator: &config.AggregatorConfig{},
				MetricCollector: config.MetricCollectorConfig{
					MetricCollectorURL: mockLogCache.URL(),
					TLSClientCerts: models.TLSCerts{
						KeyFile:    filepath.Join(testCertDir, "log-cache.key"),
						CertFile:   filepath.Join(testCertDir, "log-cache.crt"),
						CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			metricPoller = NewMetricPoller(logger, metricFetcher, appMonitorsChan, appMetricChan)
			metricPoller.Start()

			Expect(appMonitorsChan).Should(BeSent(appMonitor))
		})

		AfterEach(func() {
			metricPoller.Stop()
			mockLogCache.Stop()
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

		Context("when an error occurs during metric retrieval", func() {
			BeforeEach(func() {
				mockLogCache.ReadReturns(testAppId, &rpc.ReadResponse{}, errors.New("error"))
			})

			It("logs an error", func() {
				Eventually(logger.Buffer).Should(Say("retrieveMetric Failed"))
			})

			It("should not send any metrics to appmetric channel", func() {
				Consistently(appMetricChan).ShouldNot(Receive())
			})
		})

		Context("when empty metrics are retrieved", func() {
			BeforeEach(func() {
				mockLogCache.ReadReturns(testAppId, &rpc.ReadResponse{
					Envelopes: &loggregator_v2.EnvelopeBatch{
						Batch: []*loggregator_v2.Envelope{},
					},
				}, nil)
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
	})

	Context("Stop", func() {
		BeforeEach(func() {
			metricPoller = NewMetricPoller(logger, metricFetcher, appMonitorsChan, appMetricChan)
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
