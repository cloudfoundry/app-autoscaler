package client_test

import (
	logcache "code.cloudfoundry.org/go-log-cache"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/client"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogCacheClient", func() {
	var (
		fakeEnvelopeProcessor   *fakes.FakeEnvelopeProcessor
		fakeLogCacheClient      *fakes.FakeLogCacheClientReader
		appId                   string
		logger                  *lagertest.TestLogger
		logCacheClient          *LogCacheClient
		envelopes               []*loggregator_v2.Envelope
		metrics                 []models.AppInstanceMetric
		startTime               time.Time
		endTime                 time.Time
		collectedAt             time.Time
		logCacheClientReadError error
	)

	BeforeEach(func() {
		fakeEnvelopeProcessor = &fakes.FakeEnvelopeProcessor{}
		fakeLogCacheClient = &fakes.FakeLogCacheClientReader{}
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

	})

	JustBeforeEach(func() {
		fakeLogCacheClient.ReadReturns(envelopes, logCacheClientReadError)
		fakeEnvelopeProcessor.GetHttpStartStopInstanceMetricsReturns(metrics)
		fakeEnvelopeProcessor.GetGaugeInstanceMetricsReturnsOnCall(0, metrics, nil)
		fakeEnvelopeProcessor.GetGaugeInstanceMetricsReturnsOnCall(1, nil, errors.New("some error"))

		logCacheClient = NewLogCacheClient(logger, func() time.Time { return collectedAt }, fakeLogCacheClient, fakeEnvelopeProcessor)

	})

	Context("GetMetrics", func() {
		Describe("when log cache returns error on read", func() {
			BeforeEach(func() {
				logCacheClientReadError = errors.New("some Read error")
			})

			It("return error", func() {
				_, err := logCacheClient.GetMetric(appId, models.MetricNameMemoryUtil, startTime, endTime)
				Expect(err).To(HaveOccurred())
			})
		})

		DescribeTable("GetMetric for startStop Metrics",
			func(metricType string) {
				metrics = []models.AppInstanceMetric{
					{
						AppId: "some-id",
						Name:  metricType,
					},
				}
				fakeEnvelopeProcessor.GetHttpStartStopInstanceMetricsReturnsOnCall(0, metrics)
				actualMetrics, err := logCacheClient.GetMetric(appId, metricType, startTime, endTime)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualMetrics).To(Equal(metrics))

				Expect(err).NotTo(HaveOccurred())
				Expect(fakeLogCacheClient.ReadCallCount()).To(Equal(1))

				By("Sends the right arguments to log-cache-client")
				actualContext, actualAppId, actualStartTime, readOptions := fakeLogCacheClient.ReadArgsForCall(0)
				Expect(actualContext).To(Equal(context.Background()))
				Expect(actualAppId).To(Equal(appId))
				Expect(actualStartTime).To(Equal(startTime))
				values := url.Values{}
				readOptions[0](nil, values)
				Expect(values["end_time"][0]).To(Equal(fmt.Sprintf("%d", endTime.UnixNano())))
				values = url.Values{}
				readOptions[1](nil, values)
				Expect(values["envelope_types"][0]).To(Equal("TIMER"))

				By("Sends the right arguments to the timer processor")
				Expect(fakeEnvelopeProcessor.GetHttpStartStopInstanceMetricsCallCount()).To(Equal(1), "Should call GetHttpStartStopInstanceMetricsCallCount once")
				actualEnvelopes, actualAppId, actualCurrentTimestamp, collectionInterval := fakeEnvelopeProcessor.GetHttpStartStopInstanceMetricsArgsForCall(0)
				Expect(actualEnvelopes).To(Equal(envelopes))
				Expect(actualAppId).To(Equal(appId))
				Expect(actualCurrentTimestamp).To(Equal(collectedAt.UnixNano()))
				Expect(collectionInterval).To(Equal(30 * time.Second)) // default behaviour
			},
			Entry("When metric type is MetricNameThroughput", models.MetricNameThroughput),
			Entry("When metric type is MetricNameResponseTime", models.MetricNameResponseTime),
		)

		DescribeTable("GetMetric for Gauge Metrics",
			func(autoscalerMetricType string, requiredFilters []string) {
				metrics = []models.AppInstanceMetric{
					{
						AppId: "some-id",
						Name:  autoscalerMetricType,
					},
				}
				fakeEnvelopeProcessor.GetGaugeInstanceMetricsReturnsOnCall(0, metrics, nil)
				actualMetrics, err := logCacheClient.GetMetric(appId, autoscalerMetricType, startTime, endTime)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualMetrics).To(Equal(metrics))

				By("Sends the right arguments to log-cache-client")
				_, _, _, readOptions := fakeLogCacheClient.ReadArgsForCall(0)

				Expect(valuesFrom(readOptions[0])["end_time"][0]).To(Equal(strconv.FormatInt(int64(endTime.UnixNano()), 10)))

				Expect(len(readOptions)).To(Equal(len(requiredFilters)), "filters by envelope type and metric names based on the requested metric type sent to GetMetric")
				Expect(valuesFrom(readOptions[1])["envelope_types"][0]).To(Equal("GAUGE"))

				// after starTime and envelopeType we filter the metric names
				for i := 2; i < len(requiredFilters); i++ {
					Expect(valuesFrom(readOptions[i])["name_filter"][0]).To(Equal(requiredFilters[i]))
				}

				Expect(fakeEnvelopeProcessor.GetHttpStartStopInstanceMetricsCallCount()).To(Equal(0))

				By("Sends the right arguments to the gauge processor")
				actualEnvelopes, actualCurrentTimestamp := fakeEnvelopeProcessor.GetGaugeInstanceMetricsArgsForCall(0)
				Expect(actualEnvelopes).To(Equal(envelopes))
				Expect(actualCurrentTimestamp).To(Equal(collectedAt.UnixNano()))
			},
			Entry("When metric type is MetricNameMemoryUtil", models.MetricNameMemoryUtil, []string{"endtime", "envelope_type", "memory", "memory_quota"}),
			Entry("When metric type is MetricNameMemoryUsed", models.MetricNameMemoryUsed, []string{"endtime", "envelope_type", "memory"}),
			Entry("When metric type is MetricNameCPUUtil", models.MetricNameCPUUtil, []string{"endtime", "envelope_type", "cpu"}),
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

			It("should retrive requested metrics only", func() {
				actualMetrics, err := logCacheClient.GetMetric(appId, models.MetricNameMemoryUsed, startTime, endTime)
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
