package integration_test

import (
	"fmt"
	"net/http"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Integration_GolangApi_ScalingEngine", func() {
	var (
		initInstanceCount = 2
		appId             string
		pathVariables     []string
		parameters        map[string]string
	)

	BeforeEach(func() {
		initializeHttpClient("api.crt", "api.key", "autoscaler-ca.crt", apiScalingEngineHttpRequestTimeout)
		initializeHttpClientForPublicApi("api_public.crt", "api_public.key", "autoscaler-ca.crt", apiMetricsCollectorHttpRequestTimeout)
		startFakeCCNOAAUAA(initInstanceCount)
		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), defaultHttpClientTimeout, tmpDir)
		startScalingEngine()

		golangApiServerConfPath = components.PrepareGolangApiServerConfig(
			dbUrl,
			components.Ports[GolangAPIServer],
			components.Ports[GolangServiceBroker],
			fakeCCNOAAUAA.URL(),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]),
			"https://127.0.0.1:8888",
			true,
			tmpDir)
		startGolangApiServer()
		appId = getRandomId()
		pathVariables = []string{appId}

	})

	AfterEach(func() {
		stopGolangApiServer()
		stopScalingEngine()
	})
	Describe("Get scaling histories", func() {

		Context("Cloud Controller api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "desc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 500", func() {
				checkPublicAPIResponseContentWithParameters(getScalingHistories, components.Ports[GolangAPIServer], pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{
					"code":    "Internal-Server-Error",
					"message": "Failed to check if user is admin",
				})
			})
		})

		Context("UAA api is not available", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
					cf.Endpoints{
						AuthEndpoint:    fakeCCNOAAUAA.URL(),
						TokenEndpoint:   fakeCCNOAAUAA.URL(),
						DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
					}))
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "desc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 500", func() {
				checkPublicAPIResponseContentWithParameters(getScalingHistories, components.Ports[GolangAPIServer], pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{
					"code":    "Internal-Server-Error",
					"message": "Failed to check if user is admin",
				})
			})
		})

		Context("UAA api returns 401", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.Reset()
				fakeCCNOAAUAA.AllowUnhandledRequests = true
				fakeCCNOAAUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
					cf.Endpoints{
						AuthEndpoint:    fakeCCNOAAUAA.URL(),
						TokenEndpoint:   fakeCCNOAAUAA.URL(),
						DopplerEndpoint: strings.Replace(fakeCCNOAAUAA.URL(), "http", "ws", 1),
					}))
				fakeCCNOAAUAA.RouteToHandler("POST", "/check_token", ghttp.RespondWithJSONEncoded(http.StatusOK,
					struct {
						Scope []string `json:"scope"`
					}{
						[]string{"cloud_controller.read", "cloud_controller.write", "password.write", "openid", "network.admin", "network.write", "uaa.user"},
					}))
				fakeCCNOAAUAA.RouteToHandler("GET", "/userinfo", ghttp.RespondWithJSONEncoded(http.StatusUnauthorized, struct{}{}))
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "desc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 401", func() {
				checkPublicAPIResponseContentWithParameters(getScalingHistories, components.Ports[GolangAPIServer],
					pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{
						"code":    "Unauthorized",
						"message": "You are not authorized to perform the requested action",
					})
			})
		})

		Context("Check permission not passed", func() {
			BeforeEach(func() {
				fakeCCNOAAUAA.RouteToHandler("GET", rolesRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
					struct {
						Pagination struct {
							Total int `json:"total_results"`
						} `json:"pagination"`
					}{}))
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "desc", "page": "1", "results-per-page": "5"}
			})
			It("should error with status code 401", func() {
				checkPublicAPIResponseContentWithParameters(getScalingHistories, components.Ports[GolangAPIServer],
					pathVariables, parameters, http.StatusUnauthorized, map[string]interface{}{
						"code":    "Unauthorized",
						"message": "You are not authorized to perform the requested action",
					})
			})
		})

		Context("ScalingEngine is down", func() {
			JustBeforeEach(func() {
				stopScalingEngine()
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order-direction": "desc", "page": "1", "results-per-page": "5"}
			})

			It("should error with status code 500", func() {
				checkPublicAPIResponseContentWithParameters(getScalingHistories, components.Ports[GolangAPIServer], pathVariables, parameters, http.StatusInternalServerError, map[string]interface{}{
					"message": "Error retrieving scaling history from scaling engine",
					"code":    "Internal Server Error",
				})

			})

		})

		Context("Get scaling histories", func() {
			BeforeEach(func() {
				history1 := createScalingHistory(appId, 666666)
				insertScalingHistory(&history1)

				history2 := createScalingHistory(appId, 222222)
				insertScalingHistory(&history2)

				history3 := createScalingHistory(appId, 555555)
				insertScalingHistory(&history3)

				history4 := createScalingHistory(appId, 333333)
				insertScalingHistory(&history4)

				history5 := createScalingHistory(appId, 444444)
				insertScalingHistory(&history5)

				//add some other app id
				history6 := createScalingHistory(getRandomId(), 444444)
				insertScalingHistory(&history6)
			})
			It("should get the scaling histories ", func() {
				By("get the 1st page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "desc", "page": "1", "results-per-page": "2"}
				result := ScalingHistoryResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         1,
					NextUrl:      getScalingHistoriesUrl(appId, parameters, 2),
					Resources: []models.AppScalingHistory{
						{
							AppId:        appId,
							Timestamp:    666666,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
						{
							AppId:        appId,
							Timestamp:    555555,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
					},
				}
				checkScalingHistoryResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("get the 2nd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "desc", "page": "2", "results-per-page": "2"}
				result = ScalingHistoryResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         2,
					PrevUrl:      getScalingHistoriesUrl(appId, parameters, 1),
					NextUrl:      getScalingHistoriesUrl(appId, parameters, 3),
					Resources: []models.AppScalingHistory{
						{
							AppId:        appId,
							Timestamp:    444444,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
						{
							AppId:        appId,
							Timestamp:    333333,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
					},
				}
				checkScalingHistoryResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("get the 3rd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "desc", "page": "3", "results-per-page": "2"}
				result = ScalingHistoryResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         3,
					PrevUrl:      getScalingHistoriesUrl(appId, parameters, 2),
					Resources: []models.AppScalingHistory{
						{
							AppId:        appId,
							Timestamp:    222222,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
					},
				}
				checkScalingHistoryResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("the 4th page should be empty")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order-direction": "desc", "page": "4", "results-per-page": "2"}
				result = ScalingHistoryResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         4,
					PrevUrl:      getScalingHistoriesUrl(appId, parameters, 3),
					Resources:    []models.AppScalingHistory{},
				}
				checkScalingHistoryResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)
			})
			It("should get the scaling histories in specified time scope", func() {
				By("get the results from 555555")
				parameters = map[string]string{"start-time": "555555", "order-direction": "desc", "page": "1", "results-per-page": "10"}
				result := ScalingHistoryResult{
					TotalResults: 2,
					TotalPages:   1,
					Page:         1,
					Resources: []models.AppScalingHistory{
						{
							AppId:        appId,
							Timestamp:    666666,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
						{
							AppId:        appId,
							Timestamp:    555555,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
					},
				}
				checkScalingHistoryResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("get the results to 333333")
				parameters = map[string]string{"end-time": "333333", "order-direction": "desc", "page": "1", "results-per-page": "10"}
				result = ScalingHistoryResult{
					TotalResults: 2,
					TotalPages:   1,
					Page:         1,
					Resources: []models.AppScalingHistory{
						{
							AppId:        appId,
							Timestamp:    333333,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
						{
							AppId:        appId,
							Timestamp:    222222,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
					},
				}
				checkScalingHistoryResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)

				By("get the results from 333333 to 555555")
				parameters = map[string]string{"start-time": "333333", "end-time": "555555", "order-direction": "asc", "page": "1", "results-per-page": "10"}
				result = ScalingHistoryResult{
					TotalResults: 3,
					TotalPages:   1,
					Page:         1,
					Resources: []models.AppScalingHistory{
						{
							AppId:        appId,
							Timestamp:    333333,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
						{
							AppId:        appId,
							Timestamp:    444444,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
						{
							AppId:        appId,
							Timestamp:    555555,
							ScalingType:  models.ScalingTypeDynamic,
							Status:       models.ScalingStatusSucceeded,
							OldInstances: 2,
							NewInstances: 4,
							Reason:       "a reason",
							Message:      "a message",
							Error:        "",
						},
					},
				}
				checkScalingHistoryResult(components.Ports[GolangAPIServer], pathVariables, parameters, result)
			})

		})
	})
})
