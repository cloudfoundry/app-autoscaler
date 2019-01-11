package integration

import (
	"autoscaler/cf"
	"autoscaler/models"
	as_testhelpers "autoscaler/testhelpers"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration_Metricsgateway_Metricserver_Eventgenerator_Scalingengine", func() {
	var (
		testAppId         string
		timeout           time.Duration = 2 * time.Duration(breachDurationSecs) * time.Second
		initInstanceCount int           = 2
		fakeRLPServer     *as_testhelpers.FakeEventProducer
	)
	BeforeEach(func() {
		testAppId = getRandomId()
		startFakeCCNOAAUAA(initInstanceCount)
		fakeRLPServer = startFakeRLPServer(testAppId, 50, 100)
	})

	JustBeforeEach(func() {
		metricsServerConfPath = components.PrepareMetricsServerConfig(dbUrl, defaultHttpClientTimeout, components.Ports[MetricsServerHTTP], components.Ports[MetricsServerWS], tmpDir)
		metricsGatewayConfPath = components.PrepareMetricsGatewayConfig(dbUrl, []string{fmt.Sprintf("wss://127.0.0.1:%d", components.Ports[MetricsServerWS])}, fakeRLPServer.GetAddr(), tmpDir)
		eventGeneratorConfPath = components.PrepareEventGeneratorConfig(dbUrl, components.Ports[EventGenerator], fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsServerHTTP]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), aggregatorExecuteInterval, policyPollerInterval, saveInterval, evaluationManagerInterval, defaultHttpClientTimeout, tmpDir)
		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), cf.GrantTypePassword, defaultHttpClientTimeout, tmpDir)

		startMetricsServer()
		startMetricsGateway()
		startEventGenerator()
		startScalingEngine()

	})

	AfterEach(func() {
		stopAll()
		stopFakeRLPServer(fakeRLPServer)
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

			It("should scale out", func() {
				Eventually(func() int {
					return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount+1)
				}, 2*timeout, 1*time.Second).Should(BeNumerically(">=", 1))
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

			It("shouldn't scale out", func() {
				Consistently(func() int {
					return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount+1)
				}, timeout, 1*time.Second).Should(Equal(0))
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

			It("should scale in", func() {
				Eventually(func() int {
					return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount-1)
				}, timeout, 1*time.Second).Should(BeNumerically(">=", 1))
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
			It("shouldn't scale in", func() {
				Consistently(func() int {
					return getScalingHistoryCount(testAppId, initInstanceCount, initInstanceCount-1)
				}, timeout, 1*time.Second).Should(Equal(0))
			})

		})
	})

})
