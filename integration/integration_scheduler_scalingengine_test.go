package integration

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration_Scheduler_ScalingEngine", func() {
	var (
		testAppId         string
		testGuid          string
		initInstanceCount int = 2
		policyStr         string
	)

	BeforeEach(func() {
		initializeHttpClient("scheduler.crt", "scheduler.key", "autoscaler-ca.crt", schedulerScalingEngineHttpRequestTimeout)

		testAppId = getRandomId()
		testGuid = getRandomId()
		startFakeCCNOAAUAA(initInstanceCount)

		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), defaultHttpClientTimeout, tmpDir)
		startScalingEngine()

		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir, defaultHttpClientTimeout)
		startScheduler()

		policyStr = setPolicySpecificDateTime(readPolicyFromFile("fakePolicyWithSpecificDateSchedule.json"), 70*time.Second, 2*time.Hour)

	})

	AfterEach(func() {
		deletePolicy(testAppId)
		stopScheduler()
		stopScalingEngine()
	})

	Describe("Create Schedule", func() {
		Context("Valid specific date schedule", func() {

			AfterEach(func() {
				deleteSchedule(testAppId)
			})

			It("creates active schedule in scaling engine", func() {
				resp, err := createSchedule(testAppId, testGuid, policyStr)
				checkResponseEmptyAndStatusCode(resp, err, http.StatusOK)

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 1*time.Second).Should(BeTrue())

			})
		})

	})

	Describe("Delete Schedule", func() {
		BeforeEach(func() {
			resp, err := createSchedule(testAppId, testGuid, policyStr)
			checkResponseEmptyAndStatusCode(resp, err, http.StatusOK)

			Eventually(func() bool {
				return activeScheduleExists(testAppId)
			}, 2*time.Minute, 1*time.Second).Should(BeTrue())
		})

		It("deletes active schedule in scaling engine", func() {
			resp, err := deleteSchedule(testAppId)
			checkResponseEmptyAndStatusCode(resp, err, http.StatusNoContent)

			Eventually(func() bool {
				return activeScheduleExists(testAppId)
			}, 2*time.Minute, 1*time.Second).Should(BeFalse())
		})
	})

})
