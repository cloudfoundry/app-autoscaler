package integration_test

import (
	"autoscaler/cf"
	"code.cloudfoundry.org/cfhttp"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "integration"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"time"
)

var _ = Describe("Integration_Scheduler_ScalingEngine", func() {
	var (
		testAppId         string
		initInstanceCount int = 2
		policyStr         string
	)

	BeforeEach(func() {
		schedulerTLSConfig, err := cfhttp.NewTLSConfig(
			filepath.Join(testCertDir, "scheduler.crt"),
			filepath.Join(testCertDir, "scheduler.key"),
			filepath.Join(testCertDir, "autoscaler-ca.crt"),
		)
		Expect(err).NotTo(HaveOccurred())
		httpClient.Transport.(*http.Transport).TLSClientConfig = schedulerTLSConfig
		httpClient.Timeout = schedulerScalingEngineHttpRequestTimeout
		testAppId = getRandomId()

		startFakeCCNOAAUAA(initInstanceCount)

		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, tmpDir)
		startScalingEngine()

		policyByte := readPolicyFromFile("fakePolicyWithSpecificDateSchedule.json")
		policyStr = setPolicyDateTime(policyByte)

	})

	AfterEach(func() {
		deleteSchedule(testAppId)
		stopScalingEngine()
	})

	Describe("Create Schedule", func() {
		Context("Valid specific date schedule", func() {

			It("creates active schedule in scaling engine", func() {
				resp, err := createSchedule(testAppId, policyStr)
				checkResponse(resp, err, http.StatusOK)

				Eventually(func() bool {
					return activeScheduleExists(testAppId)
				}, 2*time.Minute).Should(BeTrue())

			})
		})

		Context("ScalingEngine Server is down", func() {
			BeforeEach(func() {
				stopScalingEngine()
			})

			It("should not create an active schedule in scaling engine", func() {
				resp, err := createSchedule(testAppId, policyStr)
				checkResponse(resp, err, http.StatusOK)

				Consistently(func() int {
					return getNumberOfActiveSchedules(testAppId)
				}, 2*time.Minute).Should(BeZero())
			})
		})

	})

	Describe("Delete Schedule", func() {
		BeforeEach(func() {
			resp, err := createSchedule(testAppId, policyStr)
			checkResponse(resp, err, http.StatusOK)

			Eventually(func() bool {
				return activeScheduleExists(testAppId)
			}, 2*time.Minute).Should(BeTrue())
		})

		It("deletes active schedule in scaling engine", func() {
			resp, err := deleteSchedule(testAppId)
			checkResponse(resp, err, http.StatusNoContent)

			Eventually(func() bool {
				return activeScheduleExists(testAppId)
			}, 2*time.Minute).Should(BeFalse())
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

func checkResponse(resp *http.Response, err error, expectedStatus int) {
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
