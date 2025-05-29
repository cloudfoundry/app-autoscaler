package main_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Operator", Serial, func() {
	var (
		runner       *OperatorRunner
		secondRunner *OperatorRunner

		healthHttpClient *http.Client

		healthURL *url.URL

		err error
	)
	BeforeEach(func() {
		initConfig()
		runner = NewOperatorRunner()

		healthURL, err = url.Parse(fmt.Sprintf("http://127.0.0.1:%d", conf.Health.ServerConfig.Port))
		Expect(err).NotTo(HaveOccurred())

		healthHttpClient = &http.Client{}
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("Using Database lock", func() {
		BeforeEach(func() {
			runner.startCheck = ""
			runner.configPath = writeConfig(&conf).Name()
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
				badfile, err := os.CreateTemp("", "bad-pr-config")
				Expect(err).NotTo(HaveOccurred())
				runner.configPath = badfile.Name()
				// #nosec G306
				err = os.WriteFile(runner.configPath, []byte("bogus"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
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

				conf.AppMetricsDb.CutoffDuration = -1

				conf := writeConfig(&conf)
				runner.configPath = conf.Name()
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
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.lock-acquired-in-first-attempt"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))
			})
		})

		Context("when operator have the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))
			})

			It("should retry acquiring lock to renew it's presence", func() {
				Eventually(runner.Session.Buffer, 8*time.Second).Should(Say("operator.retry-acquiring-lock"))

			})
		})

		Context("when interrupt occurs", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))
			})

			It("successfully release lock and exit", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.successfully-released-lock"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.exited"))
			})
		})

		Context("When one instance of operator owns lock and the other is waiting to get the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(Say("operator.started"))
				secondRunner = NewOperatorRunner()
				secondRunner.startCheck = ""
				conf.Health.BasicAuth.Username = ""
				conf.Health.BasicAuth.Password = ""
				conf.Health.ServerConfig.Port = 9000 + GinkgoParallelProcess()
				secondRunner.configPath = writeConfig(&conf).Name()
				secondRunner.Start()

			})

			AfterEach(func() {
				secondRunner.KillWithFire()
			})

			It("Competing instance should not get lock in first attempt", func() {
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(Say("operator.lock-acquired-in-first-attempt"))
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(Say("operator.successfully-acquired-lock"))

				By("checking the health endpoint of the standing-by instance")
				rsp, err := healthHttpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/health", conf.Health.ServerConfig.Port))
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

				conf.Health.BasicAuth.Username = ""
				conf.Health.BasicAuth.Password = ""
				conf.Health.ServerConfig.Port = 9000 + GinkgoParallelProcess()
				secondRunner.configPath = writeConfig(&conf).Name()
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
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))
					Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(Say("operator.lock-acquired-in-first-attempt"))
					Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(Say("operator.started"))
				} else {
					Eventually(secondRunner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))
					Consistently(runner.Session.Buffer, 5*time.Second).ShouldNot(Say("operator.lock-acquired-in-first-attempt"))
					Consistently(runner.Session.Buffer, 5*time.Second).ShouldNot(Say("operator.started"))
				}
			})
		})

		Context("when the running operator instance stopped", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 10*time.Second).Should(Say("operator.started"))
				secondRunner = NewOperatorRunner()
				conf.Health.ServerConfig.Port = 9000 + GinkgoParallelProcess()
				secondRunner.configPath = writeConfig(&conf).Name()
				secondRunner.startCheck = ""
				secondRunner.Start()
				Consistently(secondRunner.Session.Buffer, 10*time.Second).ShouldNot(Say("operator.lock-acquired-in-first-attempt"))
			})

			AfterEach(func() {
				secondRunner.ClearLockDatabase()
				secondRunner.KillWithFire()
			})

			It("competing operator instance should acquire the lock", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(Say("operator.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 5*time.Second).Should(Say("operator.successfully-released-lock"))
				Eventually(secondRunner.Session.Buffer, 10*time.Second).Should(Say("operator.successfully-acquired-lock"))
				Eventually(secondRunner.Session.Buffer, 15*time.Second).Should(Say("operator.started"))
			})
		})

		Context("when the operator acquires the lock", func() {
			JustBeforeEach(func() {
				runner.configPath = writeConfig(&conf).Name()
				runner.Start()
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

		When("db config with wrong credentials", func() {
			const dbUrl = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"

			type dbCase struct {
				key         string
				expectError string
			}

			DescribeTable("should error when db connection fails",
				func(tc dbCase) {
					localConf := conf
					localConf.Db[tc.key] = db.DatabaseConfig{URL: dbUrl}
					c := writeConfig(&localConf)
					runner.configPath = c.Name()
					runner.Start()
					defer os.Remove(runner.configPath)

					Eventually(runner.Session).Should(Exit(1))
					Expect(runner.Session.Buffer()).To(Say(tc.expectError))
				},
				Entry("appmetrics db fails", dbCase{
					key:         "appmetrics_db",
					expectError: "failed to connect appmetrics db",
				}),
				Entry("scalingengine db fails", dbCase{
					key:         "scalingengine_db",
					expectError: "failed to connect scalingengine db",
				}),
				Entry("policy db fails", dbCase{
					key:         "policy_db",
					expectError: "failed to connect policy db",
				}),
			)
		})
	})

	Describe("when Health server is ready to serve RESTful API", func() {
		BeforeEach(func() {
			conf.Health.BasicAuth = models.BasicAuth{
				Username: "",
				Password: "",
			}
			runner.configPath = writeConfig(&conf).Name()

			runner.Start()
			Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))
		})

		AfterEach(func() {
			runner.ClearLockDatabase()
		})

		When("a request to query health comes", func() {
			It("returns with a 200", func() {
				testhelpers.CheckHealthResponse(healthHttpClient, healthURL.String(), []string{
					"autoscaler_operator_policyDB", "autoscaler_operator_scalingEngineDB", "go_goroutines", "go_memstats_alloc_bytes",
				})
			})
		})
	})

	Describe("when Health server is ready to serve RESTful API with basic Auth", func() {
		BeforeEach(func() {
			runner.Start()

			Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))
		})

		AfterEach(func() {
			runner.ClearLockDatabase()
		})

		Context("when username and password are incorrect for basic authentication during health check", func() {
			It("should return 401", func() {
				testhelpers.CheckHealthAuth(GinkgoT(), healthHttpClient, healthURL.String(), "wrongusername", "wrongpassword", http.StatusUnauthorized)
			})
		})

		Context("when username and password are correct for basic authentication during health check", func() {
			It("should return 200", func() {
				testhelpers.CheckHealthAuth(GinkgoT(), healthHttpClient, healthURL.String(), conf.Health.BasicAuth.Username, conf.Health.BasicAuth.Password, http.StatusOK)
			})
		})
	})
})
