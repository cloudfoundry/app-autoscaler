package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	rpc "code.cloudfoundry.org/go-log-cache/v3/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integration_Eventgenerator_Scalingengine", func() {
	var (
		testAppId         string
		timeout           = 2 * time.Duration(breachDurationSecs) * time.Second
		initInstanceCount = 2
		tmpDir            string
		err               error
	)

	BeforeEach(func() {
		tmpDir, err = os.MkdirTemp("", "autoscaler")
		Expect(err).NotTo(HaveOccurred())

		testAppId = getRandomIdRef("testAppId")
		startFakeCCNOAAUAA(initInstanceCount)
		startMockLogCache()
	})

	JustBeforeEach(func() {
		eventGeneratorConfPath := components.PrepareEventGeneratorConfig(dbUrl, components.Ports[EventGenerator], mockLogCache.URL(), fmt.Sprintf("https://127.0.0.1:%d", components.Ports[ScalingEngine]), aggregatorExecuteInterval, policyPollerInterval, saveInterval, evaluationManagerInterval, defaultHttpClientTimeout, tmpDir)
		scalingEngineConfPath = components.PrepareScalingEngineConfig(dbUrl, components.Ports[ScalingEngine], fakeCCNOAAUAA.URL(), defaultHttpClientTimeout, tmpDir)

		startEventGenerator(eventGeneratorConfPath)
		startScalingEngine()
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
		stopEventGenerator()
		stopScalingEngine()
		stopMockLogCache()
	})
	Context("MemoryUtil", func() {
		BeforeEach(func() {
			mockLogCacheReturnsReadEnvelopes(testAppId, createContainerEnvelope(testAppId, 1, 4.0, float64(50), float64(2048000000), float64(100)))
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
			mockLogCacheReturnsReadEnvelopes(testAppId, createContainerEnvelope(testAppId, 1, 4.0, float64(50*1024*1024), float64(2048000000), float64(100*1024*1024)))
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
			mockLogCacheReturnsQueryResult(testAppId, 40)
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
			mockLogCacheReturnsQueryResult(testAppId, 40)
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
			mockLogCacheReturnsReadEnvelopes(testAppId, createCustomEnvelope(testAppId, "queuelength", "number", 50))
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

func mockLogCacheReturnsReadEnvelopes(sourceId string, envelopes []*loggregator_v2.Envelope) {
	mockLogCache.ReadReturns(sourceId, &rpc.ReadResponse{
		Envelopes: &loggregator_v2.EnvelopeBatch{
			Batch: envelopes,
		},
	}, nil)
}

func mockLogCacheReturnsQueryResult(sourceId string, value float64) {
	mockLogCache.InstantQueryReturns(sourceId, &rpc.PromQL_InstantQueryResult{
		Result: &rpc.PromQL_InstantQueryResult_Vector{
			Vector: &rpc.PromQL_Vector{
				Samples: []*rpc.PromQL_Sample{
					{
						Metric: map[string]string{
							"instance_id": "0",
						},
						Point: &rpc.PromQL_Point{
							Value: value,
						},
					},
				},
			},
		},
	}, nil)
}

func createContainerEnvelope(appId string, instanceIndex int32, cpuPercentage float64, memoryBytes float64, diskByte float64, memQuota float64) []*loggregator_v2.Envelope {
	return []*loggregator_v2.Envelope{
		{
			SourceId: appId,
			Message: &loggregator_v2.Envelope_Gauge{
				Gauge: &loggregator_v2.Gauge{
					Metrics: map[string]*loggregator_v2.GaugeValue{
						"cpu": {
							Unit:  "percentage",
							Value: cpuPercentage,
						},
						"disk": {
							Unit:  "bytes",
							Value: diskByte,
						},
						"memory": {
							Unit:  "bytes",
							Value: memoryBytes,
						},
						"memory_quota": {
							Unit:  "bytes",
							Value: memQuota,
						},
					},
				},
			},
		},
	}
}

func createCustomEnvelope(appId string, name string, unit string, value float64) []*loggregator_v2.Envelope {
	return []*loggregator_v2.Envelope{
		{
			SourceId: appId,
			DeprecatedTags: map[string]*loggregator_v2.Value{
				"origin": {
					Data: &loggregator_v2.Value_Text{
						Text: "autoscaler_metrics_forwarder",
					},
				},
			},
			Message: &loggregator_v2.Envelope_Gauge{
				Gauge: &loggregator_v2.Gauge{
					Metrics: map[string]*loggregator_v2.GaugeValue{
						name: {
							Unit:  unit,
							Value: value,
						},
					},
				},
			},
		},
	}
}
