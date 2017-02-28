package main_test

import (
	"autoscaler/eventgenerator"
	"autoscaler/eventgenerator/config"
	"io/ioutil"
	"os"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/locket"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("Eventgenerator", func() {
	var runner *EventGeneratorRunner

	BeforeEach(func() {
		consulRunner.Reset()
		runner = NewEventGeneratorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

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
			Eventually(func() bool { return len(scalingEngine.ReceivedRequests()) >= 1 }, 5*time.Second).Should(BeTrue())
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
			Eventually(runner.Session.Buffer, 4*time.Second).Should(gbytes.Say("exited-with-failure"))
			Eventually(runner.Session).Should(Exit(1))
		})
	})

	Context("when the eventgenerator initially does not have the lock", func() {
		var competingEventGeneratorProcess ifrit.Process

		BeforeEach(func() {
			consulClient := consulRunner.NewClient()
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
				Eventually(func() bool { return len(scalingEngine.ReceivedRequests()) >= 1 }, 5*time.Second).Should(BeTrue())
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
				Server: config.ServerConfig{
					Port: config.DefaultServerPort + 1,
				},
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
