package metric_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/metric"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	logcache "code.cloudfoundry.org/go-log-cache/v3"
	"code.cloudfoundry.org/go-log-cache/v3/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("logCacheFetcher", func() {

	var (
		testLogger *lagertest.TestLogger

		mockLogCacheClient  *fakes.FakeLogCacheClient
		mockEnvelopeProcess *fakes.FakeEnvelopeProcessor

		collectionInterval time.Duration

		metricFetcher metric.Fetcher
	)

	BeforeEach(func() {
		testLogger = lagertest.NewTestLogger("testLogger")

		mockLogCacheClient = &fakes.FakeLogCacheClient{}
		mockEnvelopeProcess = &fakes.FakeEnvelopeProcessor{}

		collectionInterval = 40 * time.Second

		metricFetcher = metric.StandardLogCacheFetcherCreator.NewLogCacheFetcher(testLogger, mockLogCacheClient, mockEnvelopeProcess, collectionInterval)
	})

	Describe("FetchMetrics", func() {

		When("reading metric from PromQL API", func() {

			When("PromQL call fails", func() {
				It("returns an error", func() {
					mockLogCacheClient.PromQLReturns(nil, errors.New("fail"))
					_, err := metricFetcher.FetchMetrics("app-id", "throughput", time.Now(), time.Now())
					Expect(err).To(HaveOccurred())
				})
			})

			When("PromQL result is not a vector", func() {
				It("returns an error", func() {
					mockLogCacheClient.PromQLReturns(nil, nil)
					_, err := metricFetcher.FetchMetrics("app-id", "throughput", time.Now(), time.Now())
					Expect(err).To(HaveOccurred())
				})
			})

			When("vector does not contain samples", func() {
				It("returns empty metric", func() {
					mockLogCacheClient.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
						Result: &logcache_v1.PromQL_InstantQueryResult_Vector{
							Vector: &logcache_v1.PromQL_Vector{
								Samples: nil,
							},
						},
					}, nil)

					metrics, err := metricFetcher.FetchMetrics("app-id", "throughput", time.Now(), time.Now())

					Expect(err).ToNot(HaveOccurred())
					Expect(metrics).To(HaveLen(1))
					Expect(metrics[0].AppId).To(Equal("app-id"))
					Expect(metrics[0].InstanceIndex).To(Equal(uint64(0)))
					Expect(metrics[0].Name).To(Equal("throughput"))
					Expect(metrics[0].Unit).To(Equal("rps"))
					Expect(metrics[0].Value).To(Equal("0"))
					Expect(time.Unix(0, metrics[0].CollectedAt)).To(BeTemporally("~", time.Now(), time.Second))
					Expect(time.Unix(0, metrics[0].Timestamp)).To(BeTemporally("~", time.Now(), time.Second))
				})
			})

			When("sample does not contain instance_id", func() {
				It("returns an error", func() {
					mockLogCacheClient.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
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
					_, err := metricFetcher.FetchMetrics("app-id", "throughput", time.Now(), time.Now())
					Expect(err).To(HaveOccurred())
				})
			})

			When("instance_id can not be parsed to uint", func() {
				It("returns an error", func() {
					mockLogCacheClient.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
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
					_, err := metricFetcher.FetchMetrics("app-id", "throughput", time.Now(), time.Now())
					Expect(err).To(HaveOccurred())
				})
			})

			When("sample does not contain a point", func() {
				It("returns an error", func() {
					mockLogCacheClient.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
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
					_, err := metricFetcher.FetchMetrics("app-id", "throughput", time.Now(), time.Now())
					Expect(err).To(HaveOccurred())
				})
			})

			When("reading metric for metric_type responsetime", func() {
				It("should succeed", func() {
					mockLogCacheClient.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
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

					metrics, err := metricFetcher.FetchMetrics("app-id", "responsetime", time.Now(), time.Now())

					Expect(err).ToNot(HaveOccurred())
					Expect(metrics).To(HaveLen(2))
					Expect(metrics[0].AppId).To(Equal("app-id"))
					Expect(metrics[0].InstanceIndex).To(Equal(uint64(0)))
					Expect(metrics[0].Name).To(Equal("responsetime"))
					Expect(metrics[0].Unit).To(Equal("ms"))
					Expect(metrics[0].Value).To(Equal("200"))

					Expect(metrics[1].AppId).To(Equal("app-id"))
					Expect(metrics[1].InstanceIndex).To(Equal(uint64(1)))
					Expect(metrics[1].Name).To(Equal("responsetime"))
					Expect(metrics[1].Unit).To(Equal("ms"))
					Expect(metrics[1].Value).To(Equal("300"))

					_, query, _ := mockLogCacheClient.PromQLArgsForCall(0)
					Expect(query).To(Equal("avg by (instance_id) (max_over_time(http{source_id='app-id',peer_type='Client'}[40s])) / (1000 * 1000)"))
				})
			})

			When("reading metric for metric_type throughput", func() {
				It("should succeed", func() {
					mockLogCacheClient.PromQLReturns(&logcache_v1.PromQL_InstantQueryResult{
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

					metrics, err := metricFetcher.FetchMetrics("app-id", "throughput", time.Now(), time.Now())

					Expect(err).To(Not(HaveOccurred()))
					Expect(metrics).To(HaveLen(2))

					Expect(metrics[0].AppId).To(Equal("app-id"))
					Expect(metrics[0].InstanceIndex).To(Equal(uint64(0)))
					Expect(metrics[0].Name).To(Equal("throughput"))
					Expect(metrics[0].Unit).To(Equal("rps"))
					Expect(metrics[0].Value).To(Equal("123"))

					Expect(metrics[1].AppId).To(Equal("app-id"))
					Expect(metrics[1].InstanceIndex).To(Equal(uint64(1)))
					Expect(metrics[1].Name).To(Equal("throughput"))
					Expect(metrics[1].Unit).To(Equal("rps"))
					Expect(metrics[1].Value).To(Equal("321"))

					_, query, _ := mockLogCacheClient.PromQLArgsForCall(0)
					Expect(query).To(Equal("sum by (instance_id) (count_over_time(http{source_id='app-id',peer_type='Client'}[40s])) / 40"))
				})
			})
		})

		When("reading metric from REST API", func() {
			When("log cache returns error", func() {
				BeforeEach(func() {
					mockLogCacheClient.ReadReturns(nil, errors.New("error"))
				})

				It("return error", func() {
					_, err := metricFetcher.FetchMetrics("app-id", models.MetricNameMemoryUtil, time.Now(), time.Now())
					Expect(err).To(HaveOccurred())
				})
			})

			DescribeTable("via specific metric_type that causes a read from the REST API",
				func(metricType string, nameFilter string) {
					expectedMetrics := []models.AppInstanceMetric{
						{
							AppId: "app-id",
							Name:  metricType,
						},
					}
					mockEnvelopeProcess.GetGaugeMetricsReturns(expectedMetrics)
					mockLogCacheClient.ReadReturns([]*loggregator_v2.Envelope{
						{
							SourceId: "app-id",
						},
					}, nil)
					expectedStartTime := time.Now()
					expectedEndTime := time.Now()

					actualMetrics, err := metricFetcher.FetchMetrics("app-id", metricType, expectedStartTime, expectedEndTime)

					Expect(err).NotTo(HaveOccurred())
					Expect(actualMetrics).To(Equal(expectedMetrics))

					actualContext, actualAppId, actualStartTime, readOptions := mockLogCacheClient.ReadArgsForCall(0)
					Expect(actualContext).To(Equal(context.Background()))
					Expect(actualAppId).To(Equal("app-id"))
					Expect(actualStartTime).To(Equal(expectedStartTime))

					Expect(len(readOptions)).To(Equal(3))
					Expect(valuesFrom(readOptions[0])["end_time"][0]).To(Equal(fmt.Sprintf("%d", expectedEndTime.UnixNano())))
					Expect(valuesFrom(readOptions[1])["envelope_types"][0]).To(Equal("GAUGE"))
					Expect(valuesFrom(readOptions[2])["name_filter"][0]).To(Equal(nameFilter))

					actualEnvelopes, actualCurrentTimestamp := mockEnvelopeProcess.GetGaugeMetricsArgsForCall(0)
					Expect(actualEnvelopes).To(Equal([]*loggregator_v2.Envelope{
						{SourceId: "app-id"},
					}))
					Expect(time.Unix(0, actualCurrentTimestamp)).Should(BeTemporally("~", time.Now(), time.Second))
				},
				Entry("metric type memoryutil", models.MetricNameMemoryUtil, "memory|memory_quota"),
				Entry("metric type memoryused", models.MetricNameMemoryUsed, "memory"),
				Entry("metric type cpu", models.MetricNameCPU, "cpu"),
				Entry("metric type cpuutil", models.MetricNameCPUUtil, "cpu_entitlement"),
				Entry("metric type disk", models.MetricNameDisk, "disk"),
				Entry("metric type diskutil", models.MetricNameDiskUtil, "disk|disk_quota"),
				Entry("metric type CustomMetrics", "a-custom-metric", "a-custom-metric"),
			)
		})
	})
})

func valuesFrom(option logcache.ReadOption) url.Values {
	values := url.Values{}
	option(nil, values)
	return values
}
