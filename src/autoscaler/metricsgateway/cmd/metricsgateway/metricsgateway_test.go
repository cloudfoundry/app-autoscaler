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
	JustBeforeEach(func() {
		runner.Start()
	})
	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("Metricscollector configuration check", func() {
		Context("with a valid config file", func() {

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
				runner.configPath = writeConfig(&missingConfig).Name()
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
	Describe("when it fails to connect to metricsserver when starting", func() {
		BeforeEach(func() {
			wrongConfig := conf
			wrongConfig.MetricServerAddrs = []string{"wss://localhost:9999"}
			runner.configPath = writeConfig(&wrongConfig).Name()
			runner.startCheck = ""

		})
		It("fails to start", func() {
			Eventually(runner.Session, time.Duration(2*conf.Emitter.MaxSetupRetryCount)*conf.Emitter.RetryDelay).Should(Exit(1))
			Expect(runner.Session.Buffer()).To(Say("failed to start emitter"))
		})
	})
	Describe("when an interrupt is sent", func() {

		It("should stop", func() {
			runner.Session.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})
	})

	Describe("when Health server is ready to serve RESTful API without basic Auth", func() {
		BeforeEach(func() {
			basicAuthConfig := conf
			basicAuthConfig.Health.HealthCheckUsername = ""
			basicAuthConfig.Health.HealthCheckPassword = ""
			runner.configPath = writeConfig(&basicAuthConfig).Name()
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
				Expect(healthData).To(ContainSubstring("autoscaler_metricsgateway_envelope_number_from_rlp"))
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

				req.SetBasicAuth(conf.Health.HealthCheckUsername, conf.Health.HealthCheckPassword)

				rsp, err := healthHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		Context("when a request to query profile comes", func() {
			It("returns with a 200", func() {

				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/debug/pprof", healthport), nil)
				Expect(err).NotTo(HaveOccurred())

				req.SetBasicAuth(conf.Health.HealthCheckUsername, conf.Health.HealthCheckPassword)

				rsp, err := healthHttpClient.Do(req)
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
})
