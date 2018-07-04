package server_test

import (
	"autoscaler/db"
	"autoscaler/eventgenerator/aggregator/fakes"
	. "autoscaler/eventgenerator/server"
	"autoscaler/models"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var testUrlAggregatedMetricHistories = "http://localhost/v1/apps/an-app-id/aggregated_metric_histories/a-metric-type"

var _ = Describe("EventgenHandler", func() {
	var (
		handler  *EventGenHandler
		database *fakes.FakeAppMetricDB

		resp    *httptest.ResponseRecorder
		req     *http.Request
		err     error
		metric1 models.AppMetric
		metric2 models.AppMetric
	)

	BeforeEach(func() {
		logger := lager.NewLogger("handler-test")
		database = &fakes.FakeAppMetricDB{}
		resp = httptest.NewRecorder()
		handler = NewEventGenHandler(logger, database)
	})

	Describe("GetAggregatedMetricHistories", func() {
		JustBeforeEach(func() {
			handler.GetAggregatedMetricHistories(resp, req, map[string]string{"appid": "an-app-id", "metrictype": "a-metric-type"})
		})

		Context("when request query string is invalid", func() {
			Context("when there are multiple start pararmeters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&start=231", nil)
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
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=abc", nil)
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
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?end=123&end=231", nil)
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
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?end=abc", nil)
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

			Context("when there are multiple order parameters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?order=asc&order=asc", nil)
					Expect(err).ToNot(HaveOccurred())

				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect order parameter in query string",
					}))
				})
			})

			Context("when order value is invalid", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?order=not-order-type", nil)
					Expect(err).ToNot(HaveOccurred())

				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: fmt.Sprintf("Incorrect order parameter in query string, the value can only be %s or %s", db.ASCSTR, db.DESCSTR),
					}))
				})
			})

		})

		Context("when request query string is valid", func() {
			Context("when start,end and order are all in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with the given start, end and order ", func() {
					appid, name, start, end, order := database.RetrieveAppMetricsArgsForCall(0)
					Expect(appid).To(Equal("an-app-id"))
					Expect(name).To(Equal("a-metric-type"))
					Expect(start).To(Equal(int64(123)))
					Expect(end).To(Equal(int64(567)))
					Expect(order).To(Equal(db.DESC))
				})

			})

			Context("when there is no start time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?end=123&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with start time  0", func() {
					_, _, start, _, _ := database.RetrieveAppMetricsArgsForCall(0)
					Expect(start).To(Equal(int64(0)))
				})

			})

			Context("when there is no end time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with end time -1 ", func() {
					_, _, _, end, _ := database.RetrieveAppMetricsArgsForCall(0)
					Expect(end).To(Equal(int64(-1)))
				})

			})

			Context("when there is no order in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics from database with end time -1 ", func() {
					_, _, _, _, order := database.RetrieveAppMetricsArgsForCall(0)
					Expect(order).To(Equal(db.ASC))
				})

			})

			Context("when query database succeeds", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())

					metric1 = models.AppMetric{
						AppId:      "an-app-id",
						MetricType: "a-metric-type",
						Unit:       "metric-unit",
						Value:      "12345678",
						Timestamp:  111100,
					}

					metric2 = models.AppMetric{
						AppId:      "an-app-id",
						MetricType: "a-metric-type",
						Unit:       "metric-unit",
						Value:      "87654321",
						Timestamp:  111111,
					}
					database.RetrieveAppMetricsReturns([]*models.AppMetric{&metric2, &metric1}, nil)
				})

				It("returns 200 with metrics in message body", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))

					mtrcs := &[]models.AppMetric{}
					err = json.Unmarshal(resp.Body.Bytes(), mtrcs)

					Expect(err).ToNot(HaveOccurred())
					Expect(*mtrcs).To(Equal([]models.AppMetric{metric2, metric1}))
				})
			})

			Context("when query database fails", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())

					database.RetrieveAppMetricsReturns(nil, errors.New("database error"))
				})

				It("returns 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Interal-Server-Error",
						Message: "Error getting aggregated metric histories from database",
					}))
				})
			})

		})
	})
})
