package integration_test

import (
	"encoding/json"
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	as_testhelpers "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration_Metricsgateway_Metricserver_Eventgenerator_Scalingengine", func() {
	var (
		testAppId           string
		timeout             = 2 * time.Duration(breachDurationSecs) * time.Second
		initInstanceCount   = 2
		fakeRLPServer       *as_testhelpers.FakeEventProducer
		envelopes           []*loggregator_v2.Envelope
		fakeRLPEmitInterval = 500 * time.Millisecond
	)
	BeforeEach(func() {
		testAppId = getRandomId()
		startFakeCCNOAAUAA(initInstanceCount)
	})

	JustBeforeEach(func() {
		fakeRLPServer = startFakeRLPServer(testAppId, envelopes, fakeRLPEmitInterval)
		metricsServerConfPath = components.PrepareMetricsServerConfig(dbUrl, defaultHttpClientTimeout, components.Ports[MetricsServerHTTP], components.Ports[MetricsServerWS], tmpDir)
		metricsGatewayConfPath = components.PrepareMetricsGatewayConfig(dbUrl, []string{fmt.Sprintf("wss://127.0.0.1:%d", components.Ports[MetricsServerWS])}, fakeRLPServer.GetAddr(), tmpDir)
		eventGeneratorConfPath = components.PrepareEventGeneratorConfig(dbUrl, components.Ports[EventGenerator], fmt.Sprintf("https://127.0.0.1:%d", components.Ports[MetricsServerHTTP]), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), aggregatorExecuteInterval, policyPollerInterval, saveInterval, evaluationManagerInterval, defaultHttpClientTimeout, tmpDir)
		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), defaultHttpClientTimeout, tmpDir)

		startMetricsServer()
		startMetricsGateway()
		startEventGenerator()
		startScalingEngine()

	})

	AfterEach(func() {
		stopFakeRLPServer(fakeRLPServer)
		stopMetricsGateway()
		stopMetricsServer()
		stopEventGenerator()
		stopScalingEngine()
	})
	Context("MemoryUtil", func() {
		BeforeEach(func() {
			envelopes = createContainerEnvelope(testAppId, 1, 4.0, float64(50), float64(2048000000), float64(100))

		})
		Context("Scale out", func() {
			Context("application's responsetime break the scaling out rule for more than breach duration", func() {
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
			Context("application's memoryutil do not break the scaling out rule", func() {
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
		Context("Scale in", func() {
			Context("application's memoryutil break the scaling in rule for more than breach duration", func() {
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
			Context("application's memoryutil do not break the scaling in rule", func() {
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

	Context("MemoryUsed", func() {
		BeforeEach(func() {
			envelopes = createContainerEnvelope(testAppId, 1, 4.0, float64(50*1024*1024), float64(2048000000), float64(100*1024*1024))

		})
		Context("Scale out", func() {
			Context("application's memory break the scaling out rule for more than breach duration", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameMemoryUsed,
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
					}, timeout, 1*time.Second).Should(BeNumerically(">=", 1))
				})

			})
			Context("application's memory do not break the scaling out rule", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameMemoryUsed,
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
		Context("Scale in", func() {
			Context("application's memory break the scaling in rule for more than breach duration", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameMemoryUsed,
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
			Context("application's memory do not break the scaling in rule", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameMemoryUsed,
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

	Context("ResponseTime", func() {
		BeforeEach(func() {
			envelopes = createHTTPTimerEnvelope(testAppId, 1542325492000000000, 1542325492050000000)

		})
		Context("Scale out", func() {
			Context("application's responsetime break the scaling out rule for more than breach duration", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameResponseTime,
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
					}, timeout, 1*time.Second).Should(BeNumerically(">=", 1))
				})

			})
			Context("application's responsetime do not break the scaling out rule", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameResponseTime,
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
		Context("Scale in", func() {
			Context("application's responsetime break the scaling in rule for more than breach duration", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameResponseTime,
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
			Context("application's responsetime do not break the scaling in rule", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameResponseTime,
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

	Context("Throughput", func() {
		BeforeEach(func() {
			envelopes = createHTTPTimerEnvelope(testAppId, 1542325492000000000, 1542325492050000000)
			fakeRLPEmitInterval = 100 * time.Millisecond

		})
		Context("Scale out", func() {
			Context("application's throughput break the scaling out rule for more than breach duration", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameThroughput,
								BreachDurationSeconds: breachDurationSecs,
								Threshold:             3,
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
					}, timeout, 1*time.Second).Should(BeNumerically(">=", 1))
				})

			})
			Context("application's throughput do not break the scaling out rule", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameThroughput,
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
		Context("Scale in", func() {
			Context("application's throughput break the scaling in rule for more than breach duration", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameThroughput,
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
			Context("application's throughput do not break the scaling in rule", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            models.MetricNameThroughput,
								BreachDurationSeconds: breachDurationSecs,
								Threshold:             1,
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

	Context("CustomMetric", func() {
		BeforeEach(func() {
			envelopes = createCustomEnvelope(testAppId, "queuelength", "number", 50)

		})
		Context("Scale out", func() {
			Context("application's queuelength break the scaling out rule for more than breach duration", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            "queuelength",
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
					}, timeout, 1*time.Second).Should(BeNumerically(">=", 1))
				})

			})
			Context("application's queuelength do not break the scaling out rule", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            "queuelength",
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
		Context("Scale in", func() {
			Context("application's queuelength break the scaling in rule for more than breach duration", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            "queuelength",
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
			Context("application's queuelength do not break the scaling in rule", func() {
				BeforeEach(func() {
					testPolicy := models.ScalingPolicy{
						InstanceMin: 1,
						InstanceMax: 5,
						ScalingRules: []*models.ScalingRule{
							{
								MetricType:            "queuelength",
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

})
