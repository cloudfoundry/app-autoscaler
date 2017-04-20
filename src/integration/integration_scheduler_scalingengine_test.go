package integration_test

import (
	"autoscaler/cf"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "integration"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var _ = Describe("Integration_Scheduler_ScalingEngine", func() {
	var (
		testAppId         string
		initInstanceCount int = 2
		policyStr         string
	)

	BeforeEach(func() {
		initializeHttpClient("scheduler.crt", "scheduler.key", "autoscaler-ca.crt", schedulerScalingEngineHttpRequestTimeout)

		testAppId = getRandomId()

		startFakeCCNOAAUAA(initInstanceCount)

		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, tmpDir, consulRunner.ConsulCluster())
		startScalingEngine()

		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir, strings.Split(consulRunner.Address(), ":")[1])
		schedulerProcess = startScheduler()

		policyByte := readPolicyFromFile("fakePolicyWithSpecificDateSchedule.json")
		policyStr = setPolicyDateTime(policyByte)

	})

	AfterEach(func() {
		deleteSchedule(testAppId)
		stopScheduler(schedulerProcess)
		stopScalingEngine()
	})

	Describe("Create Schedule", func() {
		Context("Valid specific date schedule", func() {

			It("creates active schedule in scaling engine", func() {
				resp, err := createSchedule(testAppId, policyStr)
				checkResponseIsEmpty(resp, err, http.StatusOK)

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 1*time.Second).Should(BeTrue())

			})
		})

		Context("ScalingEngine Server is down", func() {
			BeforeEach(func() {
				stopScalingEngine()
			})

			It("should create an active schedule in scaling engine after restart", func() {
				resp, err := createSchedule(testAppId, policyStr)
				checkResponseIsEmpty(resp, err, http.StatusOK)

				Consistently(func() error {
					_, err := getActiveSchedule(testAppId)
					return err
				}, 2*time.Minute, 1*time.Second).Should(HaveOccurred())

				startScalingEngine()
				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 20*time.Second, 1*time.Second).Should(BeTrue())
			})
		})

	})

	Describe("Delete Schedule", func() {
		BeforeEach(func() {
			resp, err := createSchedule(testAppId, policyStr)
			checkResponseIsEmpty(resp, err, http.StatusOK)

			Eventually(func() bool {
				return activeScheduleExists(testAppId)
			}, 2*time.Minute, 1*time.Second).Should(BeTrue())
		})

		It("deletes active schedule in scaling engine", func() {
			resp, err := deleteSchedule(testAppId)
			checkResponseIsEmpty(resp, err, http.StatusNoContent)

			Eventually(func() bool {
				return activeScheduleExists(testAppId)
			}, 2*time.Minute, 1*time.Second).Should(BeFalse())
		})
	})

})

func setPolicyDateTime(policyByte []byte) string {
	timeZone := "GMT"
	location, _ := time.LoadLocation(timeZone)
	timeNowInTimeZone := time.Now().In(location)
	dateTimeFormat := "2006-01-02T15:04"
	startTime := timeNowInTimeZone.Add(70 * time.Second).Format(dateTimeFormat)

	return fmt.Sprintf(string(policyByte), timeZone, startTime, timeNowInTimeZone.Add(2*time.Hour).Format(dateTimeFormat))
}

func checkResponseIsEmpty(resp *http.Response, err error, expectedStatus int) {
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	Expect(body).To(HaveLen(0))
	Expect(resp.StatusCode).To(Equal(expectedStatus))
}

func activeScheduleExists(appId string) bool {
	resp, err := getActiveSchedule(appId)
	Expect(err).NotTo(HaveOccurred())

	return resp.StatusCode == http.StatusOK
}
