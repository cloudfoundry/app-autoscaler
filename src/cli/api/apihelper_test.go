package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	. "autoscaler/models"
	. "cli/api"
	"cli/ui"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("API Helper Test", func() {

	const (
		fakeAppId       string = "fakeAppId"
		fakeAccessToken string = "fakeAccessToken"
	)

	var (
		err        error
		apiServer  *ghttp.Server
		apihelper  *APIHelper
		fakePolicy ScalingPolicy = ScalingPolicy{
			InstanceMin: 1,
			InstanceMax: 2,
			ScalingRules: []*ScalingRule{
				{
					MetricType:            "memoryused",
					StatWindowSeconds:     300,
					BreachDurationSeconds: 600,
					Threshold:             30,
					Operator:              "<=",
					CoolDownSeconds:       300,
					Adjustment:            "-1",
				},
			},
		}
	)

	BeforeEach(func() {

		os.Unsetenv("CF_TRACE")
		apiServer = ghttp.NewServer()
		apihelper = NewAPIHelper(
			&APIEndpoint{
				URL:               apiServer.URL(),
				SkipSSLValidation: false,
			},
			&CFClient{
				CCAPIEndpoint: "fakeCCAPI",
				AuthToken:     fakeAccessToken,
				AppId:         fakeAppId,
				AppName:       "fakeAppName",
			},
			"false",
		)

	})

	AfterEach(func() {
		apiServer.Close()
	})

	Context("Common invalid access errors", func() {

		Context("Server not started", func() {
			BeforeEach(func() {
				apiServer.Close()
			})

			It("Fail to check health", func() {
				err = apihelper.CheckHealth()
				Expect(err).Should(HaveOccurred())
			})

			It("Fail to get policy", func() {
				_, err = apihelper.GetPolicy()
				Expect(err).Should(HaveOccurred())
			})

		})

		Context("No /health endpoint", func() {

			BeforeEach(func() {
				apiServer.RouteToHandler("GET", "/health",
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, ""),
				)
			})

			It("Fail to check health", func() {
				err = apihelper.CheckHealth()
				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError(fmt.Sprintf(ui.InvalidAPIEndpoint, apihelper.Endpoint.URL)))
			})

			It("Fail to get policy", func() {
				_, err = apihelper.GetPolicy()
				Expect(err).Should(HaveOccurred())
			})

		})

		Context("Connecting to self-signed TLS server with --skip-ssl-validation = false", func() {
			var err error
			var apiTLSServer *ghttp.Server
			var apiTLSHelper *APIHelper

			BeforeEach(func() {
				apiTLSServer = ghttp.NewTLSServer()
				apiTLSServer.RouteToHandler("GET", "/health",
					ghttp.RespondWithJSONEncoded(http.StatusOK, ""),
				)
				apiTLSServer.RouteToHandler("GET", "/v1/apps/"+fakeAppId+"/policy",
					ghttp.RespondWithJSONEncoded(http.StatusOK, &fakePolicy),
				)
				apiTLSHelper = NewAPIHelper(
					&APIEndpoint{
						URL: apiTLSServer.URL(),
					},
					&CFClient{
						CCAPIEndpoint: "fakeCCAPI",
						AuthToken:     fakeAccessToken,
						AppId:         fakeAppId,
						AppName:       "fakeAppName",
					},
					"false",
				)

			})

			AfterEach(func() {
				apiTLSServer.Close()
			})

			It("Fail to check health", func() {
				err = apiTLSHelper.CheckHealth()
				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError(fmt.Sprintf(ui.InvalidSSLCerts, apiTLSHelper.Endpoint.URL)))
			})

			It("Fail to Get policy", func() {
				_, err = apiTLSHelper.GetPolicy()
				Expect(err).Should(HaveOccurred())
				Expect(err).Should(MatchError(fmt.Sprintf(ui.InvalidSSLCerts, apiTLSHelper.Endpoint.URL)))
			})

		})

	})

	Context("When API Server is valid", func() {

		BeforeEach(func() {
			apiServer.RouteToHandler("GET", "/health",
				ghttp.RespondWithJSONEncoded(http.StatusOK, ""),
			)
		})

		Context("Check Health", func() {
			It("Succeed", func() {
				err = apihelper.CheckHealth()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Get policy", func() {

			var urlpath string = "/v1/apps/" + fakeAppId + "/policy"

			Context("Succeed with valid auth token", func() {

				BeforeEach(func() {
					apiServer.RouteToHandler("GET", urlpath,
						ghttp.CombineHandlers(
							ghttp.RespondWithJSONEncoded(http.StatusOK, &fakePolicy),
							ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
						),
					)
				})

				It("Succeed", func() {
					response, err := apihelper.GetPolicy()
					Expect(err).NotTo(HaveOccurred())

					var actualPolicy ScalingPolicy
					_ = json.Unmarshal([]byte(response), &actualPolicy)
					Expect(actualPolicy).To(MatchFields(IgnoreExtras, Fields{
						"InstanceMin": BeNumerically("==", fakePolicy.InstanceMin),
						"InstanceMax": BeNumerically("==", fakePolicy.InstanceMax),
					}))

					Expect(*actualPolicy.ScalingRules[0]).To(MatchFields(IgnoreExtras, Fields{
						"MetricType":            Equal(fakePolicy.ScalingRules[0].MetricType),
						"StatWindowSeconds":     BeNumerically("==", fakePolicy.ScalingRules[0].StatWindowSeconds),
						"BreachDurationSeconds": BeNumerically("==", fakePolicy.ScalingRules[0].BreachDurationSeconds),
						"Threshold":             BeNumerically("==", fakePolicy.ScalingRules[0].Threshold),
						"Operator":              Equal(fakePolicy.ScalingRules[0].Operator),
						"CoolDownSeconds":       BeNumerically("==", fakePolicy.ScalingRules[0].CoolDownSeconds),
						"Adjustment":            Equal(fakePolicy.ScalingRules[0].Adjustment),
					}))
				})
			})

			Context("Fail with 401 error", func() {

				BeforeEach(func() {
					apiServer.RouteToHandler("GET", urlpath,
						ghttp.RespondWithJSONEncoded(http.StatusUnauthorized, ""),
					)
				})

				It("Fail with 401 error", func() {
					_, err = apihelper.GetPolicy()
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.Unauthorized, apihelper.Endpoint.URL, apihelper.Client.CCAPIEndpoint)))
				})
			})

			Context("Fail with 404 error", func() {

				BeforeEach(func() {
					apiServer.RouteToHandler("GET", urlpath,
						ghttp.RespondWithJSONEncoded(http.StatusNotFound, ""),
					)
				})

				It("Fail with 404 error", func() {
					_, err = apihelper.GetPolicy()
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(fmt.Sprintf(ui.PolicyNotFound, apihelper.Client.AppName)))
				})
			})

		})

	})

})
