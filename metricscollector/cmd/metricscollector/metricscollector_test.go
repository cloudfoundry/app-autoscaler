package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"autoscaler/cf"
	"autoscaler/metricscollector/config"
)

var _ = Describe("MetricsCollector", func() {
	var (
		runner *MetricsCollectorRunner
	)

	BeforeEach(func() {
		runner = NewMetricsCollectorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
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
				missingConfig.CF = cf.CFConfig{
					API: ccNOAAUAA.URL(),
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
	})

	Describe("when interrupt is sent", func() {
		BeforeEach(func() {
			runner.Start()
		})

		It("should stop", func() {
			runner.Session.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
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

})
