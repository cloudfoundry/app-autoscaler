package config_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var (
		conf       *Config
		err        error
		configFile string
		vcapReader *fakes.FakeVCAPConfigurationReader
	)

	BeforeEach(func() {
		vcapReader = &fakes.FakeVCAPConfigurationReader{}
		configFile = ""
	})

	JustBeforeEach(func() {
		conf, err = LoadConfig(configFile, vcapReader)
	})

	Context("with an empty path (defaults only)", func() {
		It("loads default ports and logging level", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.Server.Port).To(Equal(DefaultServerPort))
			Expect(conf.CFServer.Port).To(Equal(DefaultCFServerPort))
			Expect(conf.Health.ServerConfig.Port).To(Equal(DefaultHealthServerPort))
			Expect(conf.Logging.Level).To(Equal(DefaultLoggingLevel))
		})

		It("validates successfully", func() {
			Expect(conf.Validate()).To(Succeed())
		})
	})

	Context("loading the example config file", func() {
		BeforeEach(func() {
			configFile = "../exampleconfig/config.yml"
		})

		It("reads the routing_api url and scaling engine url", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.RoutingAPI.URL).To(Equal("https://api.cf.example.com"))
			Expect(conf.ScalingEngine.ScalingEngineURL).To(Equal("https://scalingengine.cf.example.com"))
			Expect(conf.RouteServiceUPSIName).To(Equal("autoscaler-activator-rs"))
			Expect(conf.Validate()).To(Succeed())
		})
	})
})
