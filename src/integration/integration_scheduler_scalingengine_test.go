package integration_test

import (
	"autoscaler/cf"
	"code.cloudfoundry.org/cfhttp"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
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
		schedulerProcess  ifrit.Process
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
		schedulerConfPath = components.PrepareSchedulerConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), tmpDir)
		schedulerProcess = startScheduler()
		startScalingEngine()

		policyByte := readPolicyFromFile("fakePolicyWithSpecificDateSchedule.json")
		policyStr = setPolicyDateTime(policyByte)

	})

	AfterEach(func() {
		deleteSchedules(testAppId)
		stopAll()
		stopScheduler(schedulerProcess)
	})

	Describe("Create Schedule", func() {
		Context("Valid specific date schedule", func() {

			It("creates active schedule in scaling engine", func() {
				Expect(getNumberOfActiveSchedules(testAppId)).To(Equal(0))
				resp, err := createSchedule(testAppId, policyStr)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal(""))
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Eventually(getAppIdOfActiveSchedules, 5*time.Minute).Should(ContainElement(testAppId))
				Expect(getNumberOfActiveSchedules(testAppId)).To(Equal(1))

			})
		})

		Context("ScalingEngine Server is down", func() {
			BeforeEach(func() {
				stopAll()
			})

			It("should return 500", func() {
				resp, err := createSchedule(testAppId, policyStr)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal(""))
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Eventually(stdOutbufferMap[Scheduler], 5*time.Minute).Should(gbytes.Say("Error connecting to scaling engine, failed with error: .* for app id: " + testAppId))
				Expect(getNumberOfActiveSchedules(testAppId)).To(Equal(0))
			})
		})

	})

	Describe("Delete Schedule", func() {
		BeforeEach(func() {
			resp, err := createSchedule(testAppId, policyStr)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal(""))
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Eventually(getAppIdOfActiveSchedules, 5*time.Minute).Should(ContainElement(testAppId))
		})

		It("deletes active schedule in scaling engine", func() {
			Expect(getNumberOfActiveSchedules(testAppId)).To(Equal(1))
			resp, err := deleteSchedule(testAppId)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal(""))
			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
			Eventually(getAppIdOfActiveSchedules, 5*time.Minute).ShouldNot(ContainElement(testAppId))
			Expect(getNumberOfActiveSchedules(testAppId)).To(Equal(0))
		})

		Context("ScalingEngine Server is down", func() {
			BeforeEach(func() {
				stopAll()
			})

			It("should return 500", func() {
				Expect(getNumberOfActiveSchedules(testAppId)).To(Equal(1))
				resp, err := deleteSchedule(testAppId)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				body, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal(""))
				Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
				Eventually(stdOutbufferMap[Scheduler], 5*time.Minute).Should(gbytes.Say("Error connecting to scaling engine, failed with error: .* for app id: " + testAppId))
				Expect(getNumberOfActiveSchedules(testAppId)).To(Equal(1))
			})
		})

	})

})

func deleteSchedules(appId string) {
	resp, err := deleteSchedule(appId)
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()
}

func setPolicyDateTime(policyByte []byte) string {

	timeZone := "GMT"
	location, _ := time.LoadLocation(timeZone)
	timeNowInTimeZone := time.Now().In(location)
	dateTimeFormat := "2006-01-02T15:04"
	startTime := timeNowInTimeZone.Add(70 * time.Second).Format(dateTimeFormat)

	return fmt.Sprintf(string(policyByte), timeZone, startTime, timeNowInTimeZone.Add(2*time.Hour).Format(dateTimeFormat))
}
