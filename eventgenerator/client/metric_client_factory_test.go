package client_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/client"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache"
	"code.cloudfoundry.org/lager/lagertest"
	"crypto/tls"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	credentials "google.golang.org/grpc/credentials"
	"net/http"
	"path/filepath"
)

const testCertDir = "../../../../test-certs"

var _ = Describe("MetricClientFactory", func() {
	var (
		conf                           config.Config
		metricClient                   MetricClient
		metricClientFactory            *MetricClientFactory
		fakeLogCacheClientCreator      fakes.FakeLogCacheClientCreator
		fakeMetricServerClientCreator  fakes.FakeMetricServerClientCreator
		fakeGoLogCacheClient           fakes.FakeGoLogCacheClient
		fakeGRPC                       fakes.FakeGrpcDialOptions
		fakeTLSConfig                  fakes.FakeTLSConfig
		expectedLogCacheClient         logcache.Client
		expectedTLSTransportCredential credentials.TransportCredentials
		expectedTLSCreds               tls.Config
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
		//	expectedTLSConfig, err := NewTLSConfig(caCertFilePath, certFilePath, keyFilePath)
	})

	JustBeforeEach(func() {
		conf = config.Config{
			UseLogCache: useLogCache,
			MetricCollector: config.MetricCollectorConfig{
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
	Describe("GetMetricServer", func() {
		BeforeEach(func() {
			metricCollectorURL = "some-metric-server-url"
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

			Context("when logCache is enabled", func() {
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

					By("Provision tls configuration to the logCacheClient")
					actualAddrs, clientOptions := fakeGoLogCacheClient.NewClientArgsForCall(0)
					Expect(actualAddrs).To(Equal("some-log-cache-url:8080"))
					Expect(clientOptions).NotTo(BeNil())
					Expect(fakeGRPC.WithTransportCredentialsCallCount()).To(Equal(1))
					actualTLSCreds := fakeTLSConfig.NewTLSArgsForCall(0)
					Expect(actualTLSCreds.Certificates).To(Equal(expectedTLSCreds.Certificates))

					actualTransportCreds := fakeGRPC.WithTransportCredentialsArgsForCall(0)
					Expect(actualTransportCreds).To(Equal(expectedTLSTransportCredential))
				})
			})
		})
	})
})
