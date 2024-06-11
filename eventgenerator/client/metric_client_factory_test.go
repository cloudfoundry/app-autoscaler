package client_test

import (
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/client"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache/v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
)

const testCertDir = "../../../../test-certs"

var _ = Describe("MetricClientFactory", func() {
	var (
		conf                         config.Config
		metricClient                 MetricClient
		metricClientFactory          *MetricClientFactory
		fakeEnvelopeProcessorCreator fakes.FakeEnvelopeProcessorCreator
		fakeGoLogCacheClient         fakes.FakeGoLogCacheClient
		expectedTLSLogCacheClient    logcache.Client
		expectedHTTPClient           logcache.HTTPClient
		expectedOauth2HTTPClientOpt  logcache.Oauth2Option
		logger                       *lagertest.TestLogger
		expectedMetricCollectorURL   string
		tlsCerts                     models.TLSCerts
		uaaCreds                     models.UAACreds
		caCertFilePath               string
		certFilePath                 string
		keyFilePath                  string
	)

	BeforeEach(func() {
		caCertFilePath = filepath.Join(testCertDir, "autoscaler-ca.crt")
		certFilePath = filepath.Join(testCertDir, "eventgenerator.crt")
		keyFilePath = filepath.Join(testCertDir, "eventgenerator.key")
		fakeGoLogCacheClient = fakes.FakeGoLogCacheClient{}
		metricClientFactory = NewMetricClientFactory()
		NewProcessor = fakeEnvelopeProcessorCreator.NewProcessor
		fakeGoLogCacheClient.NewClientReturns(&expectedTLSLogCacheClient)
		expectedOauth2HTTPClientOpt = logcache.WithOauth2HTTPClient(expectedHTTPClient)
		fakeGoLogCacheClient.WithOauth2HTTPClientReturns(expectedOauth2HTTPClientOpt)
	})

	JustBeforeEach(func() {
		conf = config.Config{
			Aggregator: config.AggregatorConfig{
				AggregatorExecuteInterval: 51 * time.Second,
			},
			MetricCollector: config.MetricCollectorConfig{
				MetricCollectorURL: expectedMetricCollectorURL,
				TLSClientCerts:     tlsCerts,
				UAACreds:           uaaCreds,
			},
		}

		logger = lagertest.NewTestLogger("LogCache")
		metricClient = metricClientFactory.GetMetricClient(logger, &conf)
	})
	Describe("GetMetricClient", func() {
		BeforeEach(func() {
			expectedMetricCollectorURL = "some-log-cache-url:8080"
			tlsCerts = models.TLSCerts{
				KeyFile:    keyFilePath,
				CertFile:   certFilePath,
				CACertFile: caCertFilePath,
			}
		})

		It("Should create a LogCacheClient", func() {
			Expect(metricClient).To(BeAssignableToTypeOf(&LogCacheClient{}))
			actualURL := metricClient.(*LogCacheClient).GetUrl()
			Expect(actualURL).To(Equal(expectedMetricCollectorURL))
		})

		Describe("when uaa client and secret are not provided", func() {
			BeforeEach(func() {
				uaaCreds = models.UAACreds{}
			})

			It("Should set TLSConfig from config opts", func() {
				expectedTlSCreds := &models.TLSCerts{KeyFile: keyFilePath, CertFile: certFilePath, CACertFile: caCertFilePath}
				expectedTLSConfig, err := expectedTlSCreds.CreateClientConfig()
				Expect(err).NotTo(HaveOccurred())
				actualTLSConfig := metricClient.(*LogCacheClient).GetTlsConfig()
				Expect(actualTLSConfig.Certificates).To(Equal(expectedTLSConfig.Certificates))
			})
		})

		Describe("when uaa client and secret are provided", func() {
			BeforeEach(func() {
				uaaCreds = models.UAACreds{
					URL:          "log-cache.some-url.com",
					ClientID:     "some-id",
					ClientSecret: "some-secret",
				}
			})

			It("should set uaa creds from config", func() {
				actualUAACreds := metricClient.(*LogCacheClient).GetUAACreds()
				Expect(actualUAACreds).To(Equal(uaaCreds))
			})
		})

		It("Should set AggregatorExecuteInterval as collectionInterval on LogCacheClient", func() {
			Expect(metricClient.(*LogCacheClient).CollectionInterval()).To(Equal(conf.Aggregator.AggregatorExecuteInterval))
		})
	})
})
