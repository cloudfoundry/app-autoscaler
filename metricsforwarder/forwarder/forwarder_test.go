package forwarder_test

import (
	"autoscaler/helpers"
	"autoscaler/metricsforwarder/config"
	"autoscaler/metricsforwarder/forwarder"
	"time"

	"autoscaler/metricsforwarder/testhelpers"

	"autoscaler/models"
	"errors"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
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
			CACertFile:     filepath.Join(testCertDir, "loggregator-ca.crt"),
			ClientCertFile: filepath.Join(testCertDir, "metron.crt"),
			ClientKeyFile:  filepath.Join(testCertDir, "metron.key"),
		}
		serverConfig := config.ServerConfig{
			Port: 10000 + GinkgoParallelNode(),
		}

		loggerConfig := helpers.LoggingConfig{
			Level: "debug",
		}

		conf := &config.Config{
			Server:            serverConfig,
			MetronAddress:     grpcIngressTestServer.GetAddr(),
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
