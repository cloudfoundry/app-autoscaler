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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"autoscaler/cf"
	"autoscaler/metricscollector"

	"github.com/onsi/gomega/gbytes"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("MetricsCollector", func() {
	var runner *MetricsCollectorRunner

	BeforeEach(func() {
		consulRunner.Reset()
		runner = NewMetricsCollectorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Context("when the metricscollector acquires the lock", func() {
		BeforeEach(func() {
			runner.startCheck = ""
			runner.Start()

			Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
		})

		It("should register itself as the active instance and start", func() {
			Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.registration-runner.succeeded-registering-service"))
			Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
			Consistently(runner.Session).ShouldNot(Exit())
		})
	})

	Context("when the metricscollector loses the lock", func() {
		BeforeEach(func() {
			runner.startCheck = ""
			runner.Start()

			Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
			Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.registration-runner.succeeded-registering-service"))
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
			consulClient := consulRunner.NewClient()
			logger := lagertest.NewTestLogger("competing-process")
			buffer := logger.Buffer()

			competingMetricsCollectorLock := locket.NewLock(logger, consulClient, metricscollector.MetricsCollectorLockSchemaPath(), []byte{}, clock.NewClock(), cfg.Lock.LockRetryInterval, cfg.Lock.LockTTL)
			competingMetricsCollectorProcess = ifrit.Invoke(competingMetricsCollectorLock)
			Eventually(buffer, 2*time.Second).Should(gbytes.Say("competing-process.lock.acquire-lock-succeeded"))

			runner.startCheck = ""
			runner.Start()
		})

		It("should not start", func() {
			Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.lock.acquiring-lock"))
			Consistently(runner.Session.Buffer, 2*time.Second).ShouldNot(gbytes.Say("metricscollector.started"))
		})

		Describe("when the lock becomes available", func() {
			BeforeEach(func() {
				ginkgomon.Kill(competingMetricsCollectorProcess)
			})

			It("acquires the lock and starts", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say(runner.acquiredLockCheck))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.registration-runner.succeeded-registering-service"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(gbytes.Say("metricscollector.started"))
				Consistently(runner.Session).ShouldNot(Exit())
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
			Expect(runner.Session.Buffer()).To(Say("failed to read config file"))
		})
	})

	Context("with missing configuration", func() {
		BeforeEach(func() {
			runner.startCheck = ""
			cfg.Cf = cf.CfConfig{
				Api: ccNOAAUAA.URL(),
			}

			cfg.Server.Port = 7000 + GinkgoParallelNode()
			cfg.Logging.Level = "debug"

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

	Context("when an interrupt is sent", func() {
		BeforeEach(func() {
			runner.Start()
		})

		It("should stop", func() {
			runner.Session.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})
	})

	Describe("when a request for memory metrics comes", func() {
		Context("when token is not expired", func() {
			BeforeEach(func() {
				eLock.Lock()
				isTokenExpired = false
				eLock.Unlock()
				runner.Start()
			})

			It("returns with a 200", func() {
				rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/metrics/memoryused", mcPort))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

		Context("when token is expired", func() {
			BeforeEach(func() {
				eLock.Lock()
				isTokenExpired = true
				eLock.Unlock()
				runner.Start()
			})
			It("refreshes the token and returns with a 200", func() {
				rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/metrics/memoryused", mcPort))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

	})

	Describe("when a request for memory metrics history comes", func() {
		BeforeEach(func() {
			runner.Start()
		})

		It("returns with a 200", func() {
			rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/metric_histories/memoryused", mcPort))
			Expect(err).NotTo(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			rsp.Body.Close()
		})
	})
})
