package app_test

import (
	"acceptance"
	"acceptance/helpers"
	"os"

	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	cfh "github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("AutoScaler dynamic policy", func() {
	var (
		policy             string
		err                error
		doneChan           chan bool
		doneAcceptChan     chan bool
		ticker             *time.Ticker
		maxHeapLimitMb     int
		memoryUtilScaleOut int64
		reportedMiB        float64
	)

	const (
		// cgroup v2 (Noble stemcell) counts kernel memory against the container limit,
		// unlike cgroup v1 which tracked it separately. This requires more headroom.
		minimalMemoryUsage = 35

		// How long the test app holds resource usage before releasing (minutes).
		// Needs: aggregate_interval (120s) + breach_duration (60s) + metric lag buffer = ~4 min.
		holdMinutes = 4

		// responsetime test: app sleeps 100ms per request, threshold at 50ms triggers scale-out.
		responseTimeSlowDelayMs   = 100
		responseTimeScaleOutMs    = 50
		responseTimeScaleInLowMs  = 50
		responseTimeScaleInHighMs = 150

		// throughput test: generate 20 rps, scale-out at 15 rps per instance.
		throughputRps                 = 20
		throughputScaleOutPerInstance = 15
		throughputScaleInLow          = 5
		throughputScaleInHigh         = 15

		// disk tests: the test app writes N*1000*1000 bytes (decimal MB) but CF
		// reports disk in binary MiB (bytes÷1024²). 550 decimal MB ≈ 524 MiB.
		diskUsageMb      = 550
		diskScaleInMb    = 300
		diskScaleOutMb   = 500
		diskUtilScaleIn  = 30
		diskUtilScaleOut = 45

		// memoryutil scale-in threshold (% quota); must stay below post-scale-out avg utilisation.
		memoryUtilScaleIn        = 30
		heapFractionOfLimit      = 0.80
		decimalMBToMiB           = 1_000_000.0 / 1_048_576.0 // cgroup v2 reports binary MiB, Go allocates decimal MB
		memoryUtilSafetyFactor   = 0.90
		memoryUsedScaleOutFactor = 0.85
		baselineMemoryMiB        = 10
	)
	BeforeEach(func() {
		maxHeapLimitMb = int(float64(cfg.NodeMemoryLimit)*heapFractionOfLimit) - minimalMemoryUsage
		reportedMiB = float64(maxHeapLimitMb) * decimalMBToMiB
		expectedUtilPct := reportedMiB / float64(cfg.NodeMemoryLimit) * 100
		memoryUtilScaleOut = int64(expectedUtilPct * memoryUtilSafetyFactor)
	})
	When("an ordinary service-binding is used", func() {
		JustBeforeEach(func() {
			appToScaleName = helpers.CreateTestAppFromDroplet(cfg, dropletPath, "dynamic-policy", initialInstanceCount)

			appToScaleGUID, err = helpers.GetAppGuid(cfg, appToScaleName)
			Expect(err).NotTo(HaveOccurred())
			helpers.StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
			instanceName = helpers.CreatePolicy(cfg, appToScaleName, appToScaleGUID, policy)
		})

		AfterEach(func() {
			if os.Getenv("SKIP_TEARDOWN") == "true" {
				fmt.Println("Skipping Teardown...")
			} else {
				AppAfterEach()
			}
		})
		Context("when scaling by memoryused", func() {

			Context("There is a scale out and scale in policy", func() {
				BeforeEach(func() {
					// Scale-in: between baseline and post-scale avg; scale-out: below single-instance reportedMiB.
					scaleInThreshold := int64(reportedMiB/4) + baselineMemoryMiB
					scaleOutThreshold := int64(reportedMiB * memoryUsedScaleOutFactor)
					policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "memoryused",
						scaleInThreshold,
						scaleOutThreshold)
					initialInstanceCount = 1
				})

				It("should scale out and then back in.", func() {
					By(fmt.Sprintf("Use heap %d MB of heap on app", maxHeapLimitMb))
					helpers.CurlAppInstance(cfg, appToScaleName, 0, fmt.Sprintf("/memory/%d/%d", maxHeapLimitMb, holdMinutes))

					By("wait for scale to 2")
					helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 8*time.Minute)

					By("Drop memory used by app")
					helpers.CurlAppInstance(cfg, appToScaleName, 0, "/memory/close")

					By("Wait for scale to minimum instances")
					helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 8*time.Minute)
				})
			})
		})
		Context("when scaling by memoryutil", func() {

			Context("when memoryutil", func() {
				BeforeEach(func() {
					policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "memoryutil", memoryUtilScaleIn, memoryUtilScaleOut)
					initialInstanceCount = 1
				})

				It("should scale out and back in", func() {
					By(fmt.Sprintf("use %d MB of memory in app", maxHeapLimitMb))
					helpers.CurlAppInstance(cfg, appToScaleName, 0, fmt.Sprintf("/memory/%d/%d", maxHeapLimitMb, holdMinutes))

					By("Wait for scale to 2 instances")
					helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 8*time.Minute)

					By("drop memory used")
					helpers.CurlAppInstance(cfg, appToScaleName, 0, "/memory/close")

					By("Wait for scale down to 1 instance")
					helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 8*time.Minute)
				})
			})
		})
		Context("when scaling by responsetime", func() {
			JustBeforeEach(func() {
				doneChan = make(chan bool)
				doneAcceptChan = make(chan bool)
			})

			AfterEach(func() {
				if os.Getenv("SKIP_TEARDOWN") == "true" {
					fmt.Println("Skipping Teardown...")
				} else {
					close(doneChan)
					Eventually(doneAcceptChan, 10*time.Second).Should(Receive())
				}
			})

			Context("when responsetime is greater than scaling out threshold", func() {

				BeforeEach(func() {
					policy = helpers.GenerateDynamicScaleOutPolicy(1, 2, "responsetime", responseTimeScaleOutMs)
					initialInstanceCount = 1
				})

				JustBeforeEach(func() {
					appUri := cfh.AppUri(appToScaleName, fmt.Sprintf("/responsetime/slow/%d", responseTimeSlowDelayMs), cfg)
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
					helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 4*time.Minute)
				})
			})

			Context("when responsetime is in range of scaling in threshold", func() {

				BeforeEach(func() {
					policy = helpers.GenerateDynamicScaleInPolicyBetween("responsetime", responseTimeScaleInLowMs, responseTimeScaleInHighMs)
					initialInstanceCount = 2
				})

				JustBeforeEach(func() {
					appUri := cfh.AppUri(appToScaleName, fmt.Sprintf("/responsetime/slow/%d", responseTimeSlowDelayMs), cfg)
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
					helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 4*time.Minute)
				})
			})

		})
		Context("when scaling by throughput", func() {
			JustBeforeEach(func() {
				doneChan = make(chan bool)
				doneAcceptChan = make(chan bool)
			})

			AfterEach(func() {
				if os.Getenv("SKIP_TEARDOWN") == "true" {
					fmt.Println("Skipping Teardown...")
				} else {
					close(doneChan)
					Eventually(doneAcceptChan, 10*time.Second).Should(Receive())
				}
			})

			Context("when throughput is greater than scaling out threshold", func() {

				BeforeEach(func() {
					policy = helpers.GenerateDynamicScaleOutPolicy(1, 2, "throughput", throughputScaleOutPerInstance)
					initialInstanceCount = 1
				})

				JustBeforeEach(func() {
					appUri := cfh.AppUri(appToScaleName, "/responsetime/fast", cfg)
					ticker = time.NewTicker(1 * time.Second)
					rps := throughputRps
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
					helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 4*time.Minute)
				})
			})

			Context("when throughput is in range of scaling in threshold", func() {

				BeforeEach(func() {
					policy = helpers.GenerateDynamicScaleInPolicyBetween("throughput", throughputScaleInLow, throughputScaleInHigh)
					initialInstanceCount = 2
				})

				JustBeforeEach(func() {
					appUri := cfh.AppUri(appToScaleName, "/responsetime/fast", cfg)
					ticker = time.NewTicker(1 * time.Second)
					rps := throughputRps
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
					helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 4*time.Minute)
				})
			})
		})

		// To check existing aggregated cpu metrics do: cf asm APP_NAME cpu
		Context("when scaling by cpu", func() {
			BeforeEach(func() {
				scaleInThreshold := int64(float64(cfg.CPUUpperThreshold) * 0.2)
				scaleOutThreshold := int64(float64(cfg.CPUUpperThreshold) * 0.4)
				policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "cpu", scaleInThreshold, scaleOutThreshold)
				initialInstanceCount = 1
			})

			It("when cpu is greater than scaling out threshold", func() {
				By("should scale out to 2 instances")
				cpuUsage := int(float64(cfg.CPUUpperThreshold) * 0.9)
				helpers.StartCPUUsage(cfg, appToScaleName, cpuUsage, holdMinutes)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 8*time.Minute)

				By("should scale in to 1 instance after cpu usage is reduced")
				//only hit the one instance that was asked to run hot.
				helpers.StopCPUUsage(cfg, appToScaleName, 0)

				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 8*time.Minute)
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
				helpers.StartCPUUsage(cfg, appToScaleName, maxCPUUsage, holdMinutes)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 8*time.Minute)

				// only hit the one instance that was asked to run hot
				helpers.StopCPUUsage(cfg, appToScaleName, 0)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 8*time.Minute)
			})
		})
		Context("when there is a scaling policy for diskutil", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "diskutil", diskUtilScaleIn, diskUtilScaleOut)
				initialInstanceCount = 1
			})

			It("should scale out and in", func() {
				helpers.ScaleDisk(cfg, appToScaleName, "1GB")

				helpers.StartDiskUsage(cfg, appToScaleName, diskUsageMb, holdMinutes)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 8*time.Minute)

				// only hit the one instance that was asked to occupy disk space
				helpers.StopDiskUsage(cfg, appToScaleName, 0)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 8*time.Minute)
			})
		})
		Context("when there is a scaling policy for disk", func() {
			BeforeEach(func() {
				policy = helpers.GenerateDynamicScaleOutAndInPolicy(1, 2, "disk", diskScaleInMb, diskScaleOutMb)
				initialInstanceCount = 1
			})

			It("should scale out and in", func() {
				helpers.ScaleDisk(cfg, appToScaleName, "1GB")

				helpers.StartDiskUsage(cfg, appToScaleName, diskUsageMb, holdMinutes)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 8*time.Minute)

				// only hit the one instance that was asked to occupy disk space
				helpers.StopDiskUsage(cfg, appToScaleName, 0)
				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, 8*time.Minute)
			})
		})
	})
	When("a service-key is used", func() {
		BeforeEach(func() {
			initialInstanceCount = 1

			appToScaleName = helpers.CreateTestAppFromDroplet(cfg, dropletPath, "dyn_policy_with_sk", initialInstanceCount)
			appToScaleGUID, err = helpers.GetAppGuid(cfg, appToScaleName)
			Expect(err).NotTo(HaveOccurred())
			helpers.StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
		})
		AfterEach(func() {
			if os.Getenv("SKIP_TEARDOWN") == "true" {
				fmt.Println("Skipping Teardown...")
			} else {
				AppAfterEach()
			}
		})
		When("providing a valid app-guid together with a policy", func() {
			var params string
			var serviceInstanceName string
			var session *Session
			BeforeEach(func() {
				// Setup
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
			"metric_type": "disk",
			"threshold":500,
			"operator": ">=",
			"adjustment": "+1"
		},
		{
			"metric_type": "disk",
			"threshold": 300,
			"operator": "<",
			"adjustment": "-1"
		}
	]
}
`
				params = fmt.Sprintf(paramsTemplate, appToScaleGUID)
				helpers.ScaleDisk(cfg, appToScaleName, "1GB")
			})
			AfterEach(func() {
				if os.Getenv("SKIP_TEARDOWN") == "true" {
					fmt.Println("Skipping Teardown...")
				} else {
					helpers.DeleteServiceInstance(cfg, serviceInstanceName)
				}
			})

			It("succeeds and scales both, up and down", func() {
				// Execution
				serviceKeyName := fmt.Sprintf("aas-key_for%s", appToScaleName)
				session = helpers.CreateServiceKeyWithParams(
					serviceInstanceName, serviceKeyName, params, cfg.DefaultTimeoutDuration())

				// Validation
				By("Creating the service-key successfully")
				Expect(session).To(Exit(0))

				// Part-validation setup
				By("Starting disk usage to trigger scale out")
				helpers.StartDiskUsage(cfg, appToScaleName, diskUsageMb, holdMinutes+1)

				// Validation
				helpers.WaitForNInstancesRunning(appToScaleGUID, 2, 8*time.Minute)

				// Part-validation setup
				By("Stopping disk usage to trigger scale in")
				// only hit the one instance that was asked to occupy disk space
				helpers.StopDiskUsage(cfg, appToScaleName, 0)

				// Validation
				waitingTime := 8 * time.Minute // This validation can take a bit longer for unknown reasons.
				helpers.WaitForNInstancesRunning(appToScaleGUID, 1, waitingTime)
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
			//#nosec G402 -- acceptance test that uses test foundations without proper certs
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
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
