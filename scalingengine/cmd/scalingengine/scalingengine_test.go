package main_test

import (
	"autoscaler/cf"
	"autoscaler/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/onsi/gomega/gbytes"

	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

var _ = Describe("Main", func() {

	var (
		runner *ScalingEngineRunner
	)

	BeforeEach(func() {
		runner = NewScalingEngineRunner()
	})

	JustBeforeEach(func() {
		runner.Start()
	})

	AfterEach(func() {
		runner.Interrupt()
	})

	It("should start", func() {
		Consistently(runner.Session).ShouldNot(Exit())
	})

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
			badfile, err := ioutil.TempFile("", "bad-engine-config")
			Expect(err).NotTo(HaveOccurred())
			runner.configPath = badfile.Name()
			ioutil.WriteFile(runner.configPath, []byte("bogus"), os.ModePerm)
		})

		AfterEach(func() {
			os.Remove(runner.configPath)
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
			missingParamConf.Cf = cf.CfConfig{
				Api: ccUAA.URL(),
			}

			missingParamConf.Server.Port = 7000 + GinkgoParallelNode()
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

	Context("when an interrupt is sent", func() {
		It("should stop", func() {
			runner.Session.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(0))
		})
	})

	Describe("when a request to trigger scaling comes", func() {
		It("returns with a 200", func() {
			body, err := json.Marshal(models.Trigger{Adjustment: "+1"})
			Expect(err).NotTo(HaveOccurred())

			rsp, err := httpClient.Post(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/scale", port, appId),
				"application/json", bytes.NewReader(body))
			Expect(err).NotTo(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			rsp.Body.Close()
		})
	})

	Describe("when a request to retrieve scaling history comes", func() {
		It("returns with a 200", func() {
			rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/scaling_histories", port, appId))
			Expect(err).NotTo(HaveOccurred())
			Expect(rsp.StatusCode).To(Equal(http.StatusOK))
			rsp.Body.Close()
		})
	})

	It("handles the start and end of a schedule", func() {
		By("start of a schedule")
		url := fmt.Sprintf("https://127.0.0.1:%d/v1/apps/%s/active_schedules/111111", port, appId)
		bodyReader := bytes.NewReader([]byte(`{"instance_min_count":1, "instance_max_count":5, "initial_min_instance_count":3}`))

		req, err := http.NewRequest(http.MethodPut, url, bodyReader)
		Expect(err).NotTo(HaveOccurred())

		rsp, err := httpClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.StatusCode).To(Equal(http.StatusOK))
		rsp.Body.Close()

		By("end of a schedule")
		req, err = http.NewRequest(http.MethodDelete, url, nil)
		Expect(err).NotTo(HaveOccurred())

		rsp, err = httpClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(rsp.StatusCode).To(Equal(http.StatusNoContent))
		rsp.Body.Close()
	})
})
