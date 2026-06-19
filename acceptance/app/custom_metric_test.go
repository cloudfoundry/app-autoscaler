package app_test

import (
	"acceptance"
	"acceptance/config"
	. "acceptance/helpers"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AutoScaler custom metrics", func() {
	var (
		policy string
		err    error
	)
	BeforeEach(func() {

		appToScaleName = CreateTestApp(cfg, "go-custom-metric", 1)
		appToScaleGUID, err = GetAppGuid(cfg, appToScaleName)
		Expect(err).NotTo(HaveOccurred())

	})
	AfterEach(AppAfterEach)

	Describe("custom metrics policy for same app", func() {
		/*
			This test exercises scaling driven by custom metrics submitted via
			Basic-Auth. Going forward, custom metrics submission should be possible
			via mTLS only; this test can be removed in the future once
			basic-authentication support is fully retired.

			It is therefore only relevant when the autoscaler is deployed with
			basic_auth_for_custom_metrics="on". The "only_existing_bindings" and
			"off" modes are covered by unit tests:
			  - api/brokerserver/broker_handler_test.go
			  - metricsforwarder/server/auth/auth_test.go
			  - api/config/parser_internal_test.go
		*/
		JustBeforeEach(func() {
			instanceName = CreatePolicy(cfg, appToScaleName, appToScaleGUID, policy)
			StartApp(appToScaleName, cfg.CfPushTimeoutDuration())
		})
		Context("when scaling by custom metrics via basic-auth", func() {
			BeforeEach(func() {
				if !cfg.BasicAuthForCustomMetricsAllowedForFreshBinding() {
					Skip(fmt.Sprintf(
						"basic_auth_for_custom_metrics=%q: skipping Basic-Auth scaling test",
						cfg.BasicAuthForCustomMetrics))
				}
				credentialType := "binding-secret"
				policy = GeneratePolicyWithCredentialType(
					1, 2, "test_metric", 500, 500, &credentialType)
			})
			It("should scale out and scale in", Label(acceptance.LabelSmokeTests), func() {
				By("Scale out to 2 instances")
				scaleOut := sendMetricToAutoscaler(cfg, appToScaleGUID, appToScaleName, 550, false)
				Eventually(scaleOut).
					WithTimeout(5 * time.Minute).
					WithPolling(15 * time.Second).
					Should(Equal(2))

				By("Scale in to 1 instances")
				scaleIn := sendMetricToAutoscaler(cfg, appToScaleGUID, appToScaleName, 100, false)
				Eventually(scaleIn).
					WithTimeout(5 * time.Minute).
					WithPolling(15 * time.Second).
					Should(Equal(1))
			})
		})

		Context("when scaling by custom metrics via mtls", func() {
			BeforeEach(func() {
				policy = GenerateDynamicScaleOutAndInPolicy(1, 2, "test_metric", 500, 500)
			})
			It("should scale out and scale in", Label(acceptance.LabelSmokeTests), func() {
				By("Scale out to 2 instances")
				scaleOut := sendMetricToAutoscaler(cfg, appToScaleGUID, appToScaleName, 550, true)
				Eventually(scaleOut).
					WithTimeout(5 * time.Minute).
					WithPolling(15 * time.Second).
					Should(Equal(2))

				By("Scale in to 1 instance")
				scaleIn := sendMetricToAutoscaler(cfg, appToScaleGUID, appToScaleName, 100, true)
				Eventually(scaleIn).
					WithTimeout(5 * time.Minute).
					WithPolling(15 * time.Second).
					Should(Equal(1))
			})
		})
	})

	Describe("Custom metrics with producer app", func() {
		BeforeEach(func() {
			// attach policy to appToScale B
			policy = GenerateBindingsWithScalingPolicy("bound_app", 1, 2, "test_metric", 100, 500)
			instanceName = CreatePolicy(cfg, appToScaleName, appToScaleGUID, policy)
			StartApp(appToScaleName, cfg.CfPushTimeoutDuration())

			// push producer app without policy
			metricProducerAppName = CreateTestApp(cfg, "go-custom_metric_producer-app", 1)
			metricProducerAppGUID, err = GetAppGuid(cfg, metricProducerAppName)
			Expect(err).NotTo(HaveOccurred())
			err := BindServiceToAppWithPolicy(cfg, metricProducerAppName, instanceName, "")
			Expect(err).NotTo(HaveOccurred())
			StartApp(metricProducerAppName, cfg.CfPushTimeoutDuration())

		})
		Context("producer app A sends custom metrics for appToScale B via mtls", func() {
			When("policy is attached with the appToScale B with bound_app", func() {
				BeforeEach(func() {
					policy = GenerateBindingsWithScalingPolicy("bound_app", 1, 2, "test_metric", 100, 500)
				})
				It("should scale out and scale in app B", Label(acceptance.LabelSmokeTests), func() {
					By(fmt.Sprintf("Scale out %s to 2 instance", appToScaleName))
					scaleOut := sendMetricToAutoscaler(cfg, appToScaleGUID, metricProducerAppName, 550, true)
					Eventually(scaleOut).
						WithTimeout(5 * time.Minute).
						WithPolling(15 * time.Second).
						Should(Equal(2))

					By(fmt.Sprintf("Scale in %s to 1 instance", appToScaleName))
					scaleIn := sendMetricToAutoscaler(cfg, appToScaleGUID, metricProducerAppName, 80, true)
					Eventually(scaleIn).
						WithTimeout(5 * time.Minute).
						WithPolling(15 * time.Second).
						Should(Equal(1))
				})
			})
		})
		Context("appToScale B tries to send metrics for producer app A with strategy same_app", func() {
			BeforeEach(func() {
				policy = GenerateBindingsWithScalingPolicy("same_app", 1, 2, "test_metric", 100, 500)
			})
			It("should not scale producer app", func() {
				By(fmt.Sprintf("Fail Scale %s ", metricProducerAppName))
				sendMetricToAutoscaler(cfg, metricProducerAppGUID, appToScaleName, 550, true)
				WaitForNInstancesRunning(metricProducerAppGUID, 1, 5*time.Second, "expected 1 instance running")
			})
		})
	})
})

func sendMetricToAutoscaler(config *config.Config, appToScaleGUID string, metricProducerAppName string, metricThreshold int, mtls bool) func() (int, error) {
	return func() (int, error) {
		if mtls {
			SendMetricMTLS(config, appToScaleGUID, metricProducerAppName, metricThreshold, 0)
		} else {
			SendMetric(config, metricProducerAppName, metricThreshold)
		}
		return RunningInstances(appToScaleGUID, 5*time.Second)
	}
}
