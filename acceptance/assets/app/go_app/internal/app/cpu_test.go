package app_test

import (
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app"
	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app/appfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CPU tests", func() {

	Context("CPU handler", func() {
		fakeCPUWaster := &appfakes.FakeCPUWaster{}

		It("should err if utilization not an int64", func() {
			apiTest(nil, nil, fakeCPUWaster, nil).
				Get("/cpu/invalid/4").
				Expect(GinkgoT()).
				Status(http.StatusBadRequest).
				Body(`{"error":{"description":"invalid utilization: strconv.ParseInt: parsing \"invalid\": invalid syntax"}}`).
				End()
		})
		It("should err if cpu out of bounds", func() {
			apiTest(nil, nil, fakeCPUWaster, nil).
				Get("/cpu/100001010101010249032897287298719874687936483275648273632429479827398798271/4").
				Expect(GinkgoT()).
				Status(http.StatusBadRequest).
				Body(`{"error":{"description":"invalid utilization: strconv.ParseInt: parsing \"100001010101010249032897287298719874687936483275648273632429479827398798271\": value out of range"}}`).
				End()
		})
		It("should err if cpu not an int", func() {
			apiTest(nil, nil, fakeCPUWaster, nil).
				Get("/cpu/5/invalid").
				Expect(GinkgoT()).
				Status(http.StatusBadRequest).
				Body(`{"error":{"description":"invalid minutes: strconv.ParseInt: parsing \"invalid\": invalid syntax"}}`).
				End()
		})
		It("should return ok and sleep correctDuration", func() {
			apiTest(nil, nil, fakeCPUWaster, nil).
				Get("/cpu/5/4").
				Expect(GinkgoT()).
				Status(http.StatusOK).
				Body(`{"utilization":5, "minutes":4 }`).
				End()

			Eventually(func() int { return fakeCPUWaster.UseCPUCallCount() }).Should(Equal(1))
			utilization, duration := fakeCPUWaster.UseCPUArgsForCall(0)
			Expect(duration).Should(Equal(4 * time.Minute))
			Expect(utilization).Should(Equal(int64(5)))
		})
	})

	// This test is timing sensitive and may fail on GitHub workers, which is why it is marked as flaky
	Context("ConcurrentBusyLoopCPUWaster", func() {
		Context("UseCPU", FlakeAttempts(3), func() {
			DescribeTable("should use cpu",
				func(utilisation int64, duration time.Duration) {
					oldCpu := getTotalCPUUsage("before test")

					By("wasting cpu time")
					cpuWaster := &app.ConcurrentBusyLoopCPUWaster{}

					cpuWaster.UseCPU(utilisation, duration)

					Expect(cpuWaster.IsRunning()).To(Equal(true))
					Eventually(cpuWaster.IsRunning).WithTimeout(duration + time.Second).WithPolling(time.Second).Should(Equal(false))

					newCpu := getTotalCPUUsage("after test")

					expectedCPUUsage := multiplyDurationByPercentage(duration, utilisation)

					// If the environment variable CI is not set to true: Give 10% tolerance - but at least 1 second, as this is the internal resolution of the CPU waster.
					// If the environment variable CI is set to true, as is by default in GitHub Actions
					// (see https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables)
					// give 50% tolerance. This is due to the fact that on CI workers the available CPU time is not guaranteed.
					var tolerance time.Duration
					if ci := os.Getenv("CI"); ci == "true" {
						tolerance = max(multiplyDurationByPercentage(expectedCPUUsage, 50), time.Second)
					} else {
						tolerance = max(multiplyDurationByPercentage(expectedCPUUsage, 10), time.Second)
					}
					Expect(newCpu - oldCpu).To(BeNumerically("~", expectedCPUUsage, tolerance))
				},
				Entry("25% for 10 seconds", int64(25), time.Second*10),
				Entry("50% for 10 seconds", int64(50), time.Second*10),
				Entry("100% for 10 seconds", int64(100), time.Second*10),
				Entry("200% for 10 seconds", int64(200), time.Second*10),
				Entry("400% for 10 seconds", int64(400), time.Second*10),
			)
		})
	})
})

func getTotalCPUUsage(action string) time.Duration {
	GinkgoHelper()

	cpuTotalUsage := app.GetClock()
	cpuTotalDuration := time.Duration(float64(time.Second) * cpuTotalUsage / app.ClocksPerSec)

	GinkgoWriter.Printf("total cpu time %s: %s\n", action, cpuTotalDuration.String())

	return cpuTotalDuration
}

func multiplyDurationByPercentage(duration time.Duration, percentage int64) time.Duration {
	return time.Duration(float64(duration) * float64(percentage) / 100)
}
