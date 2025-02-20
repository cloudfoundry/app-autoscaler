package integration_test

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"github.com/google/uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration_Scheduler_ScalingEngine", func() {
	var (
		testAppId         string
		testGuid          string
		initInstanceCount = 2
		policyStr         string
		tmpDir            string
		err               error
	)

	BeforeEach(func() {
		tmpDir, err = os.MkdirTemp("", "autoscaler")
		Expect(err).NotTo(HaveOccurred())

		httpClient = testhelpers.NewSchedulerClient()

		testAppId = uuid.NewString()
		testGuid = uuid.NewString()
		startFakeCCNOAAUAA(initInstanceCount)

		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), defaultHttpClientTimeout, tmpDir)
		startScalingEngine()

		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir, defaultHttpClientTimeout)
		startScheduler()

		policyStr = setPolicySpecificDateTime(readPolicyFromFile("fakePolicyWithSpecificDateSchedule.json"), 70*time.Second, 2*time.Hour)
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
		deletePolicy(testAppId)
		stopScheduler()
		stopScalingEngine()
	})

	Describe("Create Schedule", func() {
		Context("Valid specific date schedule", func() {

			AfterEach(func() {
				_, err := deleteSchedule(testAppId)
				Expect(err).NotTo(HaveOccurred())
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
