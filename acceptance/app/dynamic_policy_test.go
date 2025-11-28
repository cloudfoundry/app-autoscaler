package app_test

import (
	"acceptance"
	"acceptance/helpers"

	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudfoundry/cf-test-helpers/v2/generator"
	cfh "github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gexec"
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

	When("an ordinary service-binding is used", func() {
		JustBeforeEach(func() {
			appToScaleName = helpers.CreateTestApp(cfg, "dynamic-policy", initialInstanceCount)

			appToScaleGUID, err = helpers.GetAppGuid(cfg, appToScaleName)
			Expect(err).NotTo(HaveOccurred())
			helpers.StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
			instanceName = helpers.CreatePolicy(cfg, appToScaleName, appToScaleGUID, policy)
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
					policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "memoryused", int64(0.9*expectedAverageUsageAfterScaling), int64(0.9*heapToUse))
					initialInstanceCount = 1
				})

				It("should scale out and then back in.", func() {
					By(fmt.Sprintf("Use heap %d MB of heap on app", int64(heapToUse)))
					helpers.CurlAppInstance(cfg, appToScaleName, 0, fmt.Sprintf("/memory/%d/5", int64(heapToUse)))

					By("wait for scale to 2")
					helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

					By("Drop memory used by app")
					helpers.CurlAppInstance(cfg, appToScaleName, 0, "/memory/close")

					By("Wait for scale to minimum instances")
					helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
				})
			})
		})
		Context("when scaling by memoryutil", func() {

			Context("when memoryutil", func() {
				BeforeEach(func() {
					//current app resident size is 66mb so 66/128mb is 55%
					policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "memoryutil", 58, 63)
					initialInstanceCount = 1
				})

				It("should scale out and back in", func() {
					heapToUse := min(maxHeapLimitMb, int(float64(cfg.NodeMemoryLimit)*0.80))
					By(fmt.Sprintf("use 80%% or %d MB of memory in app", heapToUse))
					helpers.CurlAppInstance(cfg, appToScaleName, 0, fmt.Sprintf("/memory/%d/5", heapToUse))

					By("Wait for scale to 2 instances")
					helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

					By("drop memory used")
					helpers.CurlAppInstance(cfg, appToScaleName, 0, "/memory/close")

					By("Wait for scale down to 1 instance")
					helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
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
					policy = helpers.GenerateDynamicScaleOutPolicy(1, 2, "responsetime", 50)
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
					helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)
				})
			})

			Context("when responsetime is in range of scaling in threshold", func() {

				BeforeEach(func() {
					policy = helpers.GenerateDynamicScaleInPolicyBetween("responsetime", 50, 150)
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
					helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
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
					policy = helpers.GenerateDynamicScaleOutPolicy(1, 2, "throughput", 15)
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
					helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)
				})
			})

			Context("when throughput is in range of scaling in threshold", func() {

				BeforeEach(func() {
					policy = helpers.GenerateDynamicScaleInPolicyBetween("throughput", 5, 15)
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
					helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
				})
			})
		})

		// To check existing aggregated cpu metrics do: cf asm APP_NAME cpu
		Context("when scaling by cpu", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "cpu", int64(float64(cfg.CPUUpperThreshold)*0.2), int64(float64(cfg.CPUUpperThreshold)*0.4))
				initialInstanceCount = 1
			})

			It("when cpu is greater than scaling out threshold", func() {
				By("should scale out to 2 instances")
				helpers.StartCPUUsage(cfg, appToScaleName, int(float64(cfg.CPUUpperThreshold)*0.9), 5)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

				By("should scale in to 1 instance after cpu usage is reduced")
				//only hit the one instance that was asked to run hot.
				helpers.StopCPUUsage(cfg, appToScaleName, 0)

				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 10*time.Minute)
			})
		})
		Context("when there is a scaling policy for cpuutil", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "cpuutil", 40, 80)
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

				helpers.ScaleMemory(cfg, appToScaleName, cfg.CPUUtilScalingPolicyTest.AppMemory)

				// cpuutil will be 100% if cpu usage is reaching the value of cpu entitlement
				maxCPUUsage := cfg.CPUUtilScalingPolicyTest.AppCPUEntitlement
				helpers.StartCPUUsage(cfg, appToScaleName, maxCPUUsage, 5)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

				// only hit the one instance that was asked to run hot
				helpers.StopCPUUsage(cfg, appToScaleName, 0)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
			})
		})
		Context("when there is a scaling policy for diskutil", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "diskutil", 30, 60)
				initialInstanceCount = 1
			})

			It("should scale out and in", func() {
				helpers.ScaleDisk(cfg, appToScaleName, "1GB")

				helpers.StartDiskUsage(cfg, appToScaleName, 800, 5)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

				// only hit the one instance that was asked to occupy disk space
				helpers.StopDiskUsage(cfg, appToScaleName, 0)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
			})
		})
		Context("when there is a scaling policy for disk", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "disk", 300, 600)
				initialInstanceCount = 1
			})

			It("should scale out and in", func() {
				helpers.ScaleDisk(cfg, appToScaleName, "1GB")

				helpers.StartDiskUsage(cfg, appToScaleName, 800, 5)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 5*time.Minute)

				// only hit the one instance that was asked to occupy disk space
				helpers.StopDiskUsage(cfg, appToScaleName, 0)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 5*time.Minute)
			})
		})
	})
	When("a service-key is used", func() {
		When("providing a valid app-guid together with a policy", func() {
			var serviceInstanceName string
			var session *gexec.Session
			BeforeEach(func() {
				serviceInstanceName = helpers.CreateService(cfg)

				paramsTemplate := `
{
	"schema-version": "0.1",
	"configuration": {
		"app_guid": "%s"
	},
	"instance_min_count": 1,
	"instance_max_count": 2,
	"scaling_rules": [
		{
			"metric_type": "cpuutil",
			"threshold": 50,
			"operator": ">=",
			"adjustment": "+1"
		},
		{
			"metric_type": "cpuutil",
			"threshold": 30,
			"operator": "<",
			"adjustment": "-1"
		}
	]
}
`
				serviceInstanceName = generator.PrefixedRandomName(cfg.Prefix, cfg.InstancePrefix)
				serviceKeyName := fmt.Sprintf("%s@%s", appToScaleName, serviceInstanceName)
				params := fmt.Sprintf(paramsTemplate, appToScaleGUID)

				// Execution
				session = helpers.CreateServiceKeyWithParams(
					serviceInstanceName, serviceKeyName, params, cfg.DefaultTimeoutDuration())
			})
			AfterEach(func() {
				helpers.DeleteServiceInstance(cfg, serviceInstanceName)
			})

			It("succeeds and scales both, up and down", func() {
				// Validation
				By("Creating the service-key successfully")
				Expect(session).To(Exit(0))

				// Part-validation setup
				By("Starting CPU usage to trigger scale out")
				helpers.StartCPUUsage(cfg, appToScaleName, 60, 5)

				// Validation
				By("Waiting for scale out to 2 instances")
				totalTime := time.Duration(cfg.AggregateInterval*2)*time.Second + 3*time.Minute
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, totalTime)

				// Part-validation setup
				By("Stopping CPU usage to trigger scale in")
				helpers.StopCPUUsage(cfg, appToScaleName, 0)
				helpers.StopCPUUsage(cfg, appToScaleName, 1)

				// Validation
				By("Waiting for scale in to 1 instance")
				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, totalTime)
			})
		})
	})
})

// ================================================================================
// Helpers
// ================================================================================

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
