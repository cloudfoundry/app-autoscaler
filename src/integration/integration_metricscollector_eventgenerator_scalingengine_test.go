package integration_test

import (
	. "integration"

	"autoscaler/cf"
	"autoscaler/models"
	"fmt"
	"time"

	"code.cloudfoundry.org/locket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration_Metricscollector_Eventgenerator_Scalingengine", func() {

	var (
		testAppId         string
		timeout           time.Duration = 50 * time.Second
		initInstanceCount int           = 2
	)
	BeforeEach(func() {
		testAppId = getRandomId()
		startFakeCCNOAAUAA(initInstanceCount)
		fakeMetrics(testAppId, 400*1024*1024)
		metricsCollectorConfPath = components.PrepareMetricsCollectorConfig(dbUrl, components.Ports[MetricsCollector], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, pollInterval,
			refreshInterval, tmpDir, locket.DefaultSessionTTL, locket.RetryInterval, consulRunner.ConsulCluster())
		eventGeneratorConfPath = components.PrepareEventGeneratorConfig(dbUrl, fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsCollector]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), aggregatorExecuteInterval, policyPollerInterval, evaluationManagerInterval, tmpDir, locket.DefaultSessionTTL, locket.RetryInterval, consulRunner.ConsulCluster())
		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, tmpDir, consulRunner.ConsulCluster())
		startMetricsCollector()
		startEventGenerator()
		startScalingEngine()
	})

	AfterEach(func() {
		stopAll()
	})
	Describe("Scale out", func() {
		Context("Application's metrics break the upper scaling rule for more than breach duration time", func() {
			BeforeEach(func() {
				testPolicy := models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 5,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            models.MetricNameMemory,
							StatWindowSeconds:     30,
							BreachDurationSeconds: 30,
							Threshold:             90,
							Operator:              ">=",
							CoolDownSeconds:       30,
							Adjustment:            "+1",
						},
					},
				}
				insertPolicy(testAppId, testPolicy)

			})
			It("should scale out", func() {
				Eventually(func() int {
					return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount+1)
				}, timeout, 15*time.Second, 1*time.Second).Should(BeNumerically(">=", 1))
			})

		})
		Context("Application's metrics do not break the upper scaling rule", func() {
			BeforeEach(func() {
				testPolicy := models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 5,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            models.MetricNameMemory,
							StatWindowSeconds:     30,
							BreachDurationSeconds: 30,
							Threshold:             900,
							Operator:              ">=",
							CoolDownSeconds:       30,
							Adjustment:            "+1",
						},
					},
				}
				insertPolicy(testAppId, testPolicy)
			})
			It("shouldn't scale out", func() {
				Consistently(func() int {
					return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount+1)
				}, timeout, 15*time.Second, 1*time.Second).Should(Equal(0))
			})

		})
	})
	Describe("Scale in", func() {
		Context("Application's metrics break the lower scaling rule for more than breach duration time", func() {
			BeforeEach(func() {
				testPolicy := models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 5,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            models.MetricNameMemory,
							StatWindowSeconds:     30,
							BreachDurationSeconds: 30,
							Threshold:             900,
							Operator:              "<",
							CoolDownSeconds:       300,
							Adjustment:            "-1",
						},
					},
				}
				insertPolicy(testAppId, testPolicy)
			})
			It("should scale in", func() {
				Eventually(func() int {
					return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount-1)
				}, timeout, 15*time.Second, 1*time.Second).Should(BeNumerically(">=", 1))
			})

		})
		Context("Application's metrics do not break the lower scaling rule", func() {
			BeforeEach(func() {
				testPolicy := models.ScalingPolicy{
					InstanceMin: 1,
					InstanceMax: 5,
					ScalingRules: []*models.ScalingRule{
						{
							MetricType:            models.MetricNameMemory,
							StatWindowSeconds:     30,
							BreachDurationSeconds: 30,
							Threshold:             90,
							Operator:              "<",
							CoolDownSeconds:       30,
							Adjustment:            "+1",
						},
					},
				}
				insertPolicy(testAppId, testPolicy)
			})
			It("shouldn't scale in", func() {
				Consistently(func() int {
					return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount-1)
				}, timeout, 15*time.Second, 1*time.Second).Should(Equal(0))
			})

		})
	})
})
