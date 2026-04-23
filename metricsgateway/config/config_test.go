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

	Describe("LoadConfig off CF", func() {
		BeforeEach(func() {
			os.Unsetenv("VCAP_APPLICATION")
			vcapReader, err = configutil.NewVCAPConfigurationReader()
			Expect(err).ToNot(HaveOccurred())
		})

		It("loads default config with no file", func() {
			conf, err = config.LoadConfig("", vcapReader)
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.CFServer.Port).To(Equal(8080))
			Expect(conf.CFServer.TLS.CertFile).To(BeEmpty())
			Expect(conf.SyslogConfig.ServerAddress).To(Equal("log-cache.service.cf.internal"))
			Expect(conf.SyslogConfig.Port).To(Equal(6067))
		})
	})

	Describe("Validate", func() {
		BeforeEach(func() {
			vcapReader, err = configutil.NewVCAPConfigurationReader()
			Expect(err).ToNot(HaveOccurred())
			conf, err = config.LoadConfig("", vcapReader)
			Expect(err).ToNot(HaveOccurred())
			conf.ValidOrgGuids = []string{"some-org-guid"}
		})

		It("requires at least one org GUID", func() {
			conf.ValidOrgGuids = []string{}
			err = conf.Validate()
			Expect(err).To(MatchError("configuration error: valid_org_guids must contain at least one org GUID"))
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

		It("passes validation with defaults and org guids set", func() {
			err = conf.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("accepts multiple org GUIDs", func() {
			conf.ValidOrgGuids = []string{"org-1", "org-2", "org-3"}
			err = conf.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
