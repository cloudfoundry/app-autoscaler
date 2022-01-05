package collector_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsserver/collector"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
)

var testUrlMetricHistories = "http://metrics_collector_hostname/v1/apps/an-app-id/metric_histories/a-metric-type"

var _ = Describe("MetricHandler", func() {

	var (
		handler            *MetricHandler
		nodeIndex          int
		nodeAddrs          []string
		resp               *httptest.ResponseRecorder
		req                *http.Request
		err, queryErr      error
		metric1            models.AppInstanceMetric
		metric2            models.AppInstanceMetric
		metrics            []*models.AppInstanceMetric
		appID              string
		instanceIndex      int
		metricName         string
		startTime, endTime int64
		queryOrder         db.OrderType
	)

	BeforeEach(func() {
		nodeIndex = 0
		nodeAddrs = []string{"localhost:8080"}
		resp = httptest.NewRecorder()
		queryErr = nil
	})

	Describe("GetMetricHistory", func() {
		JustBeforeEach(func() {
			logger := lager.NewLogger("handler-test")
			queryFunc := func(id string, index int, name string,
				start, end int64, order db.OrderType) ([]*models.AppInstanceMetric, error) {
				appID = id
				instanceIndex = index
				metricName = name
				startTime = start
				endTime = end
				queryOrder = order
				return metrics, queryErr
			}
			handler = NewMetricHandler(logger, nodeIndex, nodeAddrs, queryFunc)
			handler.GetMetricHistories(resp, req, map[string]string{"appid": "an-app-id", "metrictype": "a-metric-type"})
		})

		Context("when request query string is invalid", func() {

			Context("when there are multiple instanceindex pararmeters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=123&instanceindex=231", nil)
					Expect(err).ToNot(HaveOccurred())

				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Incorrect instanceIndex parameter in query string",
					}))
				})
			})

			Context("when instanceindex is not a number", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=abc", nil)
					Expect(err).ToNot(HaveOccurred())

				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "Error parsing instanceIndex",
					}))
				})
			})

			Context("when instanceindex is smaller than 0", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=-1", nil)
					Expect(err).ToNot(HaveOccurred())

				})

				It("returns 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Bad-Request",
						Message: "InstanceIndex must be greater than or equal to 0",
					}))
				})
			})

			Context("when there are multiple start pararmeters in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?start=123&start=231", nil)
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
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?start=abc", nil)
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
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?end=123&end=231", nil)
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
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?end=abc", nil)
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
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?order=asc&order=asc", nil)
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
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?order=not-order-type", nil)
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
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=0&start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics  with the given start, end and order ", func() {
					Expect(instanceIndex).To(Equal(0))
					Expect(appID).To(Equal("an-app-id"))
					Expect(metricName).To(Equal("a-metric-type"))
					Expect(startTime).To(Equal(int64(123)))
					Expect(endTime).To(Equal(int64(567)))
					Expect(queryOrder).To(Equal(db.DESC))
				})

			})

			Context("when there is no instanceindex in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics with instanceindex  -1", func() {
					Expect(instanceIndex).To(Equal(-1))
				})

			})

			Context("when there is no start time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=0&end=123&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics  with start time  0", func() {
					Expect(startTime).To(Equal(int64(0)))
				})

			})

			Context("when there is no end time in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=0&start=123&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics with end time -1 ", func() {
					Expect(endTime).To(Equal(int64(-1)))
				})

			})

			Context("when there is no order in query string", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=0&start=123&end=567", nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("queries metrics  with end time -1 ", func() {
					Expect(queryOrder).To(Equal(db.ASC))
				})

			})

			Context("when query succeeds", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=0&start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())

					metric1 = models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   111,
						Name:          "a-metric-type",
						Unit:          "metric-unit",
						Value:         "12345678",
						Timestamp:     345,
					}

					metric2 = models.AppInstanceMetric{
						AppId:         "an-app-id",
						InstanceIndex: 0,
						CollectedAt:   222,
						Name:          "a-metric-type",
						Unit:          "metric-unit",
						Value:         "87654321",
						Timestamp:     456,
					}
					metrics = []*models.AppInstanceMetric{&metric1, &metric2}
				})

				It("returns 200 with metrics in message body", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
					result := &[]models.AppInstanceMetric{}
					err = json.Unmarshal(resp.Body.Bytes(), result)

					Expect(err).ToNot(HaveOccurred())
					Expect(*result).To(Equal([]models.AppInstanceMetric{metric1, metric2}))
				})

			})

			Context("when requesting app metrics in other shard", func() {
				BeforeEach(func() {
					nodeIndex = 1
					nodeAddrs = []string{"localhost:8080", "localhost:9090"}
				})
				Context("when it is not a redirect", func() {
					BeforeEach(func() {
						req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=0&start=123&end=567&order=desc", nil)
						Expect(err).ToNot(HaveOccurred())
					})

					It("Redirects to the right shard", func() {
						Expect(resp.Code).To(Equal(http.StatusFound))
						Expect(resp.Header()["Location"]).To(HaveLen(1))
						redirecURL, err := url.Parse(resp.Header()["Location"][0])
						Expect(err).NotTo(HaveOccurred())
						Expect(redirecURL.Scheme).To(Equal("https"))
						Expect(redirecURL.Host).To(Equal("localhost:8080"))
						Expect(redirecURL.Path).To(Equal("/v1/apps/an-app-id/metric_histories/a-metric-type"))
						Expect(redirecURL.Query()).To(BeEquivalentTo(map[string][]string{
							"instanceindex": {"0"},
							"start":         {"123"},
							"end":           {"567"},
							"order":         {"desc"},
							"referer":       {"localhost:9090"},
						}))
					})
				})

				Context("when it is a redirect from other node", func() {
					BeforeEach(func() {
						req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=0&start=123&end=567&order=desc&referer=another-node", nil)
						Expect(err).ToNot(HaveOccurred())
					})

					It("does not redirect again", func() {
						Expect(resp.Code).To(Equal(http.StatusOK))
					})
				})
			})

			Context("when query fails", func() {
				BeforeEach(func() {
					req, err = http.NewRequest(http.MethodGet, testUrlMetricHistories+"?instanceindex=0&start=123&end=567&order=desc", nil)
					Expect(err).ToNot(HaveOccurred())
					queryErr = errors.New("query error")
				})

				It("returns 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))

					errJson := &models.ErrorResponse{}
					err = json.Unmarshal(resp.Body.Bytes(), errJson)

					Expect(err).ToNot(HaveOccurred())
					Expect(errJson).To(Equal(&models.ErrorResponse{
						Code:    "Internal-Server-Error",
						Message: "Error getting instance metric histories",
					}))
				})
			})

		})
	})
})
