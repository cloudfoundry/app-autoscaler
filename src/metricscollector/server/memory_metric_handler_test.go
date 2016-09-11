package server_test

import (
	"metricscollector/fakes"
	. "metricscollector/server"
	"models"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
)

var testUrlMemoryMetricHistories = "http://localhost/v1/apps/an-app-id/metric_histories/memory"

var _ = Describe("MemoryMetricHandler", func() {

	var (
		cfc      *fakes.FakeCfClient
		consumer *fakes.FakeNoaaConsumer
		handler  *MemoryMetricHandler
		database *fakes.FakeMetricsDB

		resp *httptest.ResponseRecorder
		req  *http.Request
		err  error

		metric1 models.Metric
		metric2 models.Metric
	)

	BeforeEach(func() {
		cfc = &fakes.FakeCfClient{}
		consumer = &fakes.FakeNoaaConsumer{}
		logger := lager.NewLogger("handler-test")
		database = &fakes.FakeMetricsDB{}
		resp = httptest.NewRecorder()
		handler = NewMemoryMetricHandler(logger, cfc, consumer, database)
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

				errJson := &models.ErrorResponse{}
				err = json.Unmarshal(resp.Body.Bytes(), errJson)

				Expect(err).ToNot(HaveOccurred())
				Expect(errJson).To(Equal(&models.ErrorResponse{
					Code:    "Interal-Server-Error",
					Message: "Error getting memory metrics from doppler",
				}))
			})
		})

		Context("when retrieving container metrics succeeds", func() {
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

					metric := &models.Metric{}
					err = json.Unmarshal(resp.Body.Bytes(), metric)

					Expect(err).ToNot(HaveOccurred())
					Expect(metric.AppId).To(Equal("an-app-id"))
					Expect(metric.Name).To(Equal(models.MetricNameMemory))
					Expect(metric.Unit).To(Equal(models.UnitBytes))
					Expect(metric.Instances).To(ConsistOf(models.InstanceMetric{Index: 1, Value: "1234"}))
				})
			})

			Context("container metrics is empty", func() {
				BeforeEach(func() {
					consumer.ContainerMetricsReturns([]*events.ContainerMetric{}, nil)
				})

				It("returns a 200 with empty metrics", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))

					metric := &models.Metric{}
					err = json.Unmarshal(resp.Body.Bytes(), metric)

					Expect(err).ToNot(HaveOccurred())
					Expect(metric.Instances).To(BeEmpty())
				})
			})

		})
	})

	Describe("GetMemoryMetricHistory", func() {
		JustBeforeEach(func() {
			handler.GetMemoryMetricHistories(resp, req, map[string]string{"appid": "an-app-id"})
		})

		Context("when request query string is invalid", func() {
			Context("when there are multiple start pararmeters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMemoryMetricHistories+"?start=123&start=231", nil)
					Expect(err).ToNot(HaveOccurred())

				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect start parameter in query string",
					}))
				})
			})

			Context("when start time is not a number", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMemoryMetricHistories+"?start=abc", nil)
					Expect(err).ToNot(HaveOccurred())

				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Error parsing start time",
					}))
				})
			})

			Context("when there are multiple end parameters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMemoryMetricHistories+"?end=123&end=231", nil)
					Expect(err).ToNot(HaveOccurred())

				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect end parameter in query string",
					}))
				})
			})

			Context("when end time is not a number", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMemoryMetricHistories+"?end=abc", nil)
					Expect(err).ToNot(HaveOccurred())

				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Error parsing end time",
					}))
				})
			})

		})

		Context("when request query string is valid", func() {
			Context("when there are both start and end time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMemoryMetricHistories+"?start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with the given start and end time ", func() {
					appid, name, start, end := database.RetrieveMetricsArgsForCall(0)
					Expect(appid).To(Equal("an-app-id"))
					Expect(name).To(Equal(models.MetricNameMemory))
					Expect(start).To(Equal(int64(123)))
					Expect(end).To(Equal(int64(567)))
				})

			})

			Context("when there is no start time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMemoryMetricHistories+"?end=123", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with start time  0", func() {
					_, _, start, _ := database.RetrieveMetricsArgsForCall(0)
					Expect(start).To(Equal(int64(0)))
				})

			})

			Context("when there is no end time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMemoryMetricHistories+"?start=123", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with end time -1 ", func() {
					_, _, _, end := database.RetrieveMetricsArgsForCall(0)
					Expect(end).To(Equal(int64(-1)))
				})

			})

			Context("when query database succeeds", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMemoryMetricHistories+"?start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())

					metric1 = models.Metric{
						Name:      models.MetricNameMemory,
						Unit:      models.UnitBytes,
						AppId:     "an-app-id",
						TimeStamp: 333,
						Instances: []models.InstanceMetric{{333, 0, "6666"}, {333, 1, "7777"}},
					}

					metric2 = models.Metric{
						Name:      models.MetricNameMemory,
						Unit:      models.UnitBytes,
						AppId:     "an-app-id",
						TimeStamp: 555,
						Instances: []models.InstanceMetric{{555, 0, "7777"}, {555, 1, "8888"}},
					}
					database.RetrieveMetricsReturns([]*models.Metric{&metric1, &metric2}, nil)
				})

				It("returns 200 with metrics in message body", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))

					mtrcs := &[]models.Metric{}
					err = json.Unmarshal(resp.Body.Bytes(), mtrcs)

					Expect(err).ToNot(HaveOccurred())
					Expect(*mtrcs).To(Equal([]models.Metric{metric1, metric2}))
				})
			})

			Context("when query database fails", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMemoryMetricHistories+"?start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())

					database.RetrieveMetricsReturns(nil, errors.New("database error"))
				})

				It("returns 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Interal-Server-Error",
						Message: "Error getting memory metric histories from database",
					}))
				})
			})

		})
	})
})
