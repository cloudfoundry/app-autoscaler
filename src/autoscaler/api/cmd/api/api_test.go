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

var _ = Describe("Api", func() {
	var (
		runner *ApiRunner
		rsp    *http.Response
	)

	BeforeEach(func() {
		runner = NewApiRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("Api configuration check", func() {
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
				badfile, err := ioutil.TempFile("", "bad-ap-config")
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

				missingConfig.DB.PolicyDB.URL = ""
				missingConfig.DB.BindingDB.URL = ""
				missingConfig.BrokerUsername = ""
				missingConfig.BrokerPassword = ""
				missingConfig.BrokerServer.Port = 7000 + GinkgoParallelNode()
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

	Describe("BuildIn Mode", func() {
		Context("BuildIn Mode is false", func() {
			BeforeEach(func() {
				basicAuthConfig := cfg
				basicAuthConfig.UseBuildInMode = false
				runner.startCheck = ""
				runner.Start()
			})
			It("should start both broker and public-api", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("api.broker_http_server.broker-http-server-created"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("api.public_api_http_server.public-api-http-server-created"))
			})
		})

		Context("BuildIn Mode is true", func() {
			BeforeEach(func() {
				basicAuthConfig := cfg
				basicAuthConfig.UseBuildInMode = true
				runner.startCheck = ""
				runner.Start()
			})
			It("should start not start broker ", func() {
				Eventually(runner.Session.Buffer, 2*time.Second).ShouldNot(Say("api.broker_http_server.broker-http-server-created"))
				Eventually(runner.Session.Buffer, 2*time.Second).Should(Say("api.public_api_http_server.public-api-http-server-created"))
			})
		})
	})

	Describe("Broker Rest API", func() {
		Context("When a request comes to broker catalog", func() {
			BeforeEach(func() {
				runner.Start()
			})
			It("succeeds with a 200", func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://127.0.0.1:%d/v2/catalog", brokerPort), nil)
				Expect(err).NotTo(HaveOccurred())

				req.SetBasicAuth(username, password)

				rsp, err = brokerHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))

				bodyBytes, err := ioutil.ReadAll(rsp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(bodyBytes).To(Equal(catalogBytes))
			})
		})
	})

	Describe("Pubic API", func() {
		Context("When a request comes to public api info", func() {
			BeforeEach(func() {
				runner.Start()
			})
			It("succeeds with a 200", func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://127.0.0.1:%d/v1/info", publicApiPort), nil)
				Expect(err).NotTo(HaveOccurred())

				rsp, err = apiHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))

				bodyBytes, err := ioutil.ReadAll(rsp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(bodyBytes).To(Equal(infoBytes))
			})
		})
	})
	Describe("when Health server is ready to serve RESTful API", func() {
		BeforeEach(func() {
			basicAuthConfig := cfg
			basicAuthConfig.Health.HealthCheckUsername = ""
			basicAuthConfig.Health.HealthCheckPassword = ""
			runner.configPath = writeConfig(&basicAuthConfig).Name()
			runner.Start()
		})
		Context("when a request to query health comes", func() {
			It("returns with a 200", func() {
				rsp, err := healthHttpClient.Get(fmt.Sprintf("http://127.0.0.1:%d", healthport))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				raw, _ := ioutil.ReadAll(rsp.Body)
				healthData := string(raw)
				Expect(healthData).To(ContainSubstring("autoscaler_golangapiserver_concurrent_http_request"))
				Expect(healthData).To(ContainSubstring("autoscaler_golangapiserver_policyDB"))
				Expect(healthData).To(ContainSubstring("autoscaler_golangapiserver_bindingDB"))
				Expect(healthData).To(ContainSubstring("go_goroutines"))
				Expect(healthData).To(ContainSubstring("go_memstats_alloc_bytes"))
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
		BeforeEach(func() {
			runner.Start()
		})
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

				req.SetBasicAuth(cfg.Health.HealthCheckUsername, cfg.Health.HealthCheckPassword)

				rsp, err := healthHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})

})
