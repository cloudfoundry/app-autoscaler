package main_test

import (
	"autoscaler/operator"
	"autoscaler/operator/config"
	"io/ioutil"
	"os"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/locket"
	"github.com/hashicorp/consul/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("Operator", func() {

	var (
		runner       *OperatorRunner
		secondRunner *OperatorRunner
		consulClient consuladapter.Client
		consulConfig config.Config
	)
	BeforeEach(func() {
		initConfig()
		consulRunner.Reset()
		consulClient = consulRunner.NewClient()
		runner = NewOperatorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("Using Consul distributed lock", func() {

		Context("when the operator acquires the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
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

			It("should start scalingengine sync", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.scalingengine-sync.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})
			It("should start scalingengine sync", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.scheduler-sync.started"))
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

		Context("when the operator loses the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()

				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))

				consulRunner.Reset()
			})

			It("exits with failure", func() {
				Eventually(runner.Session, 4*time.Second).Should(Exit(1))
				Expect(runner.Session.Buffer()).Should(Say("exited-with-failure"))
			})
		})

		Context("when the operator initially does not have the lock", func() {
			var competingOperatorProcess ifrit.Process

			BeforeEach(func() {
				logger := lagertest.NewTestLogger("competing-process")
				buffer := logger.Buffer()

				competingOperatorLock := locket.NewLock(logger, consulClient, operator.OperatorLockSchemaPath(), []byte{}, clock.NewClock(), cfg.Lock.LockRetryInterval, cfg.Lock.LockTTL)
				competingOperatorProcess = ifrit.Invoke(competingOperatorLock)
				Eventually(buffer, 2*time.Second).Should(Say("competing-process.lock.acquire-lock-succeeded"))

				runner.startCheck = ""
				runner.Start()
			})

			It("should not start", func() {
				Eventually(runner.Session.Buffer).Should(Say("operator.lock.acquiring-lock"))
				Consistently(runner.Session.Buffer).ShouldNot(Say("operator.started"))
			})

			Describe("when the lock becomes available", func() {
				BeforeEach(func() {
					ginkgomon.Kill(competingOperatorProcess)
				})

				It("should acquire the lock and start instancemetrics dbpruner", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.instancemetrics-dbpruner.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})

				It("should acquire the lock and start appmetrics dbpruner", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.appmetrics-dbpruner.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})

				It("should acquire the lock and start scalingengine dbpruner", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.scalingengine-dbpruner.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})

				It("should acquire the lock and start appsyncer", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.application-sync.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})

				It("should have pruner started", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("operator.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})
			})
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

				cfg.InstanceMetricsDB.CutoffDays = -1

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

		Context("when no consul is configured", func() {
			BeforeEach(func() {
				noConsulConf := cfg
				noConsulConf.Lock.ConsulClusterConfig = ""
				runner.configPath = writeConfig(&noConsulConf).Name()
				runner.startCheck = ""
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should not get operator service", func() {
				Eventually(func() map[string]*api.AgentService {
					services, err := consulClient.Agent().Services()
					Expect(err).ToNot(HaveOccurred())
					return services
				}).ShouldNot(HaveKey("operator"))
			})

			It("should start operator", func() {
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

		Context("when an interrupt is sent", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("should stop", func() {
				runner.Interrupt()
				Eventually(runner.Session, 5).Should(Exit(130))
			})
		})
	})

	Describe("Using Database lock", func() {

		BeforeEach(func() {
			consulConfig = cfg
			consulConfig.EnableDBLock = true
			consulConfig.Lock.ConsulClusterConfig = ""
			runner.startCheck = ""
			runner.configPath = writeConfig(&consulConfig).Name()
		})

		AfterEach(func() {
			runner.ClearLockDatabase()
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
				secondRunner.configPath = writeConfig(&consulConfig).Name()
				secondRunner.Start()

			})

			AfterEach(func() {
				secondRunner.KillWithFire()
			})

			It("Competing instance should not get lock in first attempt", func() {
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("operator.lock-acquired-in-first-attempt"))
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("operator.successfully-acquired-lock"))
			})
		})

		Context("When more than one instances of operator try to get the lock simultaneously", func() {

			var runnerAcquiredLock bool

			BeforeEach(func() {
				runner.Start()
				secondRunner = NewOperatorRunner()
				secondRunner.startCheck = ""
				secondRunner.configPath = writeConfig(&consulConfig).Name()
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
				secondRunner.configPath = writeConfig(&consulConfig).Name()
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

		Context("when the operator acquires the lock and consul configuration is provided", func() {
			JustBeforeEach(func() {
				consulConfig = cfg
				Expect(consulConfig.Lock.ConsulClusterConfig).ShouldNot(BeEmpty())
				consulConfig.EnableDBLock = true
				runner.configPath = writeConfig(&consulConfig).Name()
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
	})
})
