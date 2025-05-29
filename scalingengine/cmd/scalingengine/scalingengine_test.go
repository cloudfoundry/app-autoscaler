package main_test

import (
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"github.com/onsi/gomega/gbytes"

	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

var _ = Describe("Main", func() {
	var (
		runner *ScalingEngineRunner
		err    error

		healthURL *url.URL
		serverURL *url.URL

		cfServerURL *url.URL
	)

	BeforeEach(func() {
		runner = NewScalingEngineRunner()
		serverURL, err = url.Parse("https://127.0.0.1:" + strconv.Itoa(conf.Server.Port))
		Expect(err).ToNot(HaveOccurred())

		healthURL, err = url.Parse("http://127.0.0.1:" + strconv.Itoa(conf.Health.ServerConfig.Port))
		Expect(err).ToNot(HaveOccurred())

		cfServerURL, err = url.Parse(fmt.Sprintf("http://127.0.0.1:%d", conf.CFServer.Port))
		Expect(err).ToNot(HaveOccurred())
	})

	JustBeforeEach(func() {
		runner.Start()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Describe("With incorrect config", func() {
		Context("with a missing config file", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				runner.configPath = "bogus"
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(gbytes.Say("failed to open config file"))
			})
		})

		Context("with an invalid config file", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				badfile, err := os.CreateTemp("", "bad-engine-config")
				Expect(err).NotTo(HaveOccurred())
				runner.configPath = badfile.Name()
				// #nosec G306
				err = os.WriteFile(runner.configPath, []byte("bogus"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				_ = os.Remove(runner.configPath)
			})

			It("fails with an error", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(gbytes.Say("failed to read config file"))
			})
		})

		Context("with missing configuration", func() {
			BeforeEach(func() {
				runner.startCheck = ""
				missingParamConf := conf
				missingParamConf.CF = cf.Config{
					API: ccUAA.URL(),
				}

				missingParamConf.Server.Port = 7000 + GinkgoParallelProcess()
				missingParamConf.Logging.Level = "debug"

				cfg := writeConfig(&missingParamConf)
				runner.configPath = cfg.Name()
			})

			AfterEach(func() {
				os.Remove(runner.configPath)
			})

			It("should fail validation", func() {
				Eventually(runner.Session).Should(Exit(1))
				Expect(runner.Session.Buffer()).To(gbytes.Say("failed to validate configuration"))
			})
		})
	})

	Describe("when http server is ready to serve RESTful API", func() {
		When("a request to trigger scaling comes", func() {
			It("returns with a 200", func() {
				body, err := json.Marshal(models.Trigger{Adjustment: "+1"})
				Expect(err).NotTo(HaveOccurred())

				serverURL.Path = fmt.Sprintf("/v1/apps/%s/scale", appId)
				rsp, err := httpClient.Post(serverURL.String(), "application/json", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())

				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

		When("a request to retrieve scaling history comes", func() {
			It("returns with a 200", func() {
				serverURL.Path = fmt.Sprintf("/v1/apps/%s/scaling_histories", appId)
				req, err := http.NewRequest(http.MethodGet, serverURL.String(), nil)
				Expect(err).NotTo(HaveOccurred())
				rsp, err := httpClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})
		})

		It("handles the start and end of a schedule", func() {
			By("start of a schedule")
			serverURL.Path = fmt.Sprintf("/v1/apps/%s/active_schedules/111111", appId)

			bodyReader := bytes.NewReader([]byte(`{"instance_min_count":1, "instance_max_count":5, "initial_min_instance_count":3}`))

			req, err := http.NewRequest(http.MethodPut, serverURL.String(), bodyReader)
			Expect(err).NotTo(HaveOccurred())

			rsp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			rsp.Body.Close()

			By("end of a schedule")
			req, err = http.NewRequest(http.MethodDelete, serverURL.String(), nil)
			Expect(err).NotTo(HaveOccurred())

			rsp, err = httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			rsp.Body.Close()
		})
	})

	Describe("when Health server is ready to serve RESTful API", func() {
		BeforeEach(func() {
			basicAuthConfig := conf
			basicAuthConfig.Health.BasicAuth.Username = ""
			basicAuthConfig.Health.BasicAuth.Password = ""
			runner.configPath = writeConfig(&basicAuthConfig).Name()
		})

		When("a request to query health comes", func() {
			It("returns with a 200", func() {
				CheckHealthResponse(httpClient, healthURL.String(), []string{
					"autoscaler_scalingengine_concurrent_http_request", "autoscaler_scalingengine_schedulerDB",
					"autoscaler_scalingengine_policyDB", "autoscaler_scalingengine_scalingengineDB",
					"go_goroutines", "go_memstats_alloc_bytes",
				})

			})
		})
	})

	Describe("when Health server is ready to serve RESTful API with basic Auth", func() {
		BeforeEach(func() {
			healthURL.Path = "/health"
		})

		When("username and password are incorrect for basic authentication during health check", func() {
			It("should return 401", func() {
				CheckHealthAuth(GinkgoT(), httpClient, healthURL.String(), "wrongusername", "wrongpassword", http.StatusUnauthorized)
			})
		})

		When("username and password are correct for basic authentication during health check", func() {
			It("should return 200", func() {
				CheckHealthAuth(GinkgoT(), httpClient, healthURL.String(), conf.Health.BasicAuth.Username, conf.Health.BasicAuth.Password, http.StatusOK)
			})
		})
	})

	When("running CF server", func() {
		Describe("GET /v1/liveness", func() {
			It("should return 200", func() {
				cfServerURL.Path = "/v1/liveness"

				req, err := http.NewRequest(http.MethodGet, cfServerURL.String(), nil)
				Expect(err).NotTo(HaveOccurred())

				err = SetXFCCCertHeader(req, conf.CFServer.XFCC.ValidOrgGuid, conf.CFServer.XFCC.ValidSpaceGuid)
				Expect(err).NotTo(HaveOccurred())

				rsp, err := healthHttpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())

				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})
})
