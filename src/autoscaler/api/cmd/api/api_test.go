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

	Describe("Broker Rest API", func() {
		Context("When a request comes to broker catalog", func() {
			BeforeEach(func() {
				runner.Start()
			})
			It("succeeds with a 200", func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/sb/v2/catalog", apPort), nil)
				Expect(err).NotTo(HaveOccurred())

				req.SetBasicAuth(username, password)

				rsp, err = httpClient.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))

				bodyBytes, err := ioutil.ReadAll(rsp.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(bodyBytes).To(Equal(catalogBytes))
			})
		})
	})

})
