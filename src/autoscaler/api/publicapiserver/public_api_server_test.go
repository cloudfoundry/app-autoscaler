package publicapiserver_test

import (
	"autoscaler/models"

	// . "autoscaler/api/publicapiserver"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PublicApiServer", func() {
	var (
		rsp *http.Response
		err error
	)

	BeforeEach(func() {

		scalingEngineResponse = []models.AppScalingHistory{
			{
				AppId:        TEST_APP_ID,
				Timestamp:    300,
				ScalingType:  0,
				Status:       0,
				OldInstances: 2,
				NewInstances: 4,
				Reason:       "a reason",
				Message:      "",
				Error:        "",
			},
		}

		metricsCollectorResponse = []models.AppInstanceMetric{
			{
				AppId:         TEST_APP_ID,
				Timestamp:     100,
				InstanceIndex: 0,
				CollectedAt:   0,
				Name:          TEST_METRIC_TYPE,
				Unit:          TEST_METRIC_UNIT,
				Value:         "200",
			},
		}

		eventGeneratorResponse = []models.AppMetric{
			{
				AppId:      TEST_APP_ID,
				Timestamp:  100,
				MetricType: TEST_METRIC_TYPE,
				Unit:       TEST_METRIC_UNIT,
				Value:      "200",
			},
		}
	})

	Describe("Protected Routes", func() {
		Context("when calling scaling_histories endpoint without Authorization token", func() {
			BeforeEach(func() {
				serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/scaling_histories"

				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				rsp, err = httpClient.Do(req)
			})
			It("should fail", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})
		Context("when calling scaling_histories endpoint with Authorization token", func() {
			BeforeEach(func() {
				scalingEngineStatus = http.StatusOK

				serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/scaling_histories"

				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add("Authorization", TEST_USER_TOKEN)

				rsp, err = httpClient.Do(req)
			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		Context("when calling instance metrics endpoint without Authorization token", func() {
			BeforeEach(func() {
				serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/metric_histories/" + TEST_METRIC_TYPE

				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				rsp, err = httpClient.Do(req)
			})
			It("should fail", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})
		Context("when calling instance metric endpoint with Authorization token", func() {
			BeforeEach(func() {
				metricsCollectorStatus = http.StatusOK

				serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/metric_histories/" + TEST_METRIC_TYPE

				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add("Authorization", TEST_USER_TOKEN)

				rsp, err = httpClient.Do(req)
			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		Context("when calling aggregated metrics endpoint without Authorization token", func() {
			BeforeEach(func() {
				serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/" + TEST_METRIC_TYPE

				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				rsp, err = httpClient.Do(req)
			})
			It("should fail", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})
		Context("when calling aggregated metric endpoint with Authorization token", func() {
			BeforeEach(func() {
				eventGeneratorStatus = http.StatusOK

				serverUrl.Path = "/v1/apps/" + TEST_APP_ID + "/aggregated_metric_histories/" + TEST_METRIC_TYPE

				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				req.Header.Add("Authorization", TEST_USER_TOKEN)

				rsp, err = httpClient.Do(req)
			})
			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})
	Describe("UnProtected Routes", func() {
		Context("when calling info endpoint", func() {
			BeforeEach(func() {
				serverUrl.Path = "/v1/info"
				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				rsp, err = httpClient.Do(req)

			})

			It("should succeed", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		Context("when calling health endpoint", func() {
			BeforeEach(func() {
				serverUrl.Path = "/health"
				req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				rsp, err = httpClient.Do(req)

			})

			It("should succeed", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

	Context("when requesting non existing path", func() {
		BeforeEach(func() {
			serverUrl.Path = "/non-existing-path"

			req, err := http.NewRequest(http.MethodGet, serverUrl.String(), nil)
			Expect(err).NotTo(HaveOccurred())

			req.Header.Add("Authorization", TEST_USER_TOKEN)
			rsp, err = httpClient.Do(req)
		})

		It("should get 404", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
