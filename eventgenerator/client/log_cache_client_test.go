package client_test

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"code.cloudfoundry.org/go-log-cache/v2/rpc/logcache_v1"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	logcache "code.cloudfoundry.org/go-log-cache/v2"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/client"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogCacheClient", func() {
	var (
		fakeEnvelopeProcessor   *fakes.FakeEnvelopeProcessor
		fakeGoLogCacheReader    *fakes.FakeLogCacheClientReader
		fakeGoLogCacheClient    *fakes.FakeGoLogCacheClient
		fakeGRPC                *fakes.FakeGRPCOptions
		appId                   string
		logger                  *lagertest.TestLogger
		logCacheClient          *LogCacheClient
		envelopes               []*loggregator_v2.Envelope
		metrics                 []models.AppInstanceMetric
		startTime               time.Time
		endTime                 time.Time
		collectedAt             time.Time
		logCacheClientReadError error
		expectedClientOption    logcache.ClientOption
	)

	BeforeEach(func() {
		fakeEnvelopeProcessor = &fakes.FakeEnvelopeProcessor{}
		fakeGoLogCacheReader = &fakes.FakeLogCacheClientReader{}
		fakeGoLogCacheClient = &fakes.FakeGoLogCacheClient{}
		fakeGRPC = &fakes.FakeGRPCOptions{}

		logCacheClientReadError = nil
		logger = lagertest.NewTestLogger("MetricPoller-test")
		startTime = time.Now()
		endTime = startTime.Add(-60 * time.Second)
		collectedAt = startTime.Add(10 * time.Millisecond)
		appId = "some-app-id"
		envelopes = []*loggregator_v2.Envelope{
			{SourceId: "some-id"},
		}

		metrics = []models.AppInstanceMetric{
			{AppId: "some-id"},
		}

		logCacheClient = NewLogCacheClient(logger, func() time.Time { return collectedAt }, 40*time.Second, fakeEnvelopeProcessor, "")
	})

	JustBeforeEach(func() {
		fakeGoLogCacheReader.ReadReturns(envelopes, logCacheClientReadError)
		fakeEnvelopeProcessor.GetGaugeMetricsReturnsOnCall(0, metrics, nil)
		fakeEnvelopeProcessor.GetGaugeMetricsReturnsOnCall(1, nil, errors.New("some error"))

		fakeGoLogCacheClient.WithViaGRPCReturns(expectedClientOption)

		goLogCache := GoLogCache{
			NewClient:   fakeGoLogCacheClient.NewClient,
			WithViaGRPC: fakeGoLogCacheClient.WithViaGRPC,

			WithHTTPClient:       fakeGoLogCacheClient.WithHTTPClient,
			NewOauth2HTTPClient:  fakeGoLogCacheClient.NewOauth2HTTPClient,
			WithOauth2HTTPClient: fakeGoLogCacheClient.WithOauth2HTTPClient,
		}

		logCacheClient.SetGoLogCache(goLogCache)
		logCacheClient.Configure()
	})

	Context("NewLogCacheClient", func() {
		var expectedAddrs string

		BeforeEach(func() {
			expectedAddrs = "logcache:8080"
			logCacheClient = NewLogCacheClient(logger, func() time.Time { return collectedAt }, 40*time.Second, fakeEnvelopeProcessor, expectedAddrs)
		})

		Context("when consuming log cache via grpc/mtls", func() {
			var (
				expectedTransportCredential credentials.TransportCredentials
				expectedDialOpt             gogrpc.DialOption
				caCertFilePath              string
				certFilePath                string
				keyFilePath                 string
			)

			BeforeEach(func() {
				caCertFilePath = filepath.Join(testCertDir, "autoscaler-ca.crt")
				certFilePath = filepath.Join(testCertDir, "eventgenerator.crt")
				keyFilePath = filepath.Join(testCertDir, "eventgenerator.key")

				expectedTlSCerts := &models.TLSCerts{KeyFile: keyFilePath, CertFile: certFilePath, CACertFile: caCertFilePath}
				expectedTLSConfig, err := expectedTlSCerts.CreateClientConfig()
				logCacheClient.SetTLSConfig(expectedTLSConfig)
				expectedTransportCredential = credentials.NewTLS(expectedTLSConfig)
				expectedDialOpt = gogrpc.WithTransportCredentials(expectedTransportCredential)
				expectedClientOption = logcache.WithViaGRPC(expectedDialOpt)
				fakeGRPC.WithTransportCredentialsReturns(expectedDialOpt)
				Expect(err).NotTo(HaveOccurred())

				grpc := GRPC{
					WithTransportCredentials: fakeGRPC.WithTransportCredentials,
				}
				logCacheClient.SetGRPC(grpc)
			})

			JustBeforeEach(func() {
				fakeGoLogCacheClient.WithViaGRPCReturns(expectedClientOption)
			})

			It("Should setup correct tls configurations", func() {
				actualAddrs, actualClientOptions := fakeGoLogCacheClient.NewClientArgsForCall(0)

				By("Creating the go log cache client with the correct params")
				Expect(actualAddrs).To(Equal(expectedAddrs))
				Expect(actualClientOptions).NotTo(BeEmpty())
				Expect(reflect.ValueOf(actualClientOptions[0]).Pointer()).To(Equal(reflect.ValueOf(expectedClientOption).Pointer()))

				By("Configuring GRPC client options to the logcache client")
				actualGRPCDialOpts := fakeGoLogCacheClient.WithViaGRPCArgsForCall(0)
				Expect(actualGRPCDialOpts).NotTo(BeEmpty())
				Expect(reflect.ValueOf(actualGRPCDialOpts[0]).Pointer()).To(Equal(reflect.ValueOf(expectedDialOpt).Pointer()))

				By("Sending the right transport credentials to the logcache client")
				Expect(fakeGRPC.WithTransportCredentialsCallCount()).To(Equal(1))
				actualTransportCredentials := fakeGRPC.WithTransportCredentialsArgsForCall(0)
				Expect(actualTransportCredentials).To(Equal(expectedTransportCredential))
			})
		})

		Describe("when consuming log cache via uaa/oauth", func() {
			var (
				uaaCreds                    models.UAACreds
				expectedHTTPClient          *http.Client
				expectedOauth2HTTPClient    *logcache.Oauth2HTTPClient
				expectedOauth2HTTPClientOpt logcache.Oauth2Option
			)

			BeforeEach(func() {
				uaaCreds = models.UAACreds{
					URL:          "https:some-uaa",
					ClientID:     "some-id",
					ClientSecret: "some-secret",
				}

				expectedHTTPClient = &http.Client{
					Timeout: 5 * time.Second,
					Transport: &http.Transport{
						//nolint:gosec
						TLSClientConfig: &tls.Config{InsecureSkipVerify: uaaCreds.SkipSSLValidation},
					},
				}
				logCacheClient.SetUAACreds(uaaCreds)
				expectedOauth2HTTPClient = &logcache.Oauth2HTTPClient{}
				expectedOauth2HTTPClientOpt = logcache.WithOauth2HTTPClient(expectedHTTPClient)
				fakeGoLogCacheClient.NewOauth2HTTPClientReturns(expectedOauth2HTTPClient)
				fakeGoLogCacheClient.WithOauth2HTTPClientReturns(expectedOauth2HTTPClientOpt)
				fakeGoLogCacheClient.WithHTTPClientReturns(expectedClientOption)
			})

			Describe("when skip_ssl_validation is enabled", func() {
				BeforeEach(func() {
					uaaCreds.SkipSSLValidation = true
				})

				It("Should create a LogCacheClient Clientvia OauthHTTP", func() {
					_, _, _, actualNewOauth2HTTPClientOpts := fakeGoLogCacheClient.NewOauth2HTTPClientArgsForCall(0)
					Expect(reflect.ValueOf(actualNewOauth2HTTPClientOpts[0]).Pointer()).Should(Equal(reflect.ValueOf(expectedOauth2HTTPClientOpt).Pointer()))
					actualHttpClient := fakeGoLogCacheClient.WithOauth2HTTPClientArgsForCall(0)
					Expect(actualHttpClient).To(Equal(expectedHTTPClient))
				})
			})

			It("Should create a LogCacheClient via OauthHTTP", func() {
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
				Expect(actualLogCacheAddrs).To(Equal(expectedAddrs))
				Expect(fakeGoLogCacheClient.WithHTTPClientCallCount()).To(Equal(1))
				actualHTTPClient := fakeGoLogCacheClient.WithHTTPClientArgsForCall(0)
				Expect(actualHTTPClient).NotTo(BeNil())
				Expect(actualHTTPClient).To(Equal(expectedOauth2HTTPClient))
				Expect(&actualClientOptions).NotTo(Equal(expectedClientOption))
			})
		})
	})

	Describe("CollectionInterval", func() {
		It("returns correct collection interval", func() {
			logCacheClient = NewLogCacheClient(logger, func() time.Time { return time.Now() }, 40*time.Second, fakeEnvelopeProcessor, "url")
			Expect(logCacheClient.CollectionInterval()).To(Equal(40 * time.Second))
		})
	})

	Context("GetMetrics", func() {
		JustBeforeEach(func() {
			logCacheClient.Client = fakeGoLogCacheReader
		})

		Describe("when log cache returns error on read", func() {
			BeforeEach(func() {
				logCacheClientReadError = errors.New("some Read error")
			})

			It("return error", func() {
				_, err := logCacheClient.GetMetrics(appId, models.MetricNameMemoryUtil, startTime, endTime)
				Expect(err).To(HaveOccurred())
			})
		})

		When("errors occur reading via PromQL API", func() {
			When("PromQL call fails", func() {
				It("returns an error", func() {
					fakeGoLogCacheReader.PromQLReturns(nil, errors.New("fail"))
					_, err := logCacheClient.GetMetrics("app-id", "throughput", startTime, endTime)
					Expect(err).To(HaveOccurred())
				})
			})

			When("PromQL result is not a vector", func() {
				It("returns an error", func() {
					fakeGoLogCacheReader.PromQLReturns(nil, nil)
					_, err := logCacheClient.GetMetrics("app-id", "throughput", startTime, endTime)
					Expect(err).To(HaveOccurred())
				})
			})

			When("sample does not contain instance_id", func() {
				It("returns an error", func() {
					fakeGoLogCacheReader.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
						Result: &logcache_v1.PromQL_InstantQueryResult_Vector{
							Vector: &logcache_v1.PromQL_Vector{
								Samples: []*logcache_v1.PromQL_Sample{
									{
										Metric: map[string]string{
											// "instance_id": "0", is missing here
										},
									},
								},
							},
						},
					}, nil)
					_, err := logCacheClient.GetMetrics("app-id", "throughput", startTime, endTime)
					Expect(err).To(HaveOccurred())
				})
			})

			When("instance_id can not be parsed to uint", func() {
				It("returns an error", func() {
					fakeGoLogCacheReader.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
						Result: &logcache_v1.PromQL_InstantQueryResult_Vector{
							Vector: &logcache_v1.PromQL_Vector{
								Samples: []*logcache_v1.PromQL_Sample{
									{
										Metric: map[string]string{
											"instance_id": "iam-no-uint",
										},
									},
								},
							},
						},
					}, nil)
					_, err := logCacheClient.GetMetrics("app-id", "throughput", startTime, endTime)
					Expect(err).To(HaveOccurred())
				})
			})

			When("sample does not contain a point", func() {
				It("returns an error", func() {
					fakeGoLogCacheReader.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
						Result: &logcache_v1.PromQL_InstantQueryResult_Vector{
							Vector: &logcache_v1.PromQL_Vector{
								Samples: []*logcache_v1.PromQL_Sample{
									{
										Metric: map[string]string{
											"instance_id": "0",
										},
									},
								},
							},
						},
					}, nil)
					_, err := logCacheClient.GetMetrics("app-id", "throughput", startTime, endTime)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("reading throughput metrics", func() {
			When("PromQL API returns a vector with no samples", func() {
				It("returns empty metrics", func() {
					fakeGoLogCacheReader.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
						Result: &logcache_v1.PromQL_InstantQueryResult_Vector{
							Vector: &logcache_v1.PromQL_Vector{},
						},
					}, nil)

					metrics, err := logCacheClient.GetMetrics("app-id", "throughput", startTime, endTime)

					Expect(err).To(Not(HaveOccurred()))
					Expect(metrics).To(HaveLen(1))

					Expect(metrics[0].AppId).To(Equal("app-id"))
					Expect(metrics[0].InstanceIndex).To(Equal(uint32(0)))
					Expect(metrics[0].Name).To(Equal("throughput"))
					Expect(metrics[0].Unit).To(Equal("rps"))
					Expect(metrics[0].Value).To(Equal("0"))
				})
			})

			When("promql api returns a vector with samples", func() {
				It("returns metrics", func() {
					fakeGoLogCacheReader.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
						Result: &logcache_v1.PromQL_InstantQueryResult_Vector{
							Vector: &logcache_v1.PromQL_Vector{
								Samples: []*logcache_v1.PromQL_Sample{
									{
										Metric: map[string]string{
											"instance_id": "0",
										},
										Point: &logcache_v1.PromQL_Point{
											Value: 123,
										},
									},
									{
										Metric: map[string]string{
											"instance_id": "1",
										},
										Point: &logcache_v1.PromQL_Point{
											Value: 321,
										},
									},
								},
							},
						},
					}, nil)

					metrics, err := logCacheClient.GetMetrics("app-id", "throughput", startTime, endTime)

					_, query, _ := fakeGoLogCacheReader.PromQLArgsForCall(0)
					Expect(query).To(Equal("sum by (instance_id) (count_over_time(http{source_id='app-id',peer_type='Client'}[40s])) / 40"))

					Expect(err).To(Not(HaveOccurred()))
					Expect(metrics).To(HaveLen(2))

					Expect(metrics[0].AppId).To(Equal("app-id"))
					Expect(metrics[0].InstanceIndex).To(Equal(uint32(0)))
					Expect(metrics[0].Name).To(Equal("throughput"))
					Expect(metrics[0].Unit).To(Equal("rps"))
					Expect(metrics[0].Value).To(Equal("123"))

					Expect(metrics[1].AppId).To(Equal("app-id"))
					Expect(metrics[1].InstanceIndex).To(Equal(uint32(1)))
					Expect(metrics[1].Name).To(Equal("throughput"))
					Expect(metrics[1].Unit).To(Equal("rps"))
					Expect(metrics[1].Value).To(Equal("321"))
				})
			})
		})

		When("reading responsetime metrics", func() {
			When("PromQL API returns a vector with no samples", func() {
				It("returns empty metrics", func() {
					fakeGoLogCacheReader.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
						Result: &logcache_v1.PromQL_InstantQueryResult_Vector{
							Vector: &logcache_v1.PromQL_Vector{},
						},
					}, nil)

					metrics, err := logCacheClient.GetMetrics("app-id", "responsetime", startTime, endTime)

					Expect(err).To(Not(HaveOccurred()))
					Expect(metrics).To(HaveLen(1))

					Expect(metrics[0].AppId).To(Equal("app-id"))
					Expect(metrics[0].InstanceIndex).To(Equal(uint32(0)))
					Expect(metrics[0].Name).To(Equal("responsetime"))
					Expect(metrics[0].Unit).To(Equal("ms"))
					Expect(metrics[0].Value).To(Equal("0"))
				})
			})

			When("promql api returns a vector with samples", func() {
				It("returns metrics", func() {
					fakeGoLogCacheReader.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
						Result: &logcache_v1.PromQL_InstantQueryResult_Vector{
							Vector: &logcache_v1.PromQL_Vector{
								Samples: []*logcache_v1.PromQL_Sample{
									{
										Metric: map[string]string{
											"instance_id": "0",
										},
										Point: &logcache_v1.PromQL_Point{
											Value: 200,
										},
									},
									{
										Metric: map[string]string{
											"instance_id": "1",
										},
										Point: &logcache_v1.PromQL_Point{
											Value: 300,
										},
									},
								},
							},
						},
					}, nil)

					metrics, err := logCacheClient.GetMetrics("app-id", "responsetime", startTime, endTime)

					_, query, _ := fakeGoLogCacheReader.PromQLArgsForCall(0)
					Expect(query).To(Equal("avg by (instance_id) (max_over_time(http{source_id='app-id',peer_type='Client'}[40s])) / (1000 * 1000)"))

					Expect(err).To(Not(HaveOccurred()))
					Expect(metrics).To(HaveLen(2))

					Expect(metrics[0].AppId).To(Equal("app-id"))
					Expect(metrics[0].InstanceIndex).To(Equal(uint32(0)))
					Expect(metrics[0].Name).To(Equal("responsetime"))
					Expect(metrics[0].Unit).To(Equal("ms"))
					Expect(metrics[0].Value).To(Equal("200"))

					Expect(metrics[1].AppId).To(Equal("app-id"))
					Expect(metrics[1].InstanceIndex).To(Equal(uint32(1)))
					Expect(metrics[1].Name).To(Equal("responsetime"))
					Expect(metrics[1].Unit).To(Equal("ms"))
					Expect(metrics[1].Value).To(Equal("300"))
				})
			})
		})

		DescribeTable("GetMetrics for Gauge Metrics",
			func(autoscalerMetricType string, requiredFilters []string) {
				metrics = []models.AppInstanceMetric{
					{
						AppId: "some-id",
						Name:  autoscalerMetricType,
					},
				}
				fakeEnvelopeProcessor.GetGaugeMetricsReturnsOnCall(0, metrics, nil)
				actualMetrics, err := logCacheClient.GetMetrics(appId, autoscalerMetricType, startTime, endTime)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualMetrics).To(Equal(metrics))

				By("Sends the right arguments to log-cache-client")
				actualContext, actualAppId, actualStartTime, readOptions := fakeGoLogCacheReader.ReadArgsForCall(0)
				Expect(actualContext).To(Equal(context.Background()))
				Expect(actualAppId).To(Equal(appId))
				Expect(actualStartTime).To(Equal(startTime))

				Expect(valuesFrom(readOptions[0])["end_time"][0]).To(Equal(strconv.FormatInt(int64(endTime.UnixNano()), 10)))

				Expect(len(readOptions)).To(Equal(3), "filters by envelope type and metric names based on the requested metric type sent to GetMetrics")
				Expect(valuesFrom(readOptions[1])["envelope_types"][0]).To(Equal("GAUGE"))

				// after starTime and envelopeType we filter the metric names
				Expect(valuesFrom(readOptions[2])["name_filter"][0]).To(Equal(requiredFilters[2]))

				By("Sends the right arguments to the gauge processor")
				actualEnvelopes, actualCurrentTimestamp := fakeEnvelopeProcessor.GetGaugeMetricsArgsForCall(0)
				Expect(actualEnvelopes).To(Equal(envelopes))
				Expect(actualCurrentTimestamp).To(Equal(collectedAt.UnixNano()))
			},
			Entry("When metric type is memoryutil", models.MetricNameMemoryUtil, []string{"endtime", "envelope_type", "memory|memory_quota"}),
			Entry("When metric type is memoryused", models.MetricNameMemoryUsed, []string{"endtime", "envelope_type", "memory"}),
			Entry("When metric type is cpu", models.MetricNameCPU, []string{"endtime", "envelope_type", "cpu"}),
			Entry("When metric type is cpuutil", models.MetricNameCPUUtil, []string{"endtime", "envelope_type", "cpu_entitlement"}),
			Entry("When metric type is disk", models.MetricNameDisk, []string{"endtime", "envelope_type", "disk"}),
			Entry("When metric type is diskutil", models.MetricNameDiskUtil, []string{"endtime", "envelope_type", "disk|disk_quota"}),
			Entry("When metric type is CustomMetrics", "a-custom-metric", []string{"endtime", "envelope_type", "a-custom-metric"}),
		)

		Describe("when quering 1 metrics", func() {

			BeforeEach(func() {
				metrics = nil
				metrics = append(metrics, models.AppInstanceMetric{
					AppId:         appId,
					InstanceIndex: 0,
					Name:          models.MetricNameMemoryUsed,
					Unit:          models.UnitMegaBytes,
					Value:         "21",
					Timestamp:     1111,
				})
				metrics = append(metrics, models.AppInstanceMetric{
					AppId:         appId,
					InstanceIndex: 0,
					Name:          models.MetricNameMemoryUtil,
					Unit:          models.UnitPercentage,
					Value:         "51",
					Timestamp:     1111,
				})
			})

			It("should retrieve requested metrics only", func() {
				actualMetrics, err := logCacheClient.GetMetrics(appId, models.MetricNameMemoryUsed, startTime, endTime)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(actualMetrics)).To(Equal(1))
				Expect(actualMetrics[0]).To(Equal(metrics[0]))
			})
		})
	})

})

func valuesFrom(option logcache.ReadOption) url.Values {
	values := url.Values{}
	option(nil, values)
	return values
}
