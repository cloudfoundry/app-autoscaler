package server_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CustomMetrics Server", func() {
	var (
		resp          *http.Response
		req           *http.Request
		body          []byte
		err           error
		scalingPolicy *models.ScalingPolicy
	)

	Context("when a request to forward custom metrics comes", func() {
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
				{
					Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
				},
			}
			body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
			Expect(err).NotTo(HaveOccurred())

			fakeCredentials.ValidateReturns(true, nil)

			client := &http.Client{}
			req, err = http.NewRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 200", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			resp.Body.Close()
		})
	})

	Context("when a request to forward custom metrics comes without Authorization header", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{}
			req, err = http.NewRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")

			fakeCredentials.ValidateReturns(true, nil)

			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 401", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			resp.Body.Close()
		})
	})

	Context("when a request to forward custom metrics comes without 'Basic'", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{}
			req, err = http.NewRequest("POST", serverUrl+"/v1/apps/san-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "dXNlcm5hbWU6cGFzc3dvcmQ=")

			fakeCredentials.ValidateReturns(true, nil)

			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 401", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			resp.Body.Close()
		})
	})

	Context("when a request to forward custom metrics comes with wrong user credentials", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())

			fakeCredentials.ValidateReturns(false, errors.New("wrong credentials"))

			client := &http.Client{}
			req, err = http.NewRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Basic M2YxZWY2MTJiMThlYTM5YmJlODRjZjUxMzY4MWYwYjc6YWYyNjk1Y2RmZDE0MzA4NThhMWY3MzJhYTI5NTQ2ZTk=")

			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 401", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			resp.Body.Close()
		})
	})

	Context("when a request to forward custom metrics comes with unmatched metric types", func() {
		BeforeEach(func() {
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())

			fakeCredentials.ValidateReturns(true, nil)

			client := &http.Client{}
			req, err = http.NewRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 400", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			resp.Body.Close()
		})
	})

	Context("when multiple requests to forward custom metrics comes beyond ratelimit", func() {
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
				{
					Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
				},
			}
			body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
			Expect(err).NotTo(HaveOccurred())

			fakeCredentials.ValidateReturns(true, nil)

			client := &http.Client{}
			req, err = http.NewRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			rateLimiter.ExceedsLimitReturns(false)
		})

		It("returns status code 429", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusTooManyRequests))
			resp.Body.Close()
		})
	})

})
