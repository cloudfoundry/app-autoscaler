package integration_test

import (
	"autoscaler/cf"
	"autoscaler/models"
	"encoding/json"
	"fmt"
	. "integration"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type ScalingHistoryResult struct {
	TotalResults int                        `json:"total_results"`
	TotalPages   int                        `json:"total_pages"`
	Page         int                        `json:"page"`
	Resources    []models.AppScalingHistory `json:"resources"`
}

var _ = Describe("Integration_Api_ScalingEngine", func() {
	var (
		initInstanceCount int = 2
		appId             string
		parameters        map[string]string
		history           *models.AppScalingHistory
	)

	BeforeEach(func() {
		initializeHttpClient("api.crt", "api.key", "autoscaler-ca.crt", apiScalingEngineHttpRequestTimeout)
		startFakeCCNOAAUAA(initInstanceCount)
		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, tmpDir, consulRunner.ConsulCluster())
		startScalingEngine()

		apiServerConfPath = components.PrepareApiServerConfig(components.Ports[APIServer], dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir)
		startApiServer()
		appId = getRandomId()

	})

	AfterEach(func() {
		stopApiServer()
		stopScalingEngine()
	})
	Describe("Get scaling histories", func() {
		Context("ScalingEngine is down", func() {
			JustBeforeEach(func() {
				stopScalingEngine()
				parameters = map[string]string{"start-time": "1111", "end-time": "9999", "order": "desc", "page": "0", "results-per-page": "5"}
			})

			AfterEach(func() {
				startScalingEngine()
			})

			It("should error", func() {
				checkResponseContentWithParameters(getScalingHistories, appId, parameters, http.StatusInternalServerError, map[string]interface{}{"description": fmt.Sprintf("connect ECONNREFUSED 127.0.0.1:%d", components.Ports[ScalingEngine])})

			})

		})

		Context("Get scaling histories", func() {
			BeforeEach(func() {
				history = &models.AppScalingHistory{
					AppId:        appId,
					OldInstances: 2,
					NewInstances: 4,
					Reason:       "a reason",
					Message:      "a message",
					ScalingType:  models.ScalingTypeDynamic,
					Status:       models.ScalingStatusSucceeded,
					Error:        "",
				}

				history.Timestamp = 666666
				insertScalingHistory(history)

				history.Timestamp = 222222
				insertScalingHistory(history)

				history.Timestamp = 555555
				insertScalingHistory(history)

				history.Timestamp = 333333
				insertScalingHistory(history)

				history.Timestamp = 444444
				insertScalingHistory(history)
			})
			It("should get the scaling histories ", func() {
				By("get the 1st page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order": "desc", "page": "0", "results-per-page": "2"}
				result := ScalingHistoryResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         0,
					Resources: []models.AppScalingHistory{
						models.AppScalingHistory{
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
						models.AppScalingHistory{
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
				checkScalingHistoryResult(appId, parameters, result)

				By("get the 2nd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order": "desc", "page": "1", "results-per-page": "2"}
				result = ScalingHistoryResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         1,
					Resources: []models.AppScalingHistory{
						models.AppScalingHistory{
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
						models.AppScalingHistory{
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
				checkScalingHistoryResult(appId, parameters, result)

				By("get the 3rd page")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order": "desc", "page": "2", "results-per-page": "2"}
				result = ScalingHistoryResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         2,
					Resources: []models.AppScalingHistory{
						models.AppScalingHistory{
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
				checkScalingHistoryResult(appId, parameters, result)

				By("the 4th page should be empty")
				parameters = map[string]string{"start-time": "111111", "end-time": "999999", "order": "desc", "page": "3", "results-per-page": "2"}
				result = ScalingHistoryResult{
					TotalResults: 5,
					TotalPages:   3,
					Page:         3,
					Resources:    []models.AppScalingHistory{},
				}
				checkScalingHistoryResult(appId, parameters, result)
			})

		})
	})
})

func checkScalingHistoryResult(appId string, parameters map[string]string, result ScalingHistoryResult) {
	var actual ScalingHistoryResult
	resp, err := getScalingHistories(appId, parameters)
	defer resp.Body.Close()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = json.NewDecoder(resp.Body).Decode(&actual)
	Expect(err).NotTo(HaveOccurred())
	Expect(actual).To(Equal(result))

}
