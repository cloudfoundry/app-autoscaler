package client_test

import (
	"crypto/tls"
	"net/http"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/client"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	credentials "google.golang.org/grpc/credentials"
)

const testCertDir = "../../../../test-certs"

var _ = Describe("MetricClientFactory", func() {
	var (
		conf                           config.Config
		metricClient                   MetricClient
		metricClientFactory            *MetricClientFactory
		fakeLogCacheClientCreator      fakes.FakeLogCacheClientCreator
		fakeMetricServerClientCreator  fakes.FakeMetricServerClientCreator
		fakeEnvelopeProcessorCreator   fakes.FakeEnvelopeProcessorCreator
		fakeGoLogCacheClient           fakes.FakeGoLogCacheClient
		fakeGRPC                       fakes.FakeGrpcDialOptions
		fakeTLSConfig                  fakes.FakeTLSConfig
		expectedLogCacheClient         logcache.Client
		expectedTLSTransportCredential credentials.TransportCredentials
		expectedTLSCreds               *tls.Config
		logger                         *lagertest.TestLogger
		metricCollectorURL             string
		useLogCache                    bool
		caCertFilePath                 string
		certFilePath                   string
		keyFilePath                    string
	)

	BeforeEach(func() {
		var err error
		caCertFilePath = filepath.Join(testCertDir, "autoscaler-ca.crt")
		certFilePath = filepath.Join(testCertDir, "eventgenerator.crt")
		keyFilePath = filepath.Join(testCertDir, "eventgenerator.key")
		fakeLogCacheClientCreator = fakes.FakeLogCacheClientCreator{}
		fakeGoLogCacheClient = fakes.FakeGoLogCacheClient{}
		fakeMetricServerClientCreator = fakes.FakeMetricServerClientCreator{}
		fakeTLSConfig = fakes.FakeTLSConfig{}
		fakeGRPC = fakes.FakeGrpcDialOptions{}
		metricClientFactory = NewMetricClientFactory(fakeLogCacheClientCreator.NewLogCacheClient, fakeMetricServerClientCreator.NewMetricServerClient)
		NewProcessor = fakeEnvelopeProcessorCreator.NewProcessor
		NewGoLogCacheClient = fakeGoLogCacheClient.NewClient
		LogCacheClientWithGRPC = fakeGoLogCacheClient.WithViaGRPC
		GRPCWithTransportCredentials = fakeGRPC.WithTransportCredentials
		NewTLS = fakeTLSConfig.NewTLS

		fakeGoLogCacheClient.NewClientReturns(&expectedLogCacheClient)

		expectedTLSCreds, err = NewTLSConfig(caCertFilePath, certFilePath, keyFilePath)
		Expect(err).NotTo(HaveOccurred())

		expectedTLSTransportCredential = credentials.NewTLS(expectedTLSCreds.Clone())
		Expect(err).NotTo(HaveOccurred())

		fakeTLSConfig.NewTLSReturns(expectedTLSTransportCredential)
	})

	JustBeforeEach(func() {
		conf = config.Config{
			Aggregator: config.AggregatorConfig{
				AggregatorExecuteInterval: 51 * time.Second,
			},
			MetricCollector: config.MetricCollectorConfig{
				UseLogCache:        useLogCache,
				MetricCollectorURL: metricCollectorURL,
				TLSClientCerts: models.TLSCerts{
					KeyFile:    keyFilePath,
					CertFile:   certFilePath,
					CACertFile: caCertFilePath,
				},
			},
		}

		logger = lagertest.NewTestLogger("MetricServer")
		metricClient = metricClientFactory.GetMetricClient(logger, &conf)
	})
	Describe("GetMetricClient", func() {
		BeforeEach(func() {
			metricCollectorURL = "some-metric-server-url"
			useLogCache = false
		})

		Describe("when logCacheEnabled is false", func() {
			It("should create a MetricServerClient by default", func() {
				Expect(metricClient).To(BeAssignableToTypeOf(&MetricServerClient{}))
				Expect(fakeLogCacheClientCreator.NewLogCacheClientCallCount()).To(Equal(0))
				Expect(fakeMetricServerClientCreator.NewMetricServerClientCallCount()).To(Equal(1))
				logger, url, httpClient := fakeMetricServerClientCreator.NewMetricServerClientArgsForCall(0)
				Expect(logger).NotTo(BeNil())
				Expect(url).NotTo(BeNil())
				Expect(url).To(Equal("some-metric-server-url"))
				Expect(httpClient).NotTo(BeNil())
				Expect(httpClient).To(BeAssignableToTypeOf(&http.Client{}))
				Expect(httpClient.Transport.(*http.Transport).TLSClientConfig).NotTo(BeNil())
			})
		})

		Describe("when logCacheEnabled is true", func() {
			BeforeEach(func() {
				metricCollectorURL = "some-log-cache-url:8080"
				useLogCache = true
			})

			It("Should create a LogCacheClient", func() {
				Expect(metricClient).To(BeAssignableToTypeOf(&LogCacheClient{}))
				Expect(fakeMetricServerClientCreator.NewMetricServerClientCallCount()).To(Equal(0))
				Expect(fakeLogCacheClientCreator.NewLogCacheClientCallCount()).To(Equal(1))
				logger, now, actualLogCacheClient, actualEnvelopeProcessor := fakeLogCacheClientCreator.NewLogCacheClientArgsForCall(0)
				Expect(logger).NotTo(BeNil())
				Expect(fakeGoLogCacheClient.NewClientCallCount()).To(Equal(1))
				Expect(actualLogCacheClient).To(Equal(&expectedLogCacheClient))
				Expect(actualEnvelopeProcessor).To(BeAssignableToTypeOf(envelopeprocessor.Processor{}))
				Expect(now).NotTo(BeNil())
			})

			It("Should set AggregatorExecuteInterval as processor collectionInterval", func() {
				_, actualCollectionInterval := fakeEnvelopeProcessorCreator.NewProcessorArgsForCall(0)
				Expect(actualCollectionInterval).To(Equal(conf.Aggregator.AggregatorExecuteInterval))
			})

			It("Should provision tls configuration to the logCacheClient", func() {
				actualAddrs, clientOptions := fakeGoLogCacheClient.NewClientArgsForCall(0)
				Expect(actualAddrs).To(Equal("some-log-cache-url:8080"))
				Expect(clientOptions).NotTo(BeNil())
				Expect(fakeGRPC.WithTransportCredentialsCallCount()).To(Equal(1))
				actualTLSCreds := fakeTLSConfig.NewTLSArgsForCall(0)
				Expect(actualTLSCreds.Certificates).To(Equal(expectedTLSCreds.Certificates))
			})

		})
	})
})
