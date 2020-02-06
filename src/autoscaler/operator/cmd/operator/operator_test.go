package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Operator", func() {

	var (
		runner       *OperatorRunner
		secondRunner *OperatorRunner
	)
	BeforeEach(func() {
		initConfig()
		runner = NewOperatorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("Using Database lock", func() {

		BeforeEach(func() {
			runner.startCheck = ""
			runner.configPath = writeConfig(&cfg).Name()
		})

		AfterEach(func() {
			runner.ClearLockDatabase()
		})
		Context("with a missing config file", func() {
			BeforeEach(func() {
				runner.configPath = "bogus"
				runner.Start()
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to open config file"))
			})
		})

		Context("with an invalid config file", func() {
			BeforeEach(func() {
				badfile, err := ioutil.TempFile("", "bad-pr-config")
				Expect(err).NotTo(HaveOccurred())
				runner.configPath = badfile.Name()
				ioutil.WriteFile(runner.configPath, []byte("bogus"), os.ModePerm)
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to read config file"))
			})
		})

		Context("with missing/invalid configuration", func() {
			BeforeEach(func() {

				cfg.InstanceMetricsDB.CutoffDuration = -1

				cfg := writeConfig(&cfg)
				runner.configPath = cfg.Name()
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should fail validation", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to validate configuration"))
			})
		})
		Context("when operator acquires the lock in first attempt", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("successfully acquired lock and started", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.lock-acquired-in-first-attempt"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.started"))
			})
		})

		Context("when operator have the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.started"))
			})

			It("should retry acquiring lock to renew it's presence", func() {
				Eventually(runner.Session.Buffer, 8*time.Second).Should(gbytes.Say("operator.retry-acquiring-lock"))

			})
		})

		Context("when interrupt occurs", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.started"))
			})

			It("successfully release lock and exit", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.successfully-released-lock"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.exited"))
			})
		})

		Context("When one instance of operator owns lock and the other is waiting to get the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("operator.started"))
				secondRunner = NewOperatorRunner()
				secondRunner.startCheck = ""
				cfg.Health.HealthCheckUsername = ""
				cfg.Health.HealthCheckPassword = ""
				cfg.Health.Port = 9000 + GinkgoParallelNode()
				secondRunner.configPath = writeConfig(&cfg).Name()
				secondRunner.Start()

			})

			AfterEach(func() {
				secondRunner.KillWithFire()
			})

			It("Competing instance should not get lock in first attempt", func() {
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("operator.lock-acquired-in-first-attempt"))
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("operator.successfully-acquired-lock"))

				By("checking the health endpoint of the standing-by instance")
				rsp, err := healthHttpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/health", cfg.Health.Port))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))

			})
		})

		Context("When more than one instances of operator try to get the lock simultaneously", func() {

			var runnerAcquiredLock bool

			BeforeEach(func() {
				runner.Start()
				secondRunner = NewOperatorRunner()
				secondRunner.startCheck = ""

				cfg.Health.HealthCheckUsername = ""
				cfg.Health.HealthCheckPassword = ""
				cfg.Health.Port = 9000 + GinkgoParallelNode()
				secondRunner.configPath = writeConfig(&cfg).Name()
				secondRunner.Start()
			})

			JustBeforeEach(func() {
				runnerAcquiredLock = true
				buffer := runner.Session.Out
				secondBuffer := secondRunner.Session.Out
				select {
				case <-buffer.Detect("operator.lock-acquired-in-first-attempt"):
					runnerAcquiredLock = true
				case <-secondBuffer.Detect("operator.lock-acquired-in-first-attempt"):
					runnerAcquiredLock = false
				case <-time.After(2 * time.Second):
				}
				buffer.CancelDetects()
				secondBuffer.CancelDetects()
			})

			AfterEach(func() {
				secondRunner.KillWithFire()
			})

			It("Only one instance should get the lock", func() {
				if runnerAcquiredLock {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.started"))
					Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("operator.lock-acquired-in-first-attempt"))
					Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("operator.started"))
				} else {
					Eventually(secondRunner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.started"))
					Consistently(runner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("operator.lock-acquired-in-first-attempt"))
					Consistently(runner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("operator.started"))
				}
			})
		})

		Context("when the running operator instance stopped", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 10*time.Second).Should(gbytes.Say("operator.started"))
				secondRunner = NewOperatorRunner()
				cfg.Health.Port = 9000 + GinkgoParallelNode()
				secondRunner.configPath = writeConfig(&cfg).Name()
				secondRunner.startCheck = ""
				secondRunner.Start()
				Consistently(secondRunner.Session.Buffer, 10*time.Second).ShouldNot(gbytes.Say("operator.lock-acquired-in-first-attempt"))
			})

			AfterEach(func() {
				secondRunner.ClearLockDatabase()
				secondRunner.KillWithFire()
			})

			It("competing operator instance should acquire the lock", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("operator.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("operator.successfully-released-lock"))
				Eventually(secondRunner.Session.Buffer, 10*time.Second).Should(gbytes.Say("operator.successfully-acquired-lock"))
				Eventually(secondRunner.Session.Buffer, 15*time.Second).Should(gbytes.Say("operator.started"))
			})
		})

		Context("when the operator acquires the lock", func() {
			JustBeforeEach(func() {
				runner.configPath = writeConfig(&cfg).Name()
				runner.Start()
			})

			It("should start instancemetrics dbpruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.instancemetrics-dbpruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should start appmetrics dbpruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.appmetrics-dbpruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should start scalingengine dbpruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.scalingengine-dbpruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should start appsyncer", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.application-sync.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should have operator started", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

		})

		Context("when connection to instancemetrics db fails", func() {
			BeforeEach(func() {
				cfg.InstanceMetricsDB.DB.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
				cfg := writeConfig(&cfg)
				runner.configPath = cfg.Name()
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to connect instancemetrics db"))
			})

		})

		Context("when connection to appmetrics db fails", func() {
			BeforeEach(func() {
				cfg.AppMetricsDB.DB.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
				cfg := writeConfig(&cfg)
				runner.configPath = cfg.Name()
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to connect appmetrics db"))
			})

		})

		Context("when connection to scalingengine db fails", func() {
			BeforeEach(func() {
				cfg.ScalingEngineDB.DB.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
				cfg := writeConfig(&cfg)
				runner.configPath = cfg.Name()
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to connect scalingengine db"))
			})

		})

		Context("when connection to apsyncer policy db fails", func() {
			BeforeEach(func() {
				cfg.AppSyncer.DB.URL = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
				cfg := writeConfig(&cfg)
				runner.configPath = cfg.Name()
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(Say("failed to connect policy db"))
			})

		})
	})

	Describe("when Health server is ready to serve RESTful API", func() {
		BeforeEach(func() {
			basicAuthConfig := cfg
			basicAuthConfig.Health.HealthCheckUsername = ""
			basicAuthConfig.Health.HealthCheckPassword = ""
			runner.configPath = writeConfig(&basicAuthConfig).Name()

			runner.Start()
			Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.started"))
		})

		AfterEach(func() {
			runner.ClearLockDatabase()
		})

		Context("when a request to query health comes", func() {
			It("returns with a 200", func() {
				rsp, err := healthHttpClient.Get(fmt.Sprintf("http://127.0.0.1:%d", healthport))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				raw, _ := ioutil.ReadAll(rsp.Body)
				healthData := string(raw)
				Expect(healthData).To(ContainSubstring("autoscaler_operator_policyDB"))
				Expect(healthData).To(ContainSubstring("autoscaler_operator_instanceMetricsDB"))
				Expect(healthData).To(ContainSubstring("autoscaler_operator_appMetricsDB"))
				Expect(healthData).To(ContainSubstring("autoscaler_operator_scalingEngineDB"))
				Expect(healthData).To(ContainSubstring("go_goroutines"))
				Expect(healthData).To(ContainSubstring("go_memstats_alloc_bytes"))
				rsp.Body.Close()

			})
		})
		Context("when a request to query profile comes", func() {
			It("returns with a 200", func() {
				rsp, err := healthHttpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/debug/pprof", healthport))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				raw, _ := ioutil.ReadAll(rsp.Body)
				profileIndexBody := string(raw)
				Expect(profileIndexBody).To(ContainSubstring("allocs"))
				Expect(profileIndexBody).To(ContainSubstring("block"))
				Expect(profileIndexBody).To(ContainSubstring("cmdline"))
				Expect(profileIndexBody).To(ContainSubstring("goroutine"))
				Expect(profileIndexBody).To(ContainSubstring("heap"))
				Expect(profileIndexBody).To(ContainSubstring("mutex"))
				Expect(profileIndexBody).To(ContainSubstring("profile"))
				Expect(profileIndexBody).To(ContainSubstring("threadcreate"))
				Expect(profileIndexBody).To(ContainSubstring("trace"))
				rsp.Body.Close()

			})
		})
	})

	Describe("when Health server is ready to serve RESTful API with basic Auth", func() {
		BeforeEach(func() {
			runner.Start()

			Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("operator.started"))
		})

		AfterEach(func() {
			runner.ClearLockDatabase()
		})

		Context("when username and password are incorrect for basic authentication during health check", func() {
			It("should return 401", func() {

				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", healthport), nil)
				Expect(err).NotTo(HaveOccurred())

				req.SetBasicAuth("wrongusername", "wrongpassword")

				rsp, err := healthHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when username and password are correct for basic authentication during health check", func() {
			It("should return 200", func() {

				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", healthport), nil)
				Expect(err).NotTo(HaveOccurred())

				req.SetBasicAuth(cfg.Health.HealthCheckUsername, cfg.Health.HealthCheckPassword)

				rsp, err := healthHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})
})
