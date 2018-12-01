package main_test

import (
	"autoscaler/eventgenerator/config"
	"autoscaler/helpers"
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

var _ = Describe("Eventgenerator", func() {
	var (
		runner *EventGeneratorRunner
	)

	BeforeEach(func() {
		runner = NewEventGeneratorRunner()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	Context("with a valid config file", func() {
		BeforeEach(func() {
			runner.Start()
		})

		It("Starts successfully, retrives metrics and  generates events", func() {
			Consistently(runner.Session).ShouldNot(Exit())
			Eventually(func() bool { return len(metricCollector.ReceivedRequests()) >= 1 }, 5*time.Second).Should(BeTrue())
			Eventually(func() bool { return len(scalingEngine.ReceivedRequests()) >= 1 }, time.Duration(2*breachDurationSecs)*time.Second).Should(BeTrue())
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
				Logging: helpers.LoggingConfig{
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

	Describe("EventGenerator REST API", func() {
		Context("when a request for aggregated metrics history comes", func() {
			BeforeEach(func() {
				runner.Start()
			})

			It("returns with a 200", func() {
				rsp, err := httpClient.Get(fmt.Sprintf("https://127.0.0.1:%d/v1/apps/an-app-id/aggregated_metric_histories/a-metric-type", egPort))
				Expect(err).NotTo(HaveOccurred())
				Expect(rsp.StatusCode).To(Equal(http.StatusOK))
				rsp.Body.Close()
			})

		})

	})

})
