package server_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	_ "github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Helper function to create a basic auth string
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// Helper function to create a new request
func newRequest(method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}

// Helper function to set up a new client and request
func setupRequest(method, url, authHeader string, body []byte) (*http.Client, *http.Request, error) {
	client := &http.Client{}
	req, err := newRequest(method, url, body)
	if err != nil {
		return nil, nil, err
	}
	if authHeader != "" {
		req.Header.Add("Authorization", authHeader)
	}
	return client, req, nil
}

var _ = Describe("CustomMetricsConfig Server", func() {
	var (
		resp          *http.Response
		req           *http.Request
		body          []byte
		err           error
		scalingPolicy *models.ScalingPolicy
		client        *http.Client
		authHeader    string
	)

	BeforeEach(func() {
		client = &http.Client{}
		fakeCredentials.ValidateReturns(true, nil)
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

			authHeader = "Basic " + basicAuth("username", "Password")
			client, req, err = setupRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", authHeader, body)
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

			client, req, err = setupRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", "", body)
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

			authHeader = basicAuth("username", "password")
			client, req, err = setupRequest("POST", serverUrl+"/v1/apps/san-app-id/metrics", authHeader, body)
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

			authHeader = "Basic " + basicAuth("invalidUsername", "invalidPassword")
			client, req, err = setupRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", authHeader, body)
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

			authHeader = "Basic " + basicAuth("username", "password")
			client, req, err = setupRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", authHeader, body)
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

			authHeader = "Basic " + basicAuth("username", "password")
			client, req, err = setupRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", authHeader, body)
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
		When("username and password are incorrect for basic authentication during health check", func() {
			It("should return 401", func() {
				client := &http.Client{}
				req, err = http.NewRequest("GET", serverUrl+"/health", nil)
				Expect(err).NotTo(HaveOccurred())
				req.SetBasicAuth("wrongusername", "wrongpassword")
				rsp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		When("username and password are correct for basic authentication during health check", func() {
			BeforeEach(func() {
				client = &http.Client{}
			})

			When("a request to query health comes", func() {
				It("returns with a 200", func() {
					req, err = http.NewRequest("GET", serverUrl, nil)
					Expect(err).NotTo(HaveOccurred())
					req.SetBasicAuth(conf.Health.HealthCheckUsername, conf.Health.HealthCheckPassword)
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
				req, err = http.NewRequest("GET", serverUrl+"/health", nil)
				Expect(err).NotTo(HaveOccurred())
				req.SetBasicAuth(conf.Health.HealthCheckUsername, conf.Health.HealthCheckPassword)
				rsp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})

			It("should return 200 for /health/readiness", func() {
				req, err = http.NewRequest("GET", serverUrl+"/health/readiness", nil)
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
