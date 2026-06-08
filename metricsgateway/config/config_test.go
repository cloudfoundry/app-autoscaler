package config_test

import (
	"os"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
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

	Describe("LoadConfig on CF", func() {
		var mockVCAPConfigurationReader *fakes.FakeVCAPConfigurationReader

		BeforeEach(func() {
			mockVCAPConfigurationReader = &fakes.FakeVCAPConfigurationReader{}
			mockVCAPConfigurationReader.IsRunningOnCFReturns(true)
			mockVCAPConfigurationReader.GetPortReturns(9090)
		})

		Context("when valid_org_guid is set in metricsgateway-config service binding", func() {
			BeforeEach(func() {
				mockVCAPConfigurationReader.GetServiceCredentialContentReturns([]byte(`
cf_server:
  xfcc:
    valid_org_guid: "explicit-org-guid-from-config"
`), nil)
			})

			It("uses the org GUID from the service binding config", func() {
				conf, err = config.LoadConfig("", mockVCAPConfigurationReader)
				Expect(err).ToNot(HaveOccurred())
				Expect(conf.CFServer.XFCC.ValidOrgGuid).To(Equal("explicit-org-guid-from-config"))
			})

			It("does not call GetOrgGuid", func() {
				_, err = config.LoadConfig("", mockVCAPConfigurationReader)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockVCAPConfigurationReader.GetOrgGuidCallCount()).To(Equal(0))
			})
		})

		Context("when valid_org_guid is not set in metricsgateway-config service binding", func() {
			BeforeEach(func() {
				mockVCAPConfigurationReader.GetServiceCredentialContentReturns([]byte(`{}`), nil)
				mockVCAPConfigurationReader.GetOrgGuidReturns("deployment-org-guid")
			})

			It("falls back to the deployment org GUID", func() {
				conf, err = config.LoadConfig("", mockVCAPConfigurationReader)
				Expect(err).ToNot(HaveOccurred())
				Expect(conf.CFServer.XFCC.ValidOrgGuid).To(Equal("deployment-org-guid"))
			})
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
