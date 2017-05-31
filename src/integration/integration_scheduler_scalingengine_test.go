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
		testGuid          string
		anotherGuid       string
		initInstanceCount int = 2
		policyStr         string
	)

	BeforeEach(func() {
		initializeHttpClient("scheduler.crt", "scheduler.key", "autoscaler-ca.crt", schedulerScalingEngineHttpRequestTimeout)

		testAppId = getRandomId()
		testGuid = getRandomId()
		anotherGuid = getRandomId()
		startFakeCCNOAAUAA(initInstanceCount)

		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, tmpDir, consulRunner.ConsulCluster())
		startScalingEngine()

		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir, strings.Split(consulRunner.Address(), ":")[1])
		schedulerProcess = startScheduler()

		policyStr = setPolicyDateTime(readPolicyFromFile("fakePolicyWithSpecificDateSchedule.json"))

	})

	AfterEach(func() {
		deleteSchedule(testAppId)
		deletePolicy(testAppId)
		stopScheduler(schedulerProcess)
		stopScalingEngine()
	})

	Describe("Create Schedule", func() {
		Context("Valid specific date schedule", func() {

			It("creates active schedule in scaling engine", func() {
				resp, err := createSchedule(testAppId, testGuid, policyStr)
				checkResponseEmptyAndStatusCode(resp, err, http.StatusOK)

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
				resp, err := createSchedule(testAppId, testGuid, policyStr)
				checkResponseEmptyAndStatusCode(resp, err, http.StatusOK)

				Consistently(func() error {
					_, err := getActiveSchedule(testAppId)
					return err
				}, 2*time.Minute, 1*time.Second).Should(HaveOccurred())

				startScalingEngine()
				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 30*time.Second, 1*time.Second).Should(BeTrue())
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

	Describe("Synchronized Schedule", func() {
		Context("when the app's policy has been updated ", func() {
			BeforeEach(func() {
				resp, err := createSchedule(testAppId, testGuid, policyStr)
				checkResponseEmptyAndStatusCode(resp, err, http.StatusOK)

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 5*time.Second).Should(BeTrue())

				insertPolicy(testAppId, policyStr, anotherGuid)

			})
			It("updates the schedules", func() {
				resp, err := synchronizeSchedule()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 5*time.Second).Should(BeTrue())

			})
		})

		Context("when the app's policy has been updated with the same guid ", func() {
			BeforeEach(func() {
				resp, err := createSchedule(testAppId, testGuid, policyStr)
				checkResponseEmptyAndStatusCode(resp, err, http.StatusOK)

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 5*time.Second).Should(BeTrue())

				insertPolicy(testAppId, policyStr, testGuid)

			})
			It("not update or create any schedule", func() {
				resp, err := synchronizeSchedule()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				Consistently(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 5*time.Second).Should(BeTrue())

			})
		})

		Context("when the app's policy has been deleted", func() {
			BeforeEach(func() {
				resp, err := createSchedule(testAppId, testGuid, policyStr)
				checkResponseEmptyAndStatusCode(resp, err, http.StatusOK)

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 5*time.Second).Should(BeTrue())

				deletePolicy(testAppId)

			})
			It("delete the schedules", func() {
				resp, err := synchronizeSchedule()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 5*time.Second).Should(BeFalse())

			})
		})

		Context("when the app's policy has been created", func() {
			BeforeEach(func() {
				insertPolicy(testAppId, policyStr, anotherGuid)

				_, err := deleteSchedule(testAppId)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 1*time.Second).Should(BeFalse())

			})
			It("create the schedules", func() {
				resp, err := synchronizeSchedule()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 5*time.Second).Should(BeTrue())

			})
		})

		Context("when there is no policy and schedule", func() {
			BeforeEach(func() {
				_, err := deleteSchedule(testAppId)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 1*time.Second).Should(BeFalse())

				deletePolicy(testAppId)

			})
			It("do nothing", func() {
				resp, err := synchronizeSchedule()
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				Consistently(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute, 5*time.Second).Should(BeFalse())

			})
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

func checkResponseEmptyAndStatusCode(resp *http.Response, err error, expectedStatus int) {
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
