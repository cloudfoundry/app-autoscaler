package integration_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"github.com/google/uuid"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration_Operator_Others", func() {
	var (
		testAppId         string
		testGuid          string
		initInstanceCount = 2
		policyStr         string
		serviceInstanceId string
		bindingId         string
		orgId             string
		spaceId           string

		brokerUrl *url.URL
		err       error
		tmpDir    string
	)

	BeforeEach(func() {
		tmpDir, err = os.MkdirTemp("", "autoscaler")
		Expect(err).NotTo(HaveOccurred())

		httpClient = testhelpers.NewApiClient()

		testAppId = uuid.NewString()
		testGuid = uuid.NewString()
		serviceInstanceId = getRandomIdRef("serviceInstId")
		orgId = getRandomIdRef("orgId")
		spaceId = getRandomIdRef("spaceId")
		bindingId = getRandomIdRef("bindingId")

		startFakeCCNOAAUAA(initInstanceCount)

		golangApiServerConfPath := components.PrepareGolangApiServerConfig(
			dbUrl,
			fakeCCNOAAUAA.URL(),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]),
			fmt.Sprintf("https://127.0.0.1:%d", components.Ports[EventGenerator]),
			tmpDir)

		startGolangApiServer(golangApiServerConfPath)
		brokerAuth = base64.StdEncoding.EncodeToString([]byte("broker_username:broker_password"))

		brokerUrl, err = url.Parse(fmt.Sprintf("https://127.0.0.1:%d", components.Ports[GolangServiceBroker]))
		Expect(err).NotTo(HaveOccurred())

		provisionAndBind(brokerUrl, serviceInstanceId, orgId, spaceId, bindingId, testAppId, httpClientForPublicApi)

		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), defaultHttpClientTimeout, tmpDir)
		startScalingEngine()

		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir, defaultHttpClientTimeout)
		startScheduler()

	})

	JustBeforeEach(func() {
		operatorConfPath = components.PrepareOperatorConfig(dbUrl, fakeCCNOAAUAA.URL(), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[Scheduler]), 10*time.Second, 1*24*time.Hour, defaultHttpClientTimeout, tmpDir)
		startOperator()
	})

	AfterEach(func() {
		_, err := detachPolicy(testAppId, components.Ports[GolangAPIServer], httpClient)
		Expect(err).NotTo(HaveOccurred())
		stopScheduler()
		stopScalingEngine()
		stopOperator()
		stopGolangApiServer()
		os.RemoveAll(tmpDir)
	})

	Describe("Synchronizer", func() {

		Describe("Synchronize the active schedules to scaling engine", func() {

			Context("ScalingEngine Server is down when active_schedule changes", func() {
				JustBeforeEach(func() {
					stopScalingEngine()
				})

				Context("Create an active schedule", func() {

					JustBeforeEach(func() {
						policyStr = setPolicySpecificDateTime(readPolicyFromFile("fakePolicyWithSpecificDateSchedule.json"), 70*time.Second, 2*time.Hour)
						doAttachPolicy(testAppId, []byte(policyStr), http.StatusOK, components.Ports[GolangAPIServer], httpClient)
					})

					It("should sync the active schedule to scaling engine after restart", func() {

						By("ensure scaling server is down when the active schedule is triggered in scheduler")
						Consistently(func() error {
							_, err := getActiveSchedule(testAppId)
							return err
						}, 70*time.Second, 1*time.Second).Should(HaveOccurred())

						By("The active schedule is added into scaling engine")
						startScalingEngine()
						Eventually(func() bool {
							return activeScheduleExists(testAppId)
						}, 2*time.Minute, 5*time.Second).Should(BeTrue())
					})

				})

				Context("Delete an active schedule", func() {
					BeforeEach(func() {
						policyStr = setPolicySpecificDateTime(readPolicyFromFile("fakePolicyWithSpecificDateSchedule.json"), 70*time.Second, 140*time.Second)
						doAttachPolicy(testAppId, []byte(policyStr), http.StatusOK, components.Ports[GolangAPIServer], httpClient)

						//TODO why just sleep for a minute ?
						time.Sleep(70 * time.Second)
						Consistently(func() bool { return activeScheduleExists(testAppId) }).
							WithTimeout(10 * time.Second).
							WithPolling(1 * time.Second).Should(BeTrue())

					})

					It("should delete an active schedule in scaling engine after restart", func() {

						By("ensure scaling server is down when the active schedule is deleted from scheduler")
						//TODO there is a better check than waiting 80 seconds for consecutive errors.
						Consistently(func() error {
							_, err := getActiveSchedule(testAppId)
							return err
						}).WithTimeout(80 * time.Second).
							WithPolling(10 * time.Second).
							Should(HaveOccurred())

						By("The active schedule is removed from scaling engine")
						startScalingEngine()
						Eventually(func() bool { return activeScheduleExists(testAppId) }).
							WithTimeout(2*time.Minute).
							WithPolling(5*time.Second).
							ShouldNot(BeTrue(), "Active schedule should be removed after restart")
					})

				})
			})
		})

		Describe("Synchronize policy DB and scheduler", func() {

			BeforeEach(func() {
				policyStr = string(setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithSchedule.json")))
			})

			AfterEach(func() {
				deletePolicy(testAppId)
			})

			Context("when create an orphan schedule in scheduler without any corresponding policy in policy DB", func() {
				BeforeEach(func() {
					resp, err := createSchedule(testAppId, testGuid, policyStr)
					checkResponseEmptyAndStatusCode(resp, err, http.StatusOK)

					resp, err = getSchedules(testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

				})
				It("operator should remove the orphan schedule ", func() {
					Eventually(func() bool {
						resp, _ := getSchedules(testAppId)
						return resp.StatusCode == http.StatusNotFound
					}, 2*time.Minute, 5*time.Second).Should(BeTrue())

				})
			})

			Context("when insert a policy in policy DB only without creating schedule ", func() {
				BeforeEach(func() {
					insertPolicy(testAppId, policyStr, testGuid)

					resp, err := getSchedules(testAppId)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

				})
				It("operator should sync the schedule to scheduler ", func() {
					Eventually(func() bool {
						resp, _ := getSchedules(testAppId)
						return resp.StatusCode == http.StatusOK
					}, 2*time.Minute, 5*time.Second).Should(BeTrue())

				})
			})

			Context("when update a policy to another schedule sets only in policy DB without any update in scheduler ", func() {
				BeforeEach(func() {
					doAttachPolicy(testAppId, []byte(policyStr), http.StatusOK, components.Ports[GolangAPIServer], httpClient)
					assertScheduleContents(testAppId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})

					newPolicyStr := string(setPolicyRecurringDate(readPolicyFromFile("fakePolicyWithScheduleAnother.json")))
					deletePolicy(testAppId)
					insertPolicy(testAppId, newPolicyStr, testGuid)

					By("the schedules should not be updated before operator triggers the sync")
					assertScheduleContents(testAppId, http.StatusOK, map[string]int{"recurring_schedule": 4, "specific_date": 2})
				})

				It("operator should sync the updated schedule to scheduler ", func() {
					Eventually(func() bool {
						return checkScheduleContents(testAppId, http.StatusOK, map[string]int{"recurring_schedule": 3, "specific_date": 1})
					}, 2*time.Minute, 5*time.Second).Should(BeTrue())

				})
			})

		})

	})

	Describe("Pruner", func() {

		BeforeEach(func() {
			appmetric := &models.AppMetric{
				AppId:      testAppId,
				MetricType: models.MetricNameMemoryUsed,
				Unit:       models.UnitMegaBytes,
				Value:      "123456",
				Timestamp:  time.Now().Add(-24 * time.Hour).UnixNano(),
			}
			insertAppMetric(appmetric)
			Expect(getAppMetricTotalCount(testAppId)).To(Equal(1))

			history := &models.AppScalingHistory{
				AppId:        testAppId,
				Timestamp:    time.Now().Add(-24 * time.Hour).UnixNano(),
				OldInstances: 2,
				NewInstances: 4,
				Reason:       "a reason",
				Message:      "a message",
				ScalingType:  models.ScalingTypeDynamic,
				Status:       models.ScalingStatusSucceeded,
				Error:        "",
			}
			insertScalingHistory(history)
			Expect(getScalingHistoryTotalCount(testAppId)).To(Equal(1))

		})

		It("operator should remove the stale records ", func() {
			Eventually(func() bool {
				return getScalingHistoryTotalCount(testAppId) == 0
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())
		})
	})
})
