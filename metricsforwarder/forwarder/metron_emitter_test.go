package forwarder_test

import (
	"errors"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/testhelpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricForwarder", func() {

	var (
		grpcIngressTestServer *testhelpers.TestIngressServer
		metrics               *models.CustomMetric
		err                   error
		serverConfig          helpers.ServerConfig
		loggerConfig          helpers.LoggingConfig
		emitter               forwarder.MetricForwarder
		conf                  *config.Config
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

		serverConfig = helpers.ServerConfig{
			Port: 10000 + GinkgoParallelProcess(),
		}

		loggerConfig = helpers.LoggingConfig{
			Level: "debug",
		}

		conf = &config.Config{
			Server:            serverConfig,
			Logging:           loggerConfig,
			LoggregatorConfig: loggregatorConfig,
		}
		logger := lager.NewLogger("metricsforwarder-test")
		emitter, err = forwarder.NewMetronEmitter(logger, conf)
		Expect(err).NotTo(HaveOccurred())

	})

	Describe("EmitMetrics", func() {
		It("Should emit gauge metrics", func() {
			metrics = &models.CustomMetric{Name: "queuelength", Value: 12.5, Unit: "unit", InstanceIndex: 123, AppGUID: "dummy-guid"}
			emitter.EmitMetric(metrics)
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
