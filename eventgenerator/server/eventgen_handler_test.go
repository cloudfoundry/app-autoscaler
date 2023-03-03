package server_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/aggregator"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testUrlAggregatedMetricHistories = "http://localhost/v1/apps/an-app-id/aggregated_metric_histories/a-metric-type"

var _ = Describe("EventgenHandler", func() {
	var (
		handler         *EventGenHandler
		queryAppMetrics aggregator.QueryAppMetricsFunc

		resp       *httptest.ResponseRecorder
		req        *http.Request
		err        error
		metric1    models.AppMetric
		metric2    models.AppMetric
		appid      string
		name       string
		start, end int64
		order      db.OrderType
		logger     lager.Logger
	)

	Describe("GetAggregatedMetricHistories", func() {
		JustBeforeEach(func() {
			logger = lager.NewLogger("handler-test")
			resp = httptest.NewRecorder()
			handler = NewEventGenHandler(logger, queryAppMetrics)
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
			BeforeEach(func() {
				queryAppMetrics = func(appID string, metricType string, startTime int64, endTime int64, orderType db.OrderType) ([]*models.AppMetric, error) {
					appid = appID
					name = metricType
					start = startTime
					end = endTime
					order = orderType
					return nil, nil
				}
			})
			Context("when start,end and order are all in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics with the given start, end and order ", func() {
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

				It("queries metrics with start time  0", func() {
					Expect(start).To(Equal(int64(0)))
				})

			})

			Context("when there is no end time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics with end time -1 ", func() {
					Expect(end).To(Equal(int64(-1)))
				})

			})

			Context("when there is no order in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics with ascending order ", func() {
					Expect(order).To(Equal(db.ASC))
				})

			})

			Context("when query metrics succeeds", func() {
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

					queryAppMetrics = func(appID string, metricType string, startTime int64, endTime int64, orderType db.OrderType) ([]*models.AppMetric, error) {
						return []*models.AppMetric{&metric2, &metric1}, nil
					}
				})

				It("returns 200 with metrics in message body", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))

					mtrcs := &[]models.AppMetric{}
					err = json.Unmarshal(resp.Body.Bytes(), mtrcs)

					Expect(err).ToNot(HaveOccurred())
					Expect(*mtrcs).To(Equal([]models.AppMetric{metric2, metric1}))
				})
			})

			Context("when query fails", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlAggregatedMetricHistories+"?start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())

					queryAppMetrics = func(appID string, metricType string, startTime int64, endTime int64, orderType db.OrderType) ([]*models.AppMetric, error) {
						return nil, errors.New("an error")
					}

				})

				It("returns 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Internal-Server-Error",
						Message: "Error getting aggregated metric histories",
					}))
				})
			})

		})
	})
})
