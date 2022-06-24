package aggregator_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/lagertest"
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/url"
	"time"
)

var _ = FDescribe("LogCacheClient", func() {
	var (
		fakeEnvelopeProcessor *fakes.FakeEnvelopeProcessor
		fakeLogCacheClient    *fakes.FakeLogCacheClientReader
		metricType            string
		appId                 string
		logger                *lagertest.TestLogger
		logCacheClient        *LogCacheClient
		envelopes             []*loggregator_v2.Envelope
		metrics               []models.AppInstanceMetric
		startTime             time.Time
		endTime               time.Time
		collectedAt           time.Time
	)

	BeforeEach(func() {
		fakeEnvelopeProcessor = &fakes.FakeEnvelopeProcessor{}
		fakeLogCacheClient = &fakes.FakeLogCacheClientReader{}
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
		fakeLogCacheClient.ReadReturns(envelopes, nil)
		fakeEnvelopeProcessor.GetHttpStartStopInstanceMetricsReturns(metrics)
		logCacheClient = NewLogCacheClient(logger, func() time.Time { return collectedAt }, fakeLogCacheClient, fakeEnvelopeProcessor)
	})

	Describe("GetMetric for startStop Metrics", func() {
		BeforeEach(func() { metricType = models.MetricNameThroughput })

		It("retrieve metrics", func() {
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
			actualEnvelopes, actualAppId, actualCurrentTimestamp, collectionInterval := fakeEnvelopeProcessor.GetHttpStartStopInstanceMetricsArgsForCall(0)
			Expect(actualEnvelopes).To(Equal(envelopes))
			Expect(actualAppId).To(Equal(appId))
			Expect(actualCurrentTimestamp).To(Equal(collectedAt.UnixNano()))
			Expect(collectionInterval).To(Equal(30 * time.Second)) // default behaviour
		})
		//TODO responseTime
	})

	Describe("GetMetric for Gauge Metrics", func() {
		BeforeEach(func() { metricType = models.MetricNameMemoryUsed })
		It("retrieve metrics", func() {
			actualMetrics, err := logCacheClient.GetMetric(appId, metricType, startTime, endTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(actualMetrics).To(Equal(metrics))

			By("Sends the right arguments to log-cache-client")
			_, _, _, readOptions := fakeLogCacheClient.ReadArgsForCall(0)
			values := url.Values{}
			readOptions[1](nil, values)
			Expect(values["envelope_types"][0]).To(Equal("GAUGE"))
			Expect(fakeEnvelopeProcessor.GetHttpStartStopInstanceMetricsCallCount()).To(Equal(0))

			By("Sends the right arguments to the gauge processor")
			actualEnvelopes, actualCurrentTimestamp := fakeEnvelopeProcessor.GetGaugeInstanceMetricsArgsForCall(0)
			Expect(actualEnvelopes).To(Equal(envelopes))
			Expect(actualCurrentTimestamp).To(Equal(collectedAt.UnixNano()))

		})
		//TODO add
		//	MetricNameMemoryUtil   = "memoryutil"
		//	MetricNameMemoryUsed   = "memoryused"
		//	MetricNameCPUUtil      = "cpu"
		//	MetricNameThroughput   = "throughput"
		//	MetricNameResponseTime = "responsetime"
	})
})
