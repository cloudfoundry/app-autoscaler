package forwarder_test

import (
	"path/filepath"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager/v3"
)

var _ = Describe("MetricForwarder", func() {

	var (
		metricForwarder   forwarder.Emitter
		loggregatorConfig config.LoggregatorConfig
		err               error
		conf              *config.Config
	)

	JustBeforeEach(func() {
		conf = &config.Config{
			LoggregatorConfig: loggregatorConfig,
		}

		logger := lager.NewLogger("metricsforwarder-test")
		metricForwarder, err = forwarder.NewMetricForwarder(logger, conf)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewMetricForwarder", func() {
		Context("when loggregatorConfig is not present it creates a SyslogAgentForwarder", func() {

			BeforeEach(func() {
				loggregatorConfig.MetronAddress = ""
			})

			It("should create a SyslogAgentClient", func() {
				Expect(metricForwarder).To(BeAssignableToTypeOf(&forwarder.SyslogAgentForwarder{}))
			})
		})

		Context("when loggregatorConfig is present creates a metronAgentForwarder", func() {
			BeforeEach(func() {
				testCertDir := "../../../../test-certs"
				loggregatorConfig = config.LoggregatorConfig{
					MetronAddress: "some-address",
					TLS: models.TLSCerts{
						KeyFile:    filepath.Join(testCertDir, "metron.key"),
						CertFile:   filepath.Join(testCertDir, "metron.crt"),
						CACertFile: filepath.Join(testCertDir, "loggregator-ca.crt"),
					},
				}
			})

			It("should create a metronAgentClient", func() {
				Expect(metricForwarder).To(BeAssignableToTypeOf(&forwarder.MetronAgentEmitter{}))
			})
		})
	})

	Describe("EmitMetrics", func() {
		It("Should call emmit metric", func() {
		})
	})
})
