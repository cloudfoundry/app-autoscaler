package main_test

import (
	"autoscaler/cf"
	"autoscaler/models"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/onsi/gomega/gbytes"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

var _ = Describe("Main", func() {

	var (
		runner *ScalingEngineRunner
	)

	BeforeEach(func() {
		runner = NewScalingEngineRunner()
	})

	JustBeforeEach(func() {
		runner.Start()
	})

	AfterEach(func() {
		runner.KillWithFire()
		ClearLockDatabase()
	})

	Describe("with a correct config with db lock enabled", func() {

		Context("when starting 1 scaling engine instance", func() {
			It("scaling engine should start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.startCheck))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("http server starts directly", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.http-server.new-http-server"))
			})

			Context("schedule sychronizer acquires the lock ", func() {
				It("acquired the first lock and started", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.lock-acquired-in-first-attempt"))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.synchronizer.started"))
				})

				It("started and renew the lock", func() {
					Eventually(runner.Session.Buffer, 2).Should(gbytes.Say("scalingengine.synchronizer.started"))
					Eventually(runner.Session.Buffer, 8*time.Second).Should(gbytes.Say("scalingengine.retry-acquiring-lock"))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.lock-db.renewed-lock-successfully"))

				})
			})

			Context("when an interrupt is sent", func() {
				JustBeforeEach(func() {
					Eventually(runner.Session.Buffer, 2).Should(gbytes.Say("scalingengine.started"))
				})

				It("should stop", func() {
					runner.Session.Interrupt()
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.exited"))
					Eventually(runner.Session, 5).Should(Exit(0))
				})
			})

		})

		Context("when starting multiple scaling engine instances", func() {
			var (
				secondRunner *ScalingEngineRunner
			)

			JustBeforeEach(func() {
				secondRunner = NewScalingEngineRunner()
				conf.Server.Port += 1
				conf.Health.Port += 1
				secondRunner.configPath = writeConfig(&conf).Name()
				secondRunner.Start()
			})

			AfterEach(func() {
				secondRunner.KillWithFire()
			})

			It("2 http server instances start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.http-server.new-http-server"))
				Eventually(secondRunner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.http-server.new-http-server"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.started"))
				Eventually(secondRunner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.started"))

				Consistently(runner.Session).ShouldNot(Exit())
				Consistently(secondRunner.Session).ShouldNot(Exit())
			})

			Context("Only 1 schedule sychronizer starts ", func() {

				var (
					runnerAcquiredLock bool
				)

				JustBeforeEach(func() {
					runnerAcquiredLock = true
					buffer := runner.Session.Out
					secondBuffer := secondRunner.Session.Out
					select {
					case <-buffer.Detect("scalingengine.lock-acquired-in-first-attempt"):
						runnerAcquiredLock = true
					case <-secondBuffer.Detect("scalingengine.lock-acquired-in-first-attempt"):
						runnerAcquiredLock = false
					case <-time.After(2 * time.Second):
					}
					buffer.CancelDetects()
					secondBuffer.CancelDetects()
				})

				It("Only 1 schedule sychronizer could acquire the lock and start", func() {
					if runnerAcquiredLock {
						Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.synchronizer.started"))
						Eventually(secondRunner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("scalingengine.lock-acquired-in-first-attempt"))
						Eventually(secondRunner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("scalingengine.synchronizer.started"))
					} else {
						Eventually(secondRunner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.synchronizer.started"))
						Eventually(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("scalingengine.lock-acquired-in-first-attempt"))
						Eventually(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("scalingengine.synchronizer.started"))
					}
				})

				It("Only 1 schedule sychronizer could renew the lock successfully", func() {
					if runnerAcquiredLock {
						Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.synchronizer.started"))
						Eventually(runner.Session.Buffer, 8*time.Second).Should(gbytes.Say("scalingengine.retry-acquiring-lock"))
						Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.lock-db.renewed-lock-successfully"))
						Eventually(secondRunner.Session.Buffer, 8*time.Second).ShouldNot(gbytes.Say("scalingengine.retry-acquiring-lock"))
						Eventually(secondRunner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("scalingengine.lock-db.renewed-lock-successfully"))
					} else {
						Eventually(secondRunner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.synchronizer.started"))
						Eventually(secondRunner.Session.Buffer, 8*time.Second).Should(gbytes.Say("scalingengine.retry-acquiring-lock"))
						Eventually(secondRunner.Session.Buffer, 2*time.Second).Should(gbytes.Say("scalingengine.lock-db.renewed-lock-successfully"))
						Eventually(runner.Session.Buffer, 8*time.Second).ShouldNot(gbytes.Say("scalingengine.retry-acquiring-lock"))
						Eventually(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("scalingengine.lock-db.renewed-lock-successfully"))
					}
				})

				It("When the running sychronizer loses the lock, the other one takes over", func() {
					if runnerAcquiredLock {
						Eventually(secondRunner.Session.Buffer, 10*time.Second).Should(gbytes.Say("retry-acquiring-lock"))
						runner.Interrupt()
						Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("scalingengine.received-interrupt-signal"))
						Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("scalingengine.successfully-released-lock"))
						Eventually(secondRunner.Session.Buffer, 10*time.Second).Should(gbytes.Say("scalingengine.successfully-acquired-lock"))
					} else {
						Eventually(runner.Session.Buffer, 10*time.Second).Should(gbytes.Say("retry-acquiring-lock"))
						secondRunner.Interrupt()
						Eventually(secondRunner.Session.Buffer, 5*time.Second).Should(gbytes.Say("scalingengine.received-interrupt-signal"))
						Eventually(secondRunner.Session.Buffer, 5*time.Second).Should(gbytes.Say("scalingengine.successfully-released-lock"))
						Eventually(runner.Session.Buffer, 10*time.Second).Should(gbytes.Say("scalingengine.successfully-acquired-lock"))
					}
				})

			})

		})

	})
	Describe("With incorrect config", func() {

		Context("with a missing config file", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.configPath = "bogus"
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(gbytes.Say("failed to open config file"))
			})
		})

		Context("with an invalid config file", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				badfile, err := ioutil.TempFile("", "bad-engine-config")
				Expect(err).NotTo(HaveOccurred())
				runner.configPath = badfile.Name()
				ioutil.WriteFile(runner.configPath, []byte("bogus"), os.ModePerm)
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(gbytes.Say("failed to read config file"))
			})
		})

		Context("with missing configuration", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				missingParamConf := conf
				missingParamConf.Cf = cf.CfConfig{
					Api: ccUAA.URL(),
				}

				missingParamConf.Server.Port = 7000 + GinkgoParallelNode()
				missingParamConf.Logging.Level = "debug"

				cfg := writeConfig(&missingParamConf)
				runner.configPath = cfg.Name()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should fail validation", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(gbytes.Say("failed to validate configuration"))
			})
		})
	})

	Describe("when http server is ready to serve RESTful API", func() {

		JustBeforeEach(func() {
			Eventually(runner.Session.Buffer, 2).Should(gbytes.Say("scalingengine.started"))
		})

		Context("when a request to trigger scaling comes", func() {
			It("returns with a 200", func() {
				body, err := json.Marshal(models.Trigger{Adjustment: "+1"})
				Expect(err).NotTo(HaveOccurred())

				rsp, err := httpClient.Post(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/scale", port, appId),
					"application/json", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

		Context("when a request to retrieve scaling history comes", func() {
			It("returns with a 200", func() {
				rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/scaling_histories", port, appId))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

		It("handles the start and end of a schedule", func() {
			By("start of a schedule")
			url := fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/active_schedules/111111", port, appId)
			bodyReader := bytes.NewReader([]byte(`{"instance_min_count":1, "instance_max_count":5, "initial_min_instance_count":3}`))

			req, err := http.NewRequest(http.MethodPut, url, bodyReader)
			Expect(err).NotTo(HaveOccurred())

			rsp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			rsp.Body.Close()

			By("end of a schedule")
			req, err = http.NewRequest(http.MethodDelete, url, nil)
			Expect(err).NotTo(HaveOccurred())

			rsp, err = httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusNoContent))
			rsp.Body.Close()
		})
	})

	Describe("when Health server is ready to serve RESTful API", func() {
		JustBeforeEach(func() {
			Eventually(runner.Session.Buffer, 2).Should(gbytes.Say("scalingengine.started"))
		})

		Context("when a request to query health comes", func() {
			It("returns with a 200", func() {
				rsp, err := healthHttpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/health", healthport))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				raw, _ := ioutil.ReadAll(rsp.Body)
				healthData := string(raw)
				Expect(healthData).To(ContainSubstring("autoscaler_scalingengine_concurrentHTTPReq"))
				Expect(healthData).To(ContainSubstring("autoscaler_scalingengine_openConnection_policyDB"))
				Expect(healthData).To(ContainSubstring("autoscaler_scalingengine_openConnection_scalingEngineDB"))
				Expect(healthData).To(ContainSubstring("autoscaler_scalingengine_openConnection_schedulerDB"))
				Expect(healthData).To(ContainSubstring("go_goroutines"))
				Expect(healthData).To(ContainSubstring("go_memstats_alloc_bytes"))
				rsp.Body.Close()

			})
		})
	})
})
