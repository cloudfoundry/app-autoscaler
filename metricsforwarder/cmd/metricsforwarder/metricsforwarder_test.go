package main_test

import (
	"autoscaler/models"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metricsforwarder", func() {
	var (
		runner *MetricsForwarderRunner
	)

	BeforeEach(func() {
		runner = NewMetricsForwarderRunner()
	})

	Describe("MetricsForwarder configuration check", func() {

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
				badfile, err := ioutil.TempFile("", "bad-mf-config")
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

				missingConfig.Db.PolicyDb.URL = ""

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

	Describe("MetricsForwarder REST API", func() {
		Context("when a request with custom metrics comes", func() {
			Context("when using polling for metrics collection", func() {
				BeforeEach(func() {
					runner.Start()

					customMetrics := []*models.CustomMetric{
						&models.CustomMetric{
							Name: "custom", Value: 12, Unit: "unit", InstanceIndex: 1, AppGUID: "an-app-id",
						},
					}
					body, err = json.Marshal(models.MetricsConsumer{InstanceIndex: 0, CustomMetrics: customMetrics})
					Expect(err).NotTo(HaveOccurred())

					req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/v1/an-app-id/metrics", cfg.Server.Port), bytes.NewReader(body))
					Expect(err).NotTo(HaveOccurred())

					base64EncodedUsernamePassword := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
					req.Header.Add("Content-Type", "application/json")
					req.Header.Add("Authorization", fmt.Sprintf("Basic %s", base64EncodedUsernamePassword))
				})

				AfterEach(func() {
					runner.KillWithFire()
				})

				It("returns with a 200", func() {
					res, err := httpClient.Do(req)
					Expect(err).NotTo(HaveOccurred())
					Expect(res.StatusCode).To(Equal(http.StatusOK))
					res.Body.Close()
				})
			})

		})

	})
})
