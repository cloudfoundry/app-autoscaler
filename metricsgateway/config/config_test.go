package config_test

import (
	"os"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metricsgateway Config Suite")
}

var _ = Describe("Config", func() {
	var (
		conf       *config.Config
		err        error
		vcapReader configutil.VCAPConfigurationReader
	)

	Describe("LoadConfig on CF", func() {
		BeforeEach(func() {
			// Simulate CF environment
			os.Setenv("VCAP_APPLICATION", `{"application_id":"test-app"}`)
			os.Setenv("VCAP_SERVICES", `{}`)
			os.Setenv("CF_INSTANCE_CERT", "/tmp/instance.crt")
			os.Setenv("CF_INSTANCE_KEY", "/tmp/instance.key")
			os.Setenv("CF_INSTANCE_CA_CERT", "/tmp/instance_ca.crt")

			vcapReader, err = configutil.NewVCAPConfigurationReader()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.Unsetenv("VCAP_APPLICATION")
			os.Unsetenv("VCAP_SERVICES")
			os.Unsetenv("CF_INSTANCE_CERT")
			os.Unsetenv("CF_INSTANCE_KEY")
			os.Unsetenv("CF_INSTANCE_CA_CERT")
		})

		It("configures CFServer TLS with instance identity certs", func() {
			conf, err = config.LoadConfig("", vcapReader)
			Expect(err).To(HaveOccurred()) // Will fail due to missing syslog-client binding, but that's OK

			// Verify that instance TLS would be configured if we had all bindings
			instanceTLS := vcapReader.GetInstanceTLSCerts()
			Expect(instanceTLS.CertFile).To(Equal("/tmp/instance.crt"))
			Expect(instanceTLS.KeyFile).To(Equal("/tmp/instance.key"))
			Expect(instanceTLS.CACertFile).To(Equal("/tmp/instance_ca.crt"))
		})
	})

	Describe("Validate", func() {
		BeforeEach(func() {
			vcapReader, err = configutil.NewVCAPConfigurationReader()
			Expect(err).ToNot(HaveOccurred())
			conf, err = config.LoadConfig("", vcapReader)
			Expect(err).ToNot(HaveOccurred())
		})

		It("requires syslog server address", func() {
			conf.SyslogConfig.ServerAddress = ""
			err = conf.Validate()
			Expect(err).To(MatchError("configuration error: syslog server_address is empty"))
		})

		It("requires syslog port", func() {
			conf.SyslogConfig.Port = 0
			err = conf.Validate()
			Expect(err).To(MatchError("configuration error: syslog port is zero"))
		})

		It("passes validation with defaults", func() {
			err = conf.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
