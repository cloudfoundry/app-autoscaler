package main_test

import (
	"autoscaler/pruner"
	"autoscaler/pruner/config"
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

var _ = Describe("Pruner", func() {

	var (
		runner       *PrunerRunner
		secondRunner *PrunerRunner
		consulClient consuladapter.Client
		consulConfig config.Config
	)
	BeforeEach(func() {
		initConfig()
		consulRunner.Reset()
		consulClient = consulRunner.NewClient()
		runner = NewPrunerRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("Using Consul distributed lock", func() {

		Context("when the pruner acquires the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
			})

			It("should start instancemetrics dbpruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.instancemetrics-dbpruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should start appmetrics dbpruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.appmetrics-dbpruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should start scalingengine dbpruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.scalingengine-dbpruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should have pruner started", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})
		})

		Context("when the pruner loses the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()

				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.started"))

				consulRunner.Reset()
			})

			It("exits with failure", func() {
				Eventually(runner.Session, 4*time.Second).Should(Exit(1))
				Expect(runner.Session.Buffer()).Should(Say("exited-with-failure"))
			})
		})

		Context("when the pruner initially does not have the lock", func() {
			var competingPrunerProcess ifrit.Process

			BeforeEach(func() {
				logger := lagertest.NewTestLogger("competing-process")
				buffer := logger.Buffer()

				competingPrunerLock := locket.NewLock(logger, consulClient, pruner.PrunerLockSchemaPath(), []byte{}, clock.NewClock(), cfg.Lock.LockRetryInterval, cfg.Lock.LockTTL)
				competingPrunerProcess = ifrit.Invoke(competingPrunerLock)
				Eventually(buffer, 2*time.Second).Should(Say("competing-process.lock.acquire-lock-succeeded"))

				runner.startCheck = ""
				runner.Start()
			})

			It("should not start", func() {
				Eventually(runner.Session.Buffer).Should(Say("pruner.lock.acquiring-lock"))
				Consistently(runner.Session.Buffer).ShouldNot(Say("pruner.started"))
			})

			Describe("when the lock becomes available", func() {
				BeforeEach(func() {
					ginkgomon.Kill(competingPrunerProcess)
				})

				It("should acquire the lock and start instancemetrics dbpruner", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.instancemetrics-dbpruner.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})

				It("should acquire the lock and start appmetrics dbpruner", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.appmetrics-dbpruner.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})

				It("should acquire the lock and start scalingengine dbpruner", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.scalingengine-dbpruner.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})

				It("should have pruner started", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.started"))
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

				cfg.InstanceMetricsDb.CutoffDays = -1

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

			It("should not get pruner service", func() {
				Eventually(func() map[string]*api.AgentService {
					services, err := consulClient.Agent().Services()
					Expect(err).ToNot(HaveOccurred())
					return services
				}).ShouldNot(HaveKey("pruner"))
			})

			It("should start pruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

		})

		Context("when connection to instancemetrics db fails", func() {
			BeforeEach(func() {
				cfg.InstanceMetricsDb.Db.Url = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
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
				cfg.AppMetricsDb.Db.Url = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
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
				cfg.ScalingEngineDb.Db.Url = "postgres://not-exist-user:not-exist-password@localhost/autoscaler?sslmode=disable"
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

		Context("when pruner acquires the lock in first attempt", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("successfully acquired lock and started", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("pruner.lock-acquired-in-first-attempt"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("pruner.started"))
			})
		})

		Context("when pruner have the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("pruner.started"))
			})

			It("should retry acquiring lock to renew it's presence", func() {
				Eventually(runner.Session.Buffer, 8*time.Second).Should(gbytes.Say("pruner.retry-acquiring-lock"))

			})
		})

		Context("when interrupt occurs", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("pruner.started"))
			})

			It("successfully release lock and exit", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("pruner.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("pruner.successfully-released-lock"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("pruner.exited"))
			})
		})

		Context("When one instance of pruner owns lock and the other is waiting to get the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("pruner.started"))
				secondRunner = NewPrunerRunner()
				secondRunner.startCheck = ""
				secondRunner.configPath = writeConfig(&consulConfig).Name()
				secondRunner.Start()

			})

			AfterEach(func() {
				secondRunner.KillWithFire()
			})

			It("Competing instance should not get lock in first attempt", func() {
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("pruner.lock-acquired-in-first-attempt"))
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("pruner.successfully-acquired-lock"))
			})
		})

		Context("when the running pruner instance stopped", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 10*time.Second).Should(gbytes.Say("pruner.started"))
				secondRunner = NewPrunerRunner()
				secondRunner.configPath = writeConfig(&consulConfig).Name()
				secondRunner.startCheck = ""
				secondRunner.Start()
				Consistently(secondRunner.Session.Buffer, 10*time.Second).ShouldNot(gbytes.Say("pruner.lock-acquired-in-first-attempt"))
			})

			AfterEach(func() {
				secondRunner.ClearLockDatabase()
				secondRunner.KillWithFire()
			})

			It("competing pruner instance should acquire the lock", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("pruner.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("pruner.successfully-released-lock"))
				Eventually(secondRunner.Session.Buffer, 10*time.Second).Should(gbytes.Say("pruner.successfully-acquired-lock"))
				Eventually(secondRunner.Session.Buffer, 15*time.Second).Should(gbytes.Say("pruner.started"))
			})
		})

		Context("when the pruner acquires the lock and consul configuration is provided", func() {
			JustBeforeEach(func() {
				consulConfig = cfg
				Expect(consulConfig.Lock.ConsulClusterConfig).ShouldNot(BeEmpty())
				consulConfig.EnableDBLock = true
				runner.configPath = writeConfig(&consulConfig).Name()
				runner.Start()
			})

			It("should start instancemetrics dbpruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.instancemetrics-dbpruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should start appmetrics dbpruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.appmetrics-dbpruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should start scalingengine dbpruner", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.scalingengine-dbpruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

			It("should have pruner started", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("pruner.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

		})
	})
})
