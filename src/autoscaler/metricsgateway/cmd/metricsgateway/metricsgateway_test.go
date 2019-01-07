package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Metricsgateway", func() {
	var (
		runner *MetricsGatewayRunner
	)

	BeforeEach(func() {
		runner = NewMetricsGatewayRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("Metricscollector configuration check", func() {
		Context("with a valid config file", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("Starts successfully, retrives envelopes and emit envelopes", func() {
				Consistently(runner.Session, 5*time.Second).ShouldNot(Exit())
				Eventually(func() bool { return len(fakeMetricServer.ReceivedRequests()) >= 1 }, 5*time.Second).Should(BeTrue())
				Eventually(messageChan, 5*time.Second).Should(Receive())
			})
		})
		Context("with a invalid config file", func() {
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
		Context("with missing configuration", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				missingConfig := conf
				missingConfig.AppManager.PolicyDB.URL = ""
				runner.configPath = writeConfig(missingConfig).Name()
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

	Describe("when an interrupt is sent", func() {
		BeforeEach(func() {
			runner.Start()
		})

		It("should stop", func() {
			runner.Session.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})
	})
	Describe("when Health server is ready to serve RESTful API", func() {
		BeforeEach(func() {
			runner.Start()

		})
		Context("when a request to query health comes", func() {
			It("returns with a 200", func() {
				rsp, err := healthHttpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/health", healthport))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				raw, _ := ioutil.ReadAll(rsp.Body)
				healthData := string(raw)
				Expect(healthData).To(ContainSubstring("autoscaler_metricsgateway_concurrent_http_request"))
				Expect(healthData).To(ContainSubstring("autoscaler_metricsgateway_policyDB"))
				Expect(healthData).To(ContainSubstring("go_goroutines"))
				Expect(healthData).To(ContainSubstring("go_memstats_alloc_bytes"))
				rsp.Body.Close()

			})
		})
	})
})
