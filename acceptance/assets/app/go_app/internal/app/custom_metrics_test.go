package app_test

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-logr/logr"

	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app"
	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app/appfakes"
	api "code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/custommetrics"
	"github.com/cloudfoundry-community/go-cfenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("custom metrics tests", func() {
	var fakeCustomMetricClient *appfakes.FakeCustomMetricClient
	Context("custom metrics handler", func() {

		It("should err if value out of bounds", func() {
			apiTest(nil, nil, nil, nil).
				Get("/custom-metrics/test/100001010101010249032897287298719874687936483275648273632429479827398798271").
				Expect(GinkgoT()).
				Status(http.StatusBadRequest).
				Body(`{"error":{"description":"invalid metric value: strconv.ParseUint: parsing \"100001010101010249032897287298719874687936483275648273632429479827398798271\": value out of range"}}`).
				End()
		})
		It("should err if value not a number", func() {
			apiTest(nil, nil, nil, nil).
				Get("/custom-metrics/test/invalid").
				Expect(GinkgoT()).
				Status(http.StatusBadRequest).
				Body(`{"error":{"description":"invalid metric value: strconv.ParseUint: parsing \"invalid\": invalid syntax"}}`).
				End()
		})
		It("should post the custom metric", func() {
			fakeCustomMetricClient = &appfakes.FakeCustomMetricClient{}
			apiTest(nil, nil, nil, fakeCustomMetricClient).
				Get("/custom-metrics/test/4").
				Expect(GinkgoT()).
				Status(http.StatusOK).
				Body(`{"mtls":false}`).
				End()
			Expect(fakeCustomMetricClient.PostCustomMetricCallCount()).To(Equal(1))
			_, _, appConfig, sentValue, sentMetric, mtlsUsed := fakeCustomMetricClient.PostCustomMetricArgsForCall(0)
			Expect(appConfig.AppID).Should(Equal(""))
			Expect(sentMetric).Should(Equal("test"))
			Expect(sentValue).Should(Equal(4.0))
			Expect(mtlsUsed).Should(Equal(false))
		})
		When("appToScaleGuid is provided in a producer-consumer relationship", func() {
			fakeCustomMetricClient = &appfakes.FakeCustomMetricClient{}
			It("should post the custom metric with appToScaleGuid", func() {
				fakeCustomMetricClient := &appfakes.FakeCustomMetricClient{}
				apiTest(nil, nil, nil, fakeCustomMetricClient).
					Get("/custom-metrics/test/5").
					QueryParams(map[string]string{"appToScaleGuid": "test-app-id"}).
					Expect(GinkgoT()).
					Status(http.StatusOK).
					Body(`{"mtls":false}`).
					End()
				Expect(fakeCustomMetricClient.PostCustomMetricCallCount()).To(Equal(1))
				_, _, appConfig, sentValue, sentMetric, mtlsUsed := fakeCustomMetricClient.PostCustomMetricArgsForCall(0)
				Expect(appConfig.AppID).Should(Equal("test-app-id"))
				Expect(sentMetric).Should(Equal("test"))
				Expect(sentValue).Should(Equal(5.0))
				Expect(mtlsUsed).Should(Equal(false))
			})
		})

	})
	Context("PostCustomMetrics", func() {
		var (
			service    cfenv.Service
			testAppId  string
			fakeServer *ghttp.Server
		)
		BeforeEach(func() {

			testAppId = "test-app-id"
			fakeServer = ghttp.NewServer()
			username := "test-user"
			password := "test-pass"
			fakeServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/v1/apps/"+testAppId+"/metrics"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.VerifyJSON(`{
													  "instance_index": 0,
													  "metrics": [
														{
														  "name": "test",
														  "value": 42
														}
													  ]
													}`,
					),
				),
			)

			customMetricsCredentials := api.CustomMetricsCredentials{
				Username: username,
				Password: password,
				URL:      fakeServer.URL(),
			}
			service = cfenv.Service{
				Name:        "test",
				Tags:        []string{"app-autoscaler"},
				Credentials: map[string]interface{}{"custom_metrics": customMetricsCredentials},
			}
		})
		It("should post a custom metric", func() {
			app.CfenvCurrent = func() (*cfenv.App, error) {
				return &cfenv.App{
					AppID:    testAppId,
					Index:    0,
					Services: map[string][]cfenv.Service{"autoscaler": {service}},
				}, nil
			}
			appEnv, _ := app.CfenvCurrent()
			client := &app.CustomMetricAPIClient{}
			err := client.PostCustomMetric(context.TODO(), logr.Logger{}, appEnv, 42, "test", false)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(fakeServer.ReceivedRequests())).To(Equal(1))
			fakeServer.Close()
		})
		Context("verify configs", func() {
			When("cloud foundry environment is not found", func() {
				It("should return error if cloud foundry environment is not found", func() {
					app.CfenvCurrent = func() (*cfenv.App, error) {
						return nil, errors.New("cloud foundry environment not found")
					}
					client := &app.CustomMetricAPIClient{}
					err := client.PostCustomMetric(context.TODO(), logr.Logger{}, &cfenv.App{}, 42, "test", false)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("cloud foundry environment not found"))
				})
			})
			When("appToScaleGuid is provided in a producer-consumer relationship", func() {
				It("should set appConfig with appToScaleGuid and services", func() {

					app.CfenvCurrent = func() (*cfenv.App, error) {
						return &cfenv.App{AppID: testAppId, Services: map[string][]cfenv.Service{"autoscaler": {service}}}, nil
					}
					appConfig, _ := app.CfenvCurrent()
					client := &app.CustomMetricAPIClient{}
					err := client.PostCustomMetric(context.TODO(), logr.Logger{}, appConfig, 42, "test", false)
					Expect(err).ToNot(HaveOccurred())
					Expect(appConfig.Services).NotTo(BeNil())
					Expect(appConfig.AppID).To(Equal(testAppId))
					Expect(len(fakeServer.ReceivedRequests())).To(Equal(1))
					fakeServer.Close()
				})
			})

		})

	})

})
