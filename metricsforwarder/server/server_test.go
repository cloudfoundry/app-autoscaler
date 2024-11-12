package server_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Helper function to set up a new client and request
func setupRequest(method string, url *url.URL, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

var _ = Describe("CustomMetricsConfig Server", func() {

	var (
		resp          *http.Response
		req           *http.Request
		body          []byte
		err           error
		scalingPolicy *models.ScalingPolicy
		client        *http.Client

		serverURL *url.URL
		healthURL *url.URL
	)

	BeforeEach(func() {
		client = &http.Client{}
		fakeCredentials.ValidateReturns(true, nil)

		serverURL, err = url.Parse(fmt.Sprintf("http://127.0.0.1:%d", conf.Server.Port))
		Expect(err).NotTo(HaveOccurred())

		// health url runs on the same port as metricsforwarder, maybe we need to roll back to use original port
		healthURL, err = url.Parse(fmt.Sprintf("http://127.0.0.1:%d", conf.Server.Port))
		Expect(err).NotTo(HaveOccurred())
	})

	When("POST /v1/apps/some-app-id/metrics", func() {
		BeforeEach(func() {
			scalingPolicy = &models.ScalingPolicy{
				InstanceMin: 1,
				InstanceMax: 6,
				ScalingRules: []*models.ScalingRule{{
					MetricType:            "queuelength",
					BreachDurationSeconds: 60,
					Threshold:             10,
					Operator:              ">",
					CoolDownSeconds:       60,
					Adjustment:            "+1"}}}
			policyDB.GetAppPolicyReturns(scalingPolicy, nil)
			customMetrics := []*models.CustomMetric{
				{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id"},
			}
			body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
			Expect(err).NotTo(HaveOccurred())

			serverURL.Path = "/v1/apps/an-app-id/metrics"
			req, err = setupRequest("POST", serverURL, body)
			req.SetBasicAuth("username", "password")
			Expect(err).NotTo(HaveOccurred())
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("returns status code 200", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			resp.Body.Close()
		})
	})

	When("A request to forward custom metrics comes without Authorization header", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())

			serverURL.Path = "/v1/apps/an-app-id/metrics"
			req, err = setupRequest("POST", serverURL, body)
			Expect(err).NotTo(HaveOccurred())
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 401", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			resp.Body.Close()
		})
	})

	When("a request to forward custom metrics comes without 'Basic'", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())

			serverURL.Path = "/v1/apps/an-app-id/metrics"
			req, err = setupRequest("POST", serverURL, body)
			Expect(err).NotTo(HaveOccurred())
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 401", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			resp.Body.Close()
		})
	})

	When("a request to forward custom metrics comes with wrong user credentials", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())

			fakeCredentials.ValidateReturns(false, errors.New("wrong credentials"))

			serverURL.Path = "/v1/apps/an-app-id/metrics"
			req, err = setupRequest("POST", serverURL, body)
			req.SetBasicAuth("invalidUsername", "invalidPassword")
			Expect(err).NotTo(HaveOccurred())
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 401", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			resp.Body.Close()
		})
	})

	When("a request to forward custom metrics comes with unmatched metric types", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())

			serverURL.Path = "/v1/apps/an-app-id/metrics"
			req, err = setupRequest("POST", serverURL, body)
			req.SetBasicAuth("username", "password")
			Expect(err).NotTo(HaveOccurred())
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 400", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			resp.Body.Close()
		})
	})

	When("multiple requests to forward custom metrics come beyond rate limit", func() {
		BeforeEach(func() {
			rateLimiter.ExceedsLimitReturns(true)
			scalingPolicy = &models.ScalingPolicy{
				InstanceMin: 1,
				InstanceMax: 6,
				ScalingRules: []*models.ScalingRule{{
					MetricType:            "queuelength",
					BreachDurationSeconds: 60,
					Threshold:             10,
					Operator:              ">",
					CoolDownSeconds:       60,
					Adjustment:            "+1"}}}
			policyDB.GetAppPolicyReturns(scalingPolicy, nil)
			customMetrics := []*models.CustomMetric{
				{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id"},
			}
			body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
			Expect(err).NotTo(HaveOccurred())

			serverURL.Path = "/v1/apps/an-app-id/metrics"
			req, err = setupRequest("POST", serverURL, body)
			req.SetBasicAuth("username", "password")
			Expect(err).NotTo(HaveOccurred())
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			rateLimiter.ExceedsLimitReturns(false)
		})

		It("returns status code 429", func() {
			Expect(resp.StatusCode).To(Equal(http.StatusTooManyRequests))
			resp.Body.Close()
		})
	})

	When("the Health server is ready to serve RESTful API with basic Auth", func() {
		var client *http.Client

		BeforeEach(func() {
			healthURL.Path = "/health"
			client = &http.Client{}
		})

		When("username and password are incorrect for basic authentication during health check", func() {
			It("should return 401", func() {
				req, err = http.NewRequest("GET", healthURL.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				req.SetBasicAuth("wrongusername", "wrongpassword")
				rsp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		When("username and password are correct for basic authentication during health check", func() {
			When("a request to query health comes", func() {
				It("returns with a 200", func() {
					req, err = http.NewRequest("GET", healthURL.String(), nil)
					Expect(err).NotTo(HaveOccurred())
					req.SetBasicAuth(conf.Health.BasicAuth.Username, conf.Health.BasicAuth.Password)
					rsp, err := client.Do(req)
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					raw, _ := io.ReadAll(rsp.Body)
					healthData := string(raw)
					Expect(healthData).To(ContainSubstring("autoscaler_metricsforwarder_concurrent_http_request"))
					Expect(healthData).To(ContainSubstring("autoscaler_metricsforwarder_policyDB"))
					Expect(healthData).To(ContainSubstring("go_goroutines"))
					Expect(healthData).To(ContainSubstring("go_memstats_alloc_bytes"))
					rsp.Body.Close()
				})
			})

			It("should return 200 for /health", func() {
				healthURL.Path = "/health"
				req, err = http.NewRequest("GET", healthURL.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				req.SetBasicAuth(conf.Health.BasicAuth.Username, conf.Health.BasicAuth.Password)
				rsp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})

			It("should return 200 for /health/readiness", func() {
				healthURL.Path = "/health/readiness"
				req, err = http.NewRequest("GET", healthURL.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				rsp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				body, err := io.ReadAll(rsp.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(MatchJSON(`{"overall_status": "UP","checks": [ { "name":"policy_db", "type":"database", "status":"UP"}, { "name":"storedprocedure_db", "type":"database", "status":"UP"}]}`))
			})
		})
	})
})
