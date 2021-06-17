package server_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"autoscaler/models"

	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CustomMetrics Server", func() {
	var (
		resp          *http.Response
		req           *http.Request
		body          []byte
		err           error
		credentials   *models.Credential
		scalingPolicy *models.ScalingPolicy
	)

	Context("when a request to forward custom metrics comes", func() {
		BeforeEach(func() {
			credentials = &models.Credential{}
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
				&models.CustomMetric{
					Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
				},
			}
			body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
			Expect(err).NotTo(HaveOccurred())
			credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
			credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
			credentialCache.Set("an-app-id", credentials, 10*time.Minute)
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
			credentials = &models.Credential{}
			credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
			credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
			credentialCache.Set("an-app-id", credentials, 10*time.Minute)
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{}
			req, err = http.NewRequest("POST", serverUrl+"/v1/apps/an-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
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
			credentials = &models.Credential{}
			credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
			credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
			credentialCache.Set("an-app-id", credentials, 10*time.Minute)
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())
			client := &http.Client{}
			req, err = http.NewRequest("POST", serverUrl+"/v1/apps/san-app-id/metrics", bytes.NewReader(body))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", "dXNlcm5hbWU6cGFzc3dvcmQ=")
			resp, err = client.Do(req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns status code 401", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			resp.Body.Close()
		})
	})

	Context("when a request to forward custom metrics comes with  wrong user credentials", func() {
		BeforeEach(func() {
			credentials = &models.Credential{}
			credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
			credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
			credentialCache.Set("an-app-id", credentials, 10*time.Minute)
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())
			policyDB.GetCredentialReturns(nil, sql.ErrNoRows)
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
			credentials = &models.Credential{}
			credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
			credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
			credentialCache.Set("an-app-id", credentials, 10*time.Minute)
			body, err = json.Marshal(models.CustomMetric{Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 123, AppGUID: "an-app-id"})
			Expect(err).NotTo(HaveOccurred())
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
			credentials = &models.Credential{}
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
				&models.CustomMetric{
					Name: "queuelength", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
				},
			}
			body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
			Expect(err).NotTo(HaveOccurred())
			credentials.Username = "$2a$10$YnQNQYcvl/Q2BKtThOKFZ.KB0nTIZwhKr5q1pWTTwC/PUAHsbcpFu"
			credentials.Password = "$2a$10$6nZ73cm7IV26wxRnmm5E1.nbk9G.0a4MrbzBFPChkm5fPftsUwj9G"
			credentialCache.Set("an-app-id", credentials, 10*time.Minute)
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
