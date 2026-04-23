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
		metricForwarder   forwarder.MetricForwarder
		loggregatorConfig config.LoggregatorConfig
		syslogConfig      config.SyslogConfig
		gatewayConfig     config.MetricsGatewayConfig
		err               error
		conf              *config.Config
	)

	JustBeforeEach(func() {
		conf = &config.Config{
			LoggregatorConfig: loggregatorConfig,
			SyslogConfig:      syslogConfig,
			MetricsGateway:    gatewayConfig,
		}

		logger := lager.NewLogger("metricsforwarder-test")
		metricForwarder, err = forwarder.NewMetricForwarder(logger, conf)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewMetricForwarder", func() {
		When("gateway config is present it creates a GatewayEmitter", func() {
			BeforeEach(func() {
				syslogConfig = config.SyslogConfig{}
				loggregatorConfig = config.LoggregatorConfig{}
				gatewayConfig = config.MetricsGatewayConfig{
					URL: "https://some-gateway-url",
				}
			})

			It("should create a GatewayEmitter", func() {
				Expect(metricForwarder).To(BeAssignableToTypeOf(&forwarder.GatewayEmitter{}))
			})
		})

		When("syslog it is present it creates a SyslogEmitter", func() {
			BeforeEach(func() {
				loggregatorConfig.MetronAddress = "127.0.0.1:3458"
				gatewayConfig = config.MetricsGatewayConfig{}
				syslogConfig = config.SyslogConfig{
					ServerAddress: "syslog://some-server-address",
					Port:          514,
				}
			})

			It("should create a SyslogClient", func() {
				Expect(metricForwarder).To(BeAssignableToTypeOf(&forwarder.SyslogEmitter{}))
			})
		})

		When("loggregatorConfig is present creates a metronForwarder", func() {
			BeforeEach(func() {
				testCertDir := "../../test-certs"
				syslogConfig = config.SyslogConfig{}
				gatewayConfig = config.MetricsGatewayConfig{}
				loggregatorConfig = config.LoggregatorConfig{
					MetronAddress: "some-address",
					TLS: models.TLSCerts{
						KeyFile:    filepath.Join(testCertDir, "metron.key"),
						CertFile:   filepath.Join(testCertDir, "metron.crt"),
						CACertFile: filepath.Join(testCertDir, "loggregator-ca.crt"),
					},
				}
			})

			It("should create a metronClient", func() {
				Expect(metricForwarder).To(BeAssignableToTypeOf(&forwarder.MetronEmitter{}))
			})
		})
	})
})
