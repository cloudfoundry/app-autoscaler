package server_test

import (
	"errors"
	"metricscollector/metrics"
	. "metricscollector/server"
	"metricscollector/server/fakes"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("MemoryMetricHandler", func() {

	var (
		cfc      *fakes.FakeCfClient
		consumer *fakes.FakeNoaaConsumer
		handler  *MemoryMetricHandler
		resp     *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		consumer = &fakes.FakeNoaaConsumer{}
		logger := lager.NewLogger("handler-test")
		resp = httptest.NewRecorder()

		handler = NewMemoryMetricHandler(logger, cfc, consumer)
	})

	Describe("GetMemoryMetric", func() {
		JustBeforeEach(func() {
			handler.GetMemoryMetric(resp, nil, map[string]string{"appid": "an-app-id"})
		})

		Context("when retrieving container metrics fail", func() {
			BeforeEach(func() {
				consumer.ContainerMetricsReturns(nil, errors.New("an error"))
			})

			It("returns a 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				errJson := &ErrorResponse{}
				d := json.NewDecoder(resp.Body)
				err := d.Decode(errJson)

				Expect(err).ToNot(HaveOccurred())
				Expect(errJson.Code).To(Equal("Interal-Server-Error"))
				Expect(errJson.Message).To(Equal("Error getting memory metrics from doppler"))
			})
		})

		Context("when retrieving container metrics succeed", func() {
			Context("container metrics is empty", func() {
				BeforeEach(func() {
					consumer.ContainerMetricsReturns([]*events.ContainerMetric{}, nil)
				})

				It("returns a 200 with empty metrics", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))

					metric := &metrics.Metric{}
					d := json.NewDecoder(resp.Body)
					err := d.Decode(metric)

					Expect(err).ToNot(HaveOccurred())
					Expect(metric.AppId).To(Equal("an-app-id"))
					Expect(metric.Name).To(Equal(metrics.MemoryMetricName))
					Expect(metric.Unit).To(Equal(metrics.UnitBytes))
					Expect(metric.Instances).To(BeEmpty())
				})
			})

			Context("container metrics is not empty", func() {
				BeforeEach(func() {
					consumer.ContainerMetricsReturns([]*events.ContainerMetric{
						&events.ContainerMetric{
							ApplicationId: proto.String("an-app-id"),
							InstanceIndex: proto.Int32(1),
							MemoryBytes:   proto.Uint64(1234),
						},
					}, nil)
				})

				It("returns a 200 response with metrics", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))

					metric := &metrics.Metric{}
					d := json.NewDecoder(resp.Body)
					err := d.Decode(metric)

					Expect(err).ToNot(HaveOccurred())
					Expect(metric.AppId).To(Equal("an-app-id"))
					Expect(metric.Name).To(Equal(metrics.MemoryMetricName))
					Expect(metric.Unit).To(Equal(metrics.UnitBytes))
					Expect(metric.Instances).To(ConsistOf(metrics.InstanceMetric{Index: 1, Value: "1234"}))
				})
			})
		})
	})
})
