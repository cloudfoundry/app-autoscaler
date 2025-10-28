package app_test

import (
	"acceptance"
	. "acceptance/helpers"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	cfh "github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AutoScaler dynamic policy", func() {
	var (
		policy         string
		err            error
		doneChan       chan bool
		doneAcceptChan chan bool
		ticker         *time.Ticker
		maxHeapLimitMb int
	)

	const minimalMemoryUsage = 17 // observed minimal memory usage by the test app

	JustBeforeEach(func() {
		appToScaleName = CreateTestApp(cfg, "dynamic-policy", initialInstanceCount)

		appToScaleGUID, err = GetAppGuid(cfg, appToScaleName)
		Expect(err).NotTo(HaveOccurred())
		StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
		instanceName = CreatePolicy(cfg, appToScaleName, appToScaleGUID, policy)
	})
	BeforeEach(func() {
		maxHeapLimitMb = cfg.NodeMemoryLimit - minimalMemoryUsage
	})

	AfterEach(AppAfterEach)

	Context("when scaling by memoryused", func() {

		Context("There is a scale out and scale in policy", func() {
			var heapToUse float64
			BeforeEach(func() {
				heapToUse = float64(min(maxHeapLimitMb, 200))
				expectedAverageUsageAfterScaling := float64(heapToUse)/2 + minimalMemoryUsage
				policy = GenerateDynamicScaleOutAndInPolicy(1, 2, "memoryused", int64(0.9*expectedAverageUsageAfterScaling), int64(0.9*heapToUse))
				initialInstanceCount = 1
			})

			It("should scale out and then back in.", func() {
				By(fmt.Sprintf("Use heap %d MB of heap on app", int64(heapToUse)))
				CurlAppInstance(cfg, appToScaleName, 0, fmt.Sprintf("/memory/%d/5", int64(heapToUse)))

				By("wait for scale to 2")
				WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

				By("Drop memory used by app")
				CurlAppInstance(cfg, appToScaleName, 0, "/memory/close")

				By("Wait for scale to minimum instances")
				WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
			})
		})
	})

	Context("when scaling by memoryutil", func() {

		Context("when memoryutil", func() {
			BeforeEach(func() {
				//current app resident size is 66mb so 66/128mb is 55%
				policy = GenerateDynamicScaleOutAndInPolicy(1, 2, "memoryutil", 58, 63)
				initialInstanceCount = 1
			})

			It("should scale out and back in", func() {
				heapToUse := min(maxHeapLimitMb, int(float64(cfg.NodeMemoryLimit)*0.80))
				By(fmt.Sprintf("use 80%% or %d MB of memory in app", heapToUse))
				CurlAppInstance(cfg, appToScaleName, 0, fmt.Sprintf("/memory/%d/5", heapToUse))

				By("Wait for scale to 2 instances")
				WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

				By("drop memory used")
				CurlAppInstance(cfg, appToScaleName, 0, "/memory/close")

				By("Wait for scale down to 1 instance")
				WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
			})
		})
	})

	Context("when scaling by responsetime", func() {

		JustBeforeEach(func() {
			doneChan = make(chan bool)
			doneAcceptChan = make(chan bool)
		})

		AfterEach(func() {
			close(doneChan)
			Eventually(doneAcceptChan, 10*time.Second).Should(Receive())
		})

		Context("when responsetime is greater than scaling out threshold", func() {

			BeforeEach(func() {
				policy = GenerateDynamicScaleOutPolicy(1, 2, "responsetime", 50)
				initialInstanceCount = 1
			})

			JustBeforeEach(func() {
				appUri := cfh.AppUri(appToScaleName, "/responsetime/slow/100", cfg)
				ticker = time.NewTicker(1 * time.Second)
				rps := 5
				go func(chan bool) {
					defer GinkgoRecover()
					for {
						select {
						case <-doneChan:
							ticker.Stop()
							doneAcceptChan <- true
							return
						case <-ticker.C:
							concurrentHttpGet(rps, appUri)
						}
					}
				}(doneChan)
			})

			It("should scale out", Label(acceptance.LabelSmokeTests), func() {
				WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)
			})
		})

		Context("when responsetime is in range of scaling in threshold", func() {

			BeforeEach(func() {
				policy = GenerateDynamicScaleInPolicyBetween("responsetime", 50, 150)
				initialInstanceCount = 2
			})

			JustBeforeEach(func() {
				appUri := cfh.AppUri(appToScaleName, "/responsetime/slow/100", cfg)
				ticker = time.NewTicker(1 * time.Second)
				rps := 5
				go func(chan bool) {
					defer GinkgoRecover()
					for {
						select {
						case <-doneChan:
							ticker.Stop()
							doneAcceptChan <- true
							return
						case <-ticker.C:
							concurrentHttpGet(rps, appUri)
						}
					}
				}(doneChan)
			})

			It("should scale in", func() {
				WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
			})
		})

	})

	Context("when scaling by throughput", func() {
		JustBeforeEach(func() {
			doneChan = make(chan bool)
			doneAcceptChan = make(chan bool)
		})

		AfterEach(func() {
			close(doneChan)
			Eventually(doneAcceptChan, 10*time.Second).Should(Receive())
		})

		Context("when throughput is greater than scaling out threshold", func() {

			BeforeEach(func() {
				policy = GenerateDynamicScaleOutPolicy(1, 2, "throughput", 15)
				initialInstanceCount = 1
			})

			JustBeforeEach(func() {
				appUri := cfh.AppUri(appToScaleName, "/responsetime/fast", cfg)
				ticker = time.NewTicker(1 * time.Second)
				rps := 20
				go func(chan bool) {
					defer GinkgoRecover()
					for {
						select {
						case <-doneChan:
							ticker.Stop()
							doneAcceptChan <- true
							return
						case <-ticker.C:
							concurrentHttpGet(rps, appUri)
						}
					}
				}(doneChan)
			})

			It("should scale out", func() {
				WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)
			})
		})

		Context("when throughput is in range of scaling in threshold", func() {

			BeforeEach(func() {
				policy = GenerateDynamicScaleInPolicyBetween("throughput", 5, 15)
				initialInstanceCount = 2
			})

			JustBeforeEach(func() {
				appUri := cfh.AppUri(appToScaleName, "/responsetime/fast", cfg)
				ticker = time.NewTicker(1 * time.Second)
				rps := 20
				go func(chan bool) {
					defer GinkgoRecover()
					for {
						select {
						case <-doneChan:
							ticker.Stop()
							doneAcceptChan <- true
							return
						case <-ticker.C:
							concurrentHttpGet(rps, appUri)
						}
					}
				}(doneChan)
			})

			It("should scale in", func() {
				// because we are generating 20rps but starting with 2 instances,
				// there should be on average 10rps per instance, which should trigger the scale in
				WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
			})
		})
	})

	// To check existing aggregated cpu metrics do: cf asm APP_NAME cpu
	Context("when scaling by cpu", func() {

		BeforeEach(func() {
			policy = GenerateDynamicScaleOutAndInPolicy(1, 2, "cpu", int64(float64(cfg.CPUUpperThreshold)*0.2), int64(float64(cfg.CPUUpperThreshold)*0.4))
			initialInstanceCount = 1
		})

		It("when cpu is greater than scaling out threshold", func() {
			By("should scale out to 2 instances")
			StartCPUUsage(cfg, appToScaleName, int(float64(cfg.CPUUpperThreshold)*0.9), 5)
			WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

			By("should scale in to 1 instance after cpu usage is reduced")
			//only hit the one instance that was asked to run hot.
			StopCPUUsage(cfg, appToScaleName, 0)

			WaitForNInstancesRunning(appToScaleGUID, 1, 10*time.Minute)
		})
	})

	Context("when there is a scaling policy for cpuutil", func() {
		BeforeEach(func() {
			policy = GenerateDynamicScaleOutAndInPolicy(1, 2, "cpuutil", 40, 80)
			initialInstanceCount = 1
		})

		It("should scale out and in", func() {
			// this test depends on
			//   - Diego cell size (CPU and RAM)
			//   - CPU entitlements per share configured in ci/operations/set-cpu-entitlement-per-share.yaml
			//   - app memory configured via cfg.CPUUtilScalingPolicyTest.AppMemory
			//   - app CPU entitlement configured via cfg.CPUUtilScalingPolicyTest.AppCPUEntitlement
			//
			// the following gives an example how to calculate an app CPU entitlement:
			//   - Diego cell size = 8 CPU 32GB RAM
			//   - total shares = 1024 * 32[GB host ram] / 8[upper limit of app memory in GB] = 4096
			//   - CPU entitlement per share = 8[number host CPUs] * 100/ 4096[total shares] = 0,1953%
			//   - app memory = 1GB
			//   - app CPU entitlement = 4096[total shares] / (32[GB host ram] * 1024) * (1[app memory in GB] * 1024) * 0,1953 ~= 25%

			ScaleMemory(cfg, appToScaleName, cfg.CPUUtilScalingPolicyTest.AppMemory)

			// cpuutil will be 100% if cpu usage is reaching the value of cpu entitlement
			maxCPUUsage := cfg.CPUUtilScalingPolicyTest.AppCPUEntitlement
			StartCPUUsage(cfg, appToScaleName, maxCPUUsage, 5)
			WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

			// only hit the one instance that was asked to run hot
			StopCPUUsage(cfg, appToScaleName, 0)
			WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
		})
	})

	Context("when there is a scaling policy for diskutil", func() {
		BeforeEach(func() {
			policy = GenerateDynamicScaleOutAndInPolicy(1, 2, "diskutil", 30, 60)
			initialInstanceCount = 1
		})

		It("should scale out and in", func() {
			ScaleDisk(cfg, appToScaleName, "1GB")

			StartDiskUsage(cfg, appToScaleName, 800, 5)
			WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

			// only hit the one instance that was asked to occupy disk space
			StopDiskUsage(cfg, appToScaleName, 0)
			WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
		})
	})

	Context("when there is a scaling policy for disk", func() {
		BeforeEach(func() {
			policy = GenerateDynamicScaleOutAndInPolicy(1, 2, "disk", 300, 600)
			initialInstanceCount = 1
		})

		It("should scale out and in", func() {
			ScaleDisk(cfg, appToScaleName, "1GB")

			StartDiskUsage(cfg, appToScaleName, 800, 5)
			WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

			// only hit the one instance that was asked to occupy disk space
			StopDiskUsage(cfg, appToScaleName, 0)
			WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
		})
	})
})

func concurrentHttpGet(count int, url string) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:all
			},
		},
	}

	for i := 0; i < count; i++ {
		go func() {
			GinkgoWriter.Printf("[http] [get] [request] url: %s\n", url)

			resp, err := client.Get(url)

			if err != nil {
				GinkgoWriter.Printf("[http] [get] [response] error: %s\n", err.Error())
			}

			if resp != nil {
				GinkgoWriter.Printf("[http] [get] [response] status-code: %d\n", resp.StatusCode)
				err = resp.Body.Close()
				if err != nil {
					GinkgoWriter.Printf("[http] [get] [response] error closing response body: %s\n", err.Error())
				}
			}
		}()
	}
}
