package client_test

import (
	"crypto/tls"
	"net/http"
	"path/filepath"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/envelopeprocessor"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/client"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache"
	"code.cloudfoundry.org/lager/lagertest"
)

const testCertDir = "../../../../test-certs"

var _ = Describe("MetricClientFactory", func() {
	var (
		conf                          config.Config
		metricClient                  MetricClient
		metricClientFactory           *MetricClientFactory
		fakeLogCacheClientCreator     fakes.FakeLogCacheClientCreator
		fakeMetricServerClientCreator fakes.FakeMetricServerClientCreator
		fakeEnvelopeProcessorCreator  fakes.FakeEnvelopeProcessorCreator
		fakeGoLogCacheClient          fakes.FakeGoLogCacheClient
		fakeGRPC                      fakes.FakeGrpcDialOptions
		fakeTLSConfig                 fakes.FakeTLSConfig
		expectedTLSLogCacheClient     logcache.Client
		expectedOauth2HTTPClient      *logcache.Oauth2HTTPClient
		expectedClientOption          logcache.ClientOption
		expectedHTTPClient            logcache.HTTPClient
		expectedOauth2HTTPClientOpt   logcache.Oauth2Option
		logger                        *lagertest.TestLogger
		metricCollectorURL            string
		tlsCerts                      models.TLSCerts
		uaaCreds                      models.UAACreds
		useLogCache                   bool
		caCertFilePath                string
		certFilePath                  string
		keyFilePath                   string
	)

	BeforeEach(func() {
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

		// Stub public go-log-cache functions
		GoLogCacheNewClient = fakeGoLogCacheClient.NewClient
		GoLogCacheNewOauth2HTTPClient = fakeGoLogCacheClient.NewOauth2HTTPClient
		GoLogCacheWithViaGRPC = fakeGoLogCacheClient.WithViaGRPC
		GoLogCacheWithHTTPClient = fakeGoLogCacheClient.WithHTTPClient
		GoLogCacheWithOauth2HTTPClient = fakeGoLogCacheClient.WithOauth2HTTPClient
		GRPCWithTransportCredentials = fakeGRPC.WithTransportCredentials

		NewTLS = fakeTLSConfig.NewTLS
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
				UseLogCache:        useLogCache,
				MetricCollectorURL: metricCollectorURL,
				TLSClientCerts:     tlsCerts,
				UAACreds:           uaaCreds,
			},
		}

		logger = lagertest.NewTestLogger("MetricServer")
		metricClient = metricClientFactory.GetMetricClient(logger, &conf)
	})
	Describe("GetMetricClient", func() {
		BeforeEach(func() {
			metricCollectorURL = "some-metric-server-url"
			tlsCerts = models.TLSCerts{
				KeyFile:    keyFilePath,
				CertFile:   certFilePath,
				CACertFile: caCertFilePath,
			}
		})

		Describe("when logCacheEnabled is false", func() {
			BeforeEach(func() {
				useLogCache = false
			})

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

			It("Should create a LogCacheClient and not a metricserver client", func() {
				Expect(metricClient).To(BeAssignableToTypeOf(&LogCacheClient{}))
				Expect(fakeMetricServerClientCreator.NewMetricServerClientCallCount()).To(Equal(0))
				Expect(fakeLogCacheClientCreator.NewLogCacheClientCallCount()).To(Equal(1))

				logger, now, actualLogCacheClient, actualEnvelopeProcessor := fakeLogCacheClientCreator.NewLogCacheClientArgsForCall(0)
				Expect(logger).NotTo(BeNil())
				Expect(fakeGoLogCacheClient.NewClientCallCount()).To(Equal(1))
				Expect(actualLogCacheClient).To(Equal(&expectedTLSLogCacheClient))
				Expect(actualEnvelopeProcessor).To(BeAssignableToTypeOf(envelopeprocessor.Processor{}))
				Expect(now).NotTo(BeNil())
			})

			It("Should provision logCacheClient with the correct tls configurations", func() {
				expectedTLSCreds, err := NewTLSConfig(caCertFilePath, certFilePath, keyFilePath)
				Expect(err).NotTo(HaveOccurred())

				actualAddrs, clientOptions := fakeGoLogCacheClient.NewClientArgsForCall(0)
				Expect(actualAddrs).To(Equal("some-log-cache-url:8080"))
				Expect(clientOptions).NotTo(BeNil())
				Expect(fakeGRPC.WithTransportCredentialsCallCount()).To(Equal(1))
				actualTLSCreds := fakeTLSConfig.NewTLSArgsForCall(0)
				Expect(actualTLSCreds.Certificates).To(Equal(expectedTLSCreds.Certificates))
			})

			Describe("when uaa client and secret are provided", func() {
				BeforeEach(func() {
					uaaCreds = models.UAACreds{
						URL:          "https:some-uaa",
						ClientID:     "some-id",
						ClientSecret: "some-secret",
					}
					tlsCerts = models.TLSCerts{}
					expectedOauth2HTTPClient = logcache.NewOauth2HTTPClient(uaaCreds.URL, uaaCreds.ClientID, uaaCreds.ClientSecret)
					fakeGoLogCacheClient.NewOauth2HTTPClientReturns(expectedOauth2HTTPClient)
					fakeGoLogCacheClient.WithHTTPClientReturns(expectedClientOption)
				})

				Describe("when skip_ssl_validation is enabled", func() {
					BeforeEach(func() {
						uaaCreds.SkipSSLValidation = true
						expectedHTTPClient = &http.Client{
							Timeout: 5 * time.Second,
							Transport: &http.Transport{
								//nolint:gosec
								TLSClientConfig: &tls.Config{InsecureSkipVerify: uaaCreds.SkipSSLValidation},
							},
						}
					})
					It("Should create a LogCacheClient Clientvia OauthHTTP", func() {
						_, _, _, actualNewOauth2HTTPClientOpts := fakeGoLogCacheClient.NewOauth2HTTPClientArgsForCall(0)
						Expect(reflect.ValueOf(actualNewOauth2HTTPClientOpts[0]).Pointer()).Should(Equal(reflect.ValueOf(expectedOauth2HTTPClientOpt).Pointer()))
						actualHttpClient := fakeGoLogCacheClient.WithOauth2HTTPClientArgsForCall(0)
						Expect(actualHttpClient).To(BeEquivalentTo(expectedHTTPClient))
					})
				})

				It("Should create a LogCacheClient via OauthHTTP", func() {
					Expect(fakeMetricServerClientCreator.NewMetricServerClientCallCount()).To(Equal(0))
					Expect(fakeLogCacheClientCreator.NewLogCacheClientCallCount()).To(Equal(1))

					By("Sending the right argument when creating the Oauth2HTTPClient")
					Expect(fakeGoLogCacheClient.NewOauth2HTTPClientCallCount()).To(Equal(1))
					uaaURL, uaaClientID, uaaClientSecret, oauthOpts := fakeGoLogCacheClient.NewOauth2HTTPClientArgsForCall(0)
					Expect(uaaURL).NotTo(BeNil())
					Expect(uaaClientID).NotTo(BeNil())
					Expect(uaaClientSecret).NotTo(BeNil())
					Expect(oauthOpts).NotTo(BeEmpty())

					By("Calling logcache.NewClient with an Oauth2HTTPClient as an option")
					Expect(fakeGoLogCacheClient.NewClientCallCount()).To(Equal(1))
					actualLogCacheAddrs, actualClientOptions := fakeGoLogCacheClient.NewClientArgsForCall(0)
					Expect(actualLogCacheAddrs).To(Equal(conf.MetricCollector.MetricCollectorURL))
					Expect(fakeGoLogCacheClient.WithHTTPClientCallCount()).To(Equal(1))
					actualHTTPClient := fakeGoLogCacheClient.WithHTTPClientArgsForCall(0)
					Expect(actualHTTPClient).NotTo(BeNil())
					Expect(actualHTTPClient).To(Equal(expectedOauth2HTTPClient))
					Expect(&actualClientOptions).NotTo(Equal(expectedClientOption))
				})
			})

			It("Should set AggregatorExecuteInterval as processor collectionInterval", func() {
				_, actualCollectionInterval := fakeEnvelopeProcessorCreator.NewProcessorArgsForCall(0)
				Expect(actualCollectionInterval).To(Equal(conf.Aggregator.AggregatorExecuteInterval))
			})

		})
	})
})
