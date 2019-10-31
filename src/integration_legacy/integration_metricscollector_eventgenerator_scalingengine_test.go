package integration_legacy

import (
	"autoscaler/models"
	"encoding/json"
	"fmt"
	"time"

	"autoscaler/metricscollector/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration_Metricscollector_Eventgenerator_Scalingengine", func() {

	var (
		testAppId         string
		timeout           time.Duration = 2 * time.Duration(breachDurationSecs) * time.Second
		initInstanceCount int           = 2
		collectMethod     string        = config.CollectMethodPolling
	)
	BeforeEach(func() {
		testAppId = getRandomId()
		startFakeCCNOAAUAA(initInstanceCount)
	})

	JustBeforeEach(func() {
		metricsCollectorConfPath = components.PrepareMetricsCollectorConfig(dbUrl, components.Ports[MetricsCollector], fakeCCNOAAUAA.URL(), collectInterval,
			refreshInterval, saveInterval, collectMethod, defaultHttpClientTimeout, tmpDir)
		eventGeneratorConfPath = components.PrepareEventGeneratorConfig(dbUrl, components.Ports[EventGenerator], fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), aggregatorExecuteInterval, policyPollerInterval, saveInterval, evaluationManagerInterval, defaultHttpClientTimeout, tmpDir)
		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), defaultHttpClientTimeout, tmpDir)
		startMetricsCollector()
		startEventGenerator()
		startScalingEngine()

	})

	AfterEach(func() {
		stopAll()
	})

	Describe("Scale out", func() {
		Context("application's metrics break the scaling out rule for more than breach duration", func() {
			BeforeEach(func() {
				testPolicy := models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 5,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            models.MetricNameMemoryUtil,
							BreachDurationSeconds: breachDurationSecs,
							Threshold:             30,
							Operator:              ">=",
							CoolDownSeconds:       10,
							Adjustment:            "+1",
						},
					},
				}
				policyBytes, err := json.Marshal(testPolicy)
				Expect(err).NotTo(HaveOccurred())
				insertPolicy(testAppId, string(policyBytes), "1234")
			})
			Context("when using polling for metrics collection", func() {
				BeforeEach(func() {
					fakeMetricsPolling(testAppId, 102400000, 204800000)
					collectMethod = config.CollectMethodPolling
				})
				It("should scale out", func() {
					Eventually(func() int {
						return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount+1)
					}, timeout, 1*time.Second).Should(BeNumerically(">=", 1))
				})
			})

			Context("when using streaming for metrics collection", func() {
				BeforeEach(func() {
					fakeMetricsStreaming(testAppId, 102400000, 204800000)
					collectMethod = config.CollectMethodStreaming
				})
				AfterEach(func() {
					closeFakeMetricsStreaming()
				})
				It("should scale out", func() {
					Eventually(func() int {
						return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount+1)
					}, timeout, 1*time.Second).Should(BeNumerically(">=", 1))
				})
			})

		})
		Context("application's metrics do not break the scaling out rule", func() {
			BeforeEach(func() {
				testPolicy := models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 5,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            models.MetricNameMemoryUtil,
							BreachDurationSeconds: breachDurationSecs,
							Threshold:             80,
							Operator:              ">=",
							CoolDownSeconds:       10,
							Adjustment:            "+1",
						},
					},
				}
				policyBytes, err := json.Marshal(testPolicy)
				Expect(err).NotTo(HaveOccurred())
				insertPolicy(testAppId, string(policyBytes), "1234")
			})
			Context("when using polling for metrics collection", func() {
				BeforeEach(func() {
					fakeMetricsPolling(testAppId, 102400000, 204800000)
					collectMethod = config.CollectMethodPolling
				})
				It("shouldn't scale out", func() {
					Consistently(func() int {
						return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount+1)
					}, timeout, 1*time.Second).Should(Equal(0))
				})
			})

			Context("when using streaming for metrics collection", func() {
				BeforeEach(func() {
					fakeMetricsStreaming(testAppId, 102400000, 204800000)
					collectMethod = config.CollectMethodStreaming
				})
				AfterEach(func() {
					closeFakeMetricsStreaming()
				})
				It("shouldn't scale out", func() {
					Consistently(func() int {
						return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount+1)
					}, timeout, 1*time.Second).Should(Equal(0))
				})
			})

		})
	})
	Describe("Scale in", func() {
		Context("application's metrics break the scaling in rule for more than breach duration", func() {
			BeforeEach(func() {
				testPolicy := models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 5,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            models.MetricNameMemoryUtil,
							BreachDurationSeconds: breachDurationSecs,
							Threshold:             80,
							Operator:              "<",
							CoolDownSeconds:       10,
							Adjustment:            "-1",
						},
					},
				}
				policyBytes, err := json.Marshal(testPolicy)
				Expect(err).NotTo(HaveOccurred())
				insertPolicy(testAppId, string(policyBytes), "1234")
			})

			Context("when using polling for metrics collection", func() {
				BeforeEach(func() {
					fakeMetricsPolling(testAppId, 102400000, 204800000)
					collectMethod = config.CollectMethodPolling
				})
				It("should scale in", func() {
					Eventually(func() int {
						return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount-1)
					}, timeout, 1*time.Second).Should(BeNumerically(">=", 1))
				})
			})

			Context("when using streaming for metrics collection", func() {
				BeforeEach(func() {
					fakeMetricsStreaming(testAppId, 102400000, 204800000)
					collectMethod = config.CollectMethodStreaming
				})
				AfterEach(func() {
					closeFakeMetricsStreaming()
				})
				It("should scale in", func() {
					Eventually(func() int {
						return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount-1)
					}, timeout, 1*time.Second).Should(BeNumerically(">=", 1))
				})
			})

		})
		Context("application's metrics do not break the scaling in rule", func() {
			BeforeEach(func() {
				testPolicy := models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 5,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            models.MetricNameMemoryUtil,
							BreachDurationSeconds: breachDurationSecs,
							Threshold:             30,
							Operator:              "<",
							CoolDownSeconds:       30,
							Adjustment:            "-1",
						},
					},
				}
				policyBytes, err := json.Marshal(testPolicy)
				Expect(err).NotTo(HaveOccurred())
				insertPolicy(testAppId, string(policyBytes), "1234")
			})

			Context("when using polling for metrics collection", func() {
				BeforeEach(func() {
					fakeMetricsPolling(testAppId, 102400000, 204800000)
					collectMethod = config.CollectMethodPolling
				})
				It("shouldn't scale in", func() {
					Consistently(func() int {
						return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount-1)
					}, timeout, 1*time.Second).Should(Equal(0))
				})
			})

			Context("when using streaming for metrics collection", func() {
				BeforeEach(func() {
					fakeMetricsStreaming(testAppId, 102400000, 204800000)
					collectMethod = config.CollectMethodStreaming
				})
				AfterEach(func() {
					closeFakeMetricsStreaming()
				})
				It("shouldn't scale in", func() {
					Consistently(func() int {
						return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount-1)
					}, timeout, 1*time.Second).Should(Equal(0))
				})
			})

		})
	})

})
