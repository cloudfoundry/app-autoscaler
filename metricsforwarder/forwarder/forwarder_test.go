package forwarder_test

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/testhelpers"

	"errors"
	"path/filepath"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
)

var _ = Describe("MetricForwarder", func() {

	var (
		metricForwarder       forwarder.MetricForwarder
		metrics               *models.CustomMetric
		grpcIngressTestServer *testhelpers.TestIngressServer
		err                   error
	)

	BeforeEach(func() {
		testCertDir := "../../../../test-certs"

		grpcIngressTestServer, err = testhelpers.NewTestIngressServer(
			filepath.Join(testCertDir, "metron.crt"),
			filepath.Join(testCertDir, "metron.key"),
			filepath.Join(testCertDir, "loggregator-ca.crt"),
		)

		Expect(err).ToNot(HaveOccurred())
		err = grpcIngressTestServer.Start()
		Expect(err).NotTo(HaveOccurred())
		loggregatorConfig := config.LoggregatorConfig{
			MetronAddress: grpcIngressTestServer.GetAddr(),
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metron.key"),
				CertFile:   filepath.Join(testCertDir, "metron.crt"),
				CACertFile: filepath.Join(testCertDir, "loggregator-ca.crt"),
			},
		}
		serverConfig := config.ServerConfig{
			Port: 10000 + GinkgoParallelProcess(),
		}

		loggerConfig := helpers.LoggingConfig{
			Level: "debug",
		}

		conf := &config.Config{
			Server:            serverConfig,
			Logging:           loggerConfig,
			LoggregatorConfig: loggregatorConfig,
		}

		logger := lager.NewLogger("metricsforwarder-test")

		metricForwarder, err = forwarder.NewMetricForwarder(logger, conf)
		Expect(err).ToNot(HaveOccurred())

	})

	Describe("EmitMetrics", func() {

		Context("when a request to emit custom metrics comes", func() {

			BeforeEach(func() {
				metrics = &models.CustomMetric{Name: "queuelength", Value: 12.5, Unit: "unit", InstanceIndex: 123, AppGUID: "dummy-guid"}
				metricForwarder.EmitMetric(metrics)

			})

			It("Should emit gauge metrics", func() {
				env, err := getEnvelopeAt(grpcIngressTestServer.Receivers, 0)
				Expect(err).NotTo(HaveOccurred())
				ts := time.Unix(0, env.Timestamp)
				Expect(ts).Should(BeTemporally("~", time.Now(), time.Second))
				metrics := env.GetGauge()
				Expect(metrics).NotTo(BeNil())
				Expect(metrics.GetMetrics()).To(HaveLen(1))
				Expect(metrics.GetMetrics()["queuelength"].Value).To(Equal(12.5))
				Expect(env.Tags["origin"]).To(Equal("autoscaler_metrics_forwarder"))
			})

		})
	})

	AfterEach(func() {
		grpcIngressTestServer.Stop()
	})

})

func getEnvelopeAt(receivers chan loggregator_v2.Ingress_BatchSenderServer, idx int) (*loggregator_v2.Envelope, error) {
	var recv loggregator_v2.Ingress_BatchSenderServer
	Eventually(receivers, 10).Should(Receive(&recv))

	envBatch, err := recv.Recv()
	if err != nil {
		return nil, err
	}

	if len(envBatch.Batch) < 1 {
		return nil, errors.New("no envelopes")
	}

	return envBatch.Batch[idx], nil
}
