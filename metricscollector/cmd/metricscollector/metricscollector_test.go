package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/locket"

	"github.com/hashicorp/consul/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"autoscaler/cf"
	"autoscaler/metricscollector"
	"autoscaler/metricscollector/config"

	"code.cloudfoundry.org/consuladapter"
	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("MetricsCollector", func() {
	var (
		runner       *MetricsCollectorRunner
		secondRunner *MetricsCollectorRunner
		consulClient consuladapter.Client
		consulConfig config.Config
	)

	BeforeEach(func() {
		consulRunner.Reset()
		consulClient = consulRunner.NewClient()
		runner = NewMetricsCollectorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("Using consul distributed lock", func() {

		Context("when the metricscollector acquires the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
			})

			It("should start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.collector.collector-started"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})
		})

		Context("when the metricscollector loses the lock", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.Start()

				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))

				consulRunner.Reset()
			})

			It("exits with failure", func() {
				Eventually(runner.Session.Buffer, 4*time.Second).Should(gbytes.Say("exited-with-failure"))
				Eventually(runner.Session).Should(Exit(1))
			})
		})

		Context("when the metricscollector initially does not have the lock", func() {
			var competingMetricsCollectorProcess ifrit.Process

			BeforeEach(func() {
				logger := lagertest.NewTestLogger("competing-process")
				buffer := logger.Buffer()

				competingMetricsCollectorLock := locket.NewLock(logger, consulClient, metricscollector.MetricsCollectorLockSchemaPath(), []byte{}, clock.NewClock(), cfg.Lock.LockRetryInterval, cfg.Lock.LockTTL)
				competingMetricsCollectorProcess = ifrit.Invoke(competingMetricsCollectorLock)
				Eventually(buffer, 2*time.Second).Should(gbytes.Say("competing-process.lock.acquire-lock-succeeded"))

				runner.startCheck = ""
				runner.Start()
			})

			It("should not start", func() {
				Consistently(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("metricscollector.collector.collector-started"))
				Consistently(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("metricscollector.registration-runner"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.lock.acquiring-lock"))
				Consistently(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("metricscollector.started"))
			})

			Describe("when the lock becomes available", func() {
				BeforeEach(func() {
					ginkgomon.Kill(competingMetricsCollectorProcess)
				})

				It("acquires the lock and starts", func() {
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
					Consistently(runner.Session).ShouldNot(Exit())
				})

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

	Describe("Metricscollector configuration check", func() {

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
				Expect(runner.Session.Buffer()).To(Say("failed to read config file"))
			})
		})

		Context("with missing configuration", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				missingConfig := cfg
				missingConfig.Cf = cf.CfConfig{
					Api: ccNOAAUAA.URL(),
				}

				missingConfig.Server.Port = 7000 + GinkgoParallelNode()
				missingConfig.Logging.Level = "debug"
				runner.configPath = writeConfig(&missingConfig).Name()
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

			It("should not get metricscollector service", func() {
				Eventually(func() map[string]*api.AgentService {
					services, err := consulClient.Agent().Services()
					Expect(err).ToNot(HaveOccurred())
					return services
				}).ShouldNot(HaveKey("metricscollector"))
			})

			It("should start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})
		})
	})

	Describe("MetricsCollector REST API", func() {
		Context("when a request for metrics history comes", func() {
			Context("when using polling for metrics collection", func() {
				BeforeEach(func() {
					runner.Start()
				})

				It("returns with a 200", func() {
					rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/metric_histories/a-metric-type", mcPort))
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					rsp.Body.Close()
				})
			})
			Context("when using streaming for metrics collection", func() {
				BeforeEach(func() {
					streamingCfg := cfg
					streamingCfg.Collector.CollectMethod = config.CollectMethodStreaming
					runner.configPath = writeConfig(&streamingCfg).Name()
					runner.Start()
				})

				AfterEach(func() {
					os.Remove(runner.configPath)
				})

				It("returns with a 200", func() {
					rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/metric_histories/a-metric-type", mcPort))
					Expect(err).NotTo(HaveOccurred())
					Expect(rsp.StatusCode).To(Equal(http.StatusOK))
					rsp.Body.Close()
				})
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

		Context("when metricscollector acquires the lock in first attempt", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("successfully acquired lock and started", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.lock-acquired-in-first-attempt"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
			})
		})

		Context("when metricscollector have the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
			})

			It("should retry acquiring lock to renew it's presence", func() {
				Eventually(runner.Session.Buffer, 8*time.Second).Should(gbytes.Say("metricscollector.retry-acquiring-lock"))

			})
		})

		Context("when interrupt occurs", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
			})

			It("successfully release lock and exit", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.successfully-released-lock"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.exited"))
			})
		})

		Context("When one instance of metricscollector owns lock and the other is waiting to get the lock", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("metricscollector.started"))

				secondRunner = NewMetricsCollectorRunner()
				consulConfig.Server.Port = 8000
				secondRunner.startCheck = ""
				secondRunner.configPath = writeConfig(&consulConfig).Name()
				secondRunner.Start()

			})

			AfterEach(func() {
				secondRunner.KillWithFire()
			})

			It("Competing instance should not get lock in first attempt", func() {
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("metricscollector.lock-acquired-in-first-attempt"))
				Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("metricscollector.successfully-acquired-lock"))
			})
		})

		Context("When more than one instances of metricscollector try to get the lock simultaneously", func() {

			var runnerAcquiredLock bool

			BeforeEach(func() {
				runner.Start()
				secondRunner = NewMetricsCollectorRunner()
				consulConfig.Server.Port = 8000
				secondRunner.startCheck = ""
				secondRunner.configPath = writeConfig(&consulConfig).Name()
				secondRunner.Start()
			})

			JustBeforeEach(func() {
				runnerAcquiredLock = true
				buffer := runner.Session.Out
				secondBuffer := secondRunner.Session.Out
				select {
				case <-buffer.Detect("metricscollector.lock-acquired-in-first-attempt"):
					runnerAcquiredLock = true
				case <-secondBuffer.Detect("metricscollector.lock-acquired-in-first-attempt"):
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
					Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
					Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("metricscollector.lock-acquired-in-first-attempt"))
					Consistently(secondRunner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("metricscollector.started"))
				} else {
					Eventually(secondRunner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
					Consistently(runner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("metricscollector.lock-acquired-in-first-attempt"))
					Consistently(runner.Session.Buffer, 5*time.Second).ShouldNot(gbytes.Say("metricscollector.started"))
				}
			})
		})

		Context("when the running metricscollector instance stopped", func() {
			BeforeEach(func() {
				runner.Start()
				Eventually(runner.Session.Buffer, 10*time.Second).Should(gbytes.Say("metricscollector.started"))
				secondRunner = NewMetricsCollectorRunner()
				consulConfig.Server.Port = 8000
				secondRunner.configPath = writeConfig(&consulConfig).Name()
				secondRunner.startCheck = ""
				secondRunner.Start()
				Consistently(secondRunner.Session.Buffer, 10*time.Second).ShouldNot(gbytes.Say("metricscollector.lock-acquired-in-first-attempt"))
			})

			AfterEach(func() {
				secondRunner.ClearLockDatabase()
				secondRunner.KillWithFire()
			})

			It("competing metricscollector instance should acquire the lock", func() {
				runner.Interrupt()
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("metricscollector.received-interrupt-signal"))
				Eventually(runner.Session.Buffer, 5*time.Second).Should(gbytes.Say("metricscollector.successfully-released-lock"))
				Eventually(secondRunner.Session.Buffer, 10*time.Second).Should(gbytes.Say("metricscollector.successfully-acquired-lock"))
				Eventually(secondRunner.Session.Buffer, 15*time.Second).Should(gbytes.Say("metricscollector.started"))
			})
		})

		Context("when the metricscollector acquires the lock and consul configuration is provided", func() {
			JustBeforeEach(func() {
				consulConfig = cfg
				consulConfig.EnableDBLock = true
				runner.configPath = writeConfig(&consulConfig).Name()
				runner.Start()

			})

			It("should start", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.collector.collector-started"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
				Consistently(runner.Session).ShouldNot(Exit())
			})
		})
	})

})
