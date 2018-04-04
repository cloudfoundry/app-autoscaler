package main_test

import (
	"autoscaler/eventgenerator"
	"autoscaler/eventgenerator/config"
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

var _ = Describe("Eventgenerator", func() {
	var (
		runner       *EventGeneratorRunner
		consulClient consuladapter.Client
		secondRunner *EventGeneratorRunner
		consulConfig config.Config
	)

	BeforeEach(func() {
		consulRunner.Reset()
		consulClient = consulRunner.NewClient()
		runner = NewEventGeneratorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("Using Consul distributed lock", func() {

		Context("when the eventgenerator acquires the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()

				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
			})

			It("should start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.started"))
				Consistently(runner.Session).ShouldNot(Exit())
				Eventually(func() bool { return len(metricCollector.ReceivedRequests()) >= 1 }, 5*time.Second).Should(BeTrue())
				Eventually(func() bool { return len(scalingEngine.ReceivedRequests()) >= 1 }, time.Duration(2*breachDurationSecs)*time.Second).Should(BeTrue())
			})
		})

		Context("when the eventgenerator loses the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()

				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.started"))

				consulRunner.Reset()
			})

			It("exits with failure", func() {
				Eventually(runner.Session, 4*time.Second).Should(Exit(1))
				Expect(runner.Session.Buffer()).Should(Say("exited-with-failure"))
			})
		})

		Context("when the eventgenerator initially does not have the lock", func() {
			var competingEventGeneratorProcess ifrit.Process

			BeforeEach(func() {
				logger := lagertest.NewTestLogger("competing-process")
				buffer := logger.Buffer()

				competingEventGeneratorLock := locket.NewLock(logger, consulClient, eventgenerator.EventGeneratorLockSchemaPath(), []byte{}, clock.NewClock(), conf.Lock.LockRetryInterval, conf.Lock.LockTTL)
				competingEventGeneratorProcess = ifrit.Invoke(competingEventGeneratorLock)
				Eventually(buffer, 2*time.Second).Should(gbytes.Say("competing-process.lock.acquire-lock-succeeded"))

				runner.startCheck = ""
				runner.Start()
			})

			It("should not start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.lock.acquiring-lock"))
				Consistently(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("eventgenerator.started"))
			})

			Describe("when the lock becomes available", func() {
				BeforeEach(func() {
					ginkgomon.Kill(competingEventGeneratorProcess)
				})

				It("acquires the lock and starts", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.started"))
					Consistently(runner.Session).ShouldNot(Exit())
					Eventually(func() bool { return len(metricCollector.ReceivedRequests()) >= 1 }, 5*time.Second).Should(BeTrue())
					Eventually(func() bool { return len(scalingEngine.ReceivedRequests()) >= 1 }, time.Duration(2*breachDurationSecs)*time.Second).Should(BeTrue())
				})

			})
		})

		Context("with a missing config file", func() {
			BeforeEach(func() {
				runner.startCheck = ""
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
				runner.startCheck = ""
				badfile, err := ioutil.TempFile("", "bad-mc-config")
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
				Expect(runner.Session.Buffer()).To(Say("failed to parse config file"))
			})
		})

		Context("with missing configuration", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				conf := &config.Config{
					Logging: config.LoggingConfig{
						Level: "debug",
					},
					Aggregator: config.AggregatorConfig{
						AggregatorExecuteInterval: 2 * time.Second,
						PolicyPollerInterval:      2 * time.Second,
						MetricPollerCount:         2,
						AppMonitorChannelSize:     2,
					},
					Evaluator: config.EvaluatorConfig{
						EvaluationManagerInterval: 2 * time.Second,
						EvaluatorCount:            2,
						TriggerArrayChannelSize:   2,
					},
				}
				configFile := writeConfig(conf)
				runner.configPath = configFile.Name()
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
				noConsulConf := conf
				noConsulConf.Lock.ConsulClusterConfig = ""
				runner.configPath = writeConfig(&noConsulConf).Name()
				runner.startCheck = ""
				runner.Start()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should not get eventgenerator service", func() {
				Eventually(func() map[string]*api.AgentService {
					services, err := consulClient.Agent().Services()
					Expect(err).ToNot(HaveOccurred())
					return services
				}).ShouldNot(HaveKey("eventgenerator"))
			})

			It("should start eventgenerator", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("eventgenerator.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})

		})

		Context("when an interrupt is sent", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("should stop", func() {
				runner.Session.Interrupt()
				Eventually(runner.Session, 5).Should(Exit(0))
			})
		})

	})

	Describe("Using Database lock", func() {

		BeforeEach(func() {
			consulConfig = conf
			consulConfig.EnableDBLock = true
			consulConfig.Lock.ConsulClusterConfig = ""
			runner.startCheck = ""
			runner.configPath = writeConfig(&consulConfig).Name()
		})

		AfterEach(func() {
			runner.ClearLockDatabase()
		})

		Context("when eventgenerator acquires the lock in first attempt", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("successfully acquired lock and started", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.lock-acquired-in-first-attempt"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.started"))
			})
		})

		Context("when eventgenerator have the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.started"))
			})

			It("should retry acquiring lock to renew it's presence", func() {
				Eventually(runner.Session.Buffer, 8*time.Second).Should(gbytes.Say("eventgenerator.retry-acquiring-lock"))

			})
		})

		Context("when interrupt occurs", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.started"))
			})

			It("successfully release lock and exit", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.successfully-released-lock"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.exited"))
			})
		})

		Context("When one instance of eventgenerator owns lock and the other is waiting to get the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("eventgenerator.started"))
				secondRunner = NewEventGeneratorRunner()
				secondRunner.startCheck = ""
				secondRunner.configPath = writeConfig(&consulConfig).Name()
				secondRunner.Start()

			})

			AfterEach(func() {
				secondRunner.KillWithFire()
			})

			It("Competing instance should not get lock in first attempt", func() {
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("eventgenerator.lock-acquired-in-first-attempt"))
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("eventgenerator.successfully-acquired-lock"))
			})
		})

		Context("when the running eventgenerator instance stopped", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 10*time.Second).Should(gbytes.Say("eventgenerator.started"))
				secondRunner = NewEventGeneratorRunner()
				secondRunner.configPath = writeConfig(&consulConfig).Name()
				secondRunner.startCheck = ""
				secondRunner.Start()
				Consistently(secondRunner.Session.Buffer, 10*time.Second).ShouldNot(gbytes.Say("eventgenerator.lock-acquired-in-first-attempt"))
			})

			AfterEach(func() {
				secondRunner.ClearLockDatabase()
				secondRunner.KillWithFire()
			})

			It("competing eventgenerator instance should acquire the lock", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("eventgenerator.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("eventgenerator.successfully-released-lock"))
				Eventually(secondRunner.Session.Buffer, 10*time.Second).Should(gbytes.Say("eventgenerator.successfully-acquired-lock"))
				Eventually(secondRunner.Session.Buffer, 15*time.Second).Should(gbytes.Say("eventgenerator.started"))
			})
		})

		Context("when the eventgenerator acquires the lock and consul configuration is provided", func() {
			JustBeforeEach(func() {
				consulConfig = conf
				Expect(consulConfig.Lock.ConsulClusterConfig).ShouldNot(BeEmpty())
				consulConfig.EnableDBLock = true
				runner.configPath = writeConfig(&consulConfig).Name()
				runner.Start()
			})

			It("should start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("eventgenerator.started"))
				Consistently(runner.Session).ShouldNot(Exit())
				Eventually(func() bool { return len(metricCollector.ReceivedRequests()) >= 1 }, 5*time.Second).Should(BeTrue())
				Eventually(func() bool { return len(scalingEngine.ReceivedRequests()) >= 1 }, time.Duration(2*breachDurationSecs)*time.Second).Should(BeTrue())
			})
		})
	})

})
