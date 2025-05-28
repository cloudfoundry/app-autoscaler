package metric_test

import (
	"crypto/tls"
	"net/http"
	"path/filepath"
	"reflect"
	"time"
	"unsafe"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/metric"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	logcache "code.cloudfoundry.org/go-log-cache/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("logCacheFetcherFactory", func() {

	var (
		testLogger *lagertest.TestLogger
		conf       config.Config

		mockLogCacheMetricFetcherCreator *fakes.FakeLogCacheFetcherCreator
		mockMetricFetcher                *fakes.FakeFetcher
		metricFetcherFactory             metric.FetcherFactory
	)

	BeforeEach(func() {
		testLogger = lagertest.NewTestLogger("testLogger")

		mockLogCacheMetricFetcherCreator = &fakes.FakeLogCacheFetcherCreator{}
		mockMetricFetcher = &fakes.FakeFetcher{}

		metricFetcherFactory = metric.NewLogCacheFetcherFactory(mockLogCacheMetricFetcherCreator)
	})

	Describe("CreateFetcher", func() {
		When("UAACreds are configured", func() {
			BeforeEach(func() {
				conf = config.Config{
					Aggregator: &config.AggregatorConfig{
						AggregatorExecuteInterval: 40 * time.Second,
					},
					MetricCollector: config.MetricCollectorConfig{
						MetricCollectorURL: "foo",
						UAACreds: models.UAACreds{
							URL:               "foo",
							ClientSecret:      "foo",
							ClientID:          "foo",
							SkipSSLValidation: true,
						},
					},
				}
			})

			It("creates a log cache client that uses an HTTP-client", func() {
				expectedLogCacheClient := logcache.NewClient(
					conf.MetricCollector.MetricCollectorURL,
					logcache.WithHTTPClient(
						logcache.NewOauth2HTTPClient(
							conf.MetricCollector.UAACreds.URL,
							conf.MetricCollector.UAACreds.ClientID,
							conf.MetricCollector.UAACreds.ClientSecret,
							logcache.WithOauth2HTTPClient(&http.Client{
								Timeout: 5 * time.Second,
								Transport: &http.Transport{
									TLSClientConfig: &tls.Config{
										// #nosec G402
										InsecureSkipVerify: conf.MetricCollector.UAACreds.SkipSSLValidation,
									},
								},
							}),
						),
					),
				)
				mockLogCacheMetricFetcherCreator.NewLogCacheFetcherReturns(mockMetricFetcher)

				metricFetcher, err := metricFetcherFactory.CreateFetcher(testLogger, conf)

				Expect(err).ToNot(HaveOccurred())
				Expect(metricFetcher).To(Equal(mockMetricFetcher))
				Expect(mockLogCacheMetricFetcherCreator.NewLogCacheFetcherCallCount()).To(Equal(1))
				logger, logCacheClient, envelopeProcessor, collectionInterval := mockLogCacheMetricFetcherCreator.NewLogCacheFetcherArgsForCall(0)
				Expect(logger).To(Equal(testLogger))
				Expect(logCacheClient).To(Equal(expectedLogCacheClient))
				Expect(envelopeProcessor).ToNot(BeNil())
				Expect(collectionInterval).To(Equal(conf.Aggregator.AggregatorExecuteInterval))
			})
		})

		When("no UAACreds are configured but TLSClientCerts are there", func() {
			BeforeEach(func() {
				testCertDir := testhelpers.TestCertFolder()
				conf = config.Config{
					Aggregator: &config.AggregatorConfig{
						AggregatorExecuteInterval: 40 * time.Second,
					},
					MetricCollector: config.MetricCollectorConfig{
						MetricCollectorURL: "foo",
						TLSClientCerts: models.TLSCerts{
							CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
							CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
							KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
						},
					},
				}
			})

			It("creates a log cache client that uses a GRPC-client", func() {
				mockLogCacheMetricFetcherCreator.NewLogCacheFetcherReturns(mockMetricFetcher)

				metricFetcher, err := metricFetcherFactory.CreateFetcher(testLogger, conf)

				Expect(err).ToNot(HaveOccurred())
				Expect(metricFetcher).To(Equal(mockMetricFetcher))
				Expect(mockLogCacheMetricFetcherCreator.NewLogCacheFetcherCallCount()).To(Equal(1))
				logger, logCacheClient, envelopeProcessor, collectionInterval := mockLogCacheMetricFetcherCreator.NewLogCacheFetcherArgsForCall(0)
				Expect(logger).To(Equal(testLogger))
				Expect(getUnexportedField("grpcClient", logCacheClient)).ToNot(BeNil())
				Expect(envelopeProcessor).ToNot(BeNil())
				Expect(collectionInterval).To(Equal(conf.Aggregator.AggregatorExecuteInterval))
			})
		})
	})
})

func getUnexportedField(name string, client metric.LogCacheClient) interface{} {
	field := reflect.ValueOf(client).Elem().FieldByName(name)
	// #nosec G115 -- test code
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}
