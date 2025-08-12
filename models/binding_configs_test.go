package models_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BindingConfigs", func() {

	var (
		bindingConfig *BindingConfig
		err           error
		strategy      string
		testAppGUID   GUID = GUID("test-app-guid")
	)

	Context("BindingConfigFromParameters", func() {
		JustBeforeEach(func() {
			bindingConfig, err = BindingConfigFromParameters(testAppGUID, strategy)
		})

		When("valid parameters are provided", func() {
			Context("with CustomMetricsSameApp strategy", func() {
				BeforeEach(func() {
					strategy = CustomMetricsSameApp
				})

				It("should create binding config with correct values", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(bindingConfig).NotTo(BeNil())
					Expect(bindingConfig.AppGUID).To(Equal(testAppGUID))
					Expect(bindingConfig.GetCustomMetricsStrategy()).To(Equal(CustomMetricsSameApp))
				})
			})

			Context("with CustomMetricsBoundApp strategy", func() {
				BeforeEach(func() {
					strategy = CustomMetricsBoundApp
				})

				It("should create binding config with correct values", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(bindingConfig).NotTo(BeNil())
					Expect(bindingConfig.AppGUID).To(Equal(testAppGUID))
					Expect(bindingConfig.GetCustomMetricsStrategy()).To(Equal(CustomMetricsBoundApp))
				})
			})
		})

		When("invalid strategy is provided", func() {
			BeforeEach(func() {
				strategy = "invalid_strategy"
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error: provided strategy is unsupported"))
				Expect(bindingConfig).To(BeNil())
			})
		})

		When("empty strategy is provided", func() {
			BeforeEach(func() {
				strategy = ""
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error: provided strategy is unsupported"))
				Expect(bindingConfig).To(BeNil())
			})
		})

		When("empty appGUID is provided", func() {
			BeforeEach(func() {
				testAppGUID = GUID("")
				strategy = CustomMetricsSameApp
			})

			It("should create binding config with empty appGUID", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingConfig).NotTo(BeNil())
				Expect(bindingConfig.AppGUID).To(Equal(GUID("")))
				Expect(bindingConfig.GetCustomMetricsStrategy()).To(Equal(CustomMetricsSameApp))
			})
		})
	})

	Context("BindingConfigFromServiceBinding", func() {
		var (
			serviceBinding *ServiceBinding
		)

		JustBeforeEach(func() {
			bindingConfig, err = BindingConfigFromServiceBinding(serviceBinding)
		})

		When("valid service binding is provided", func() {
			Context("with CustomMetricsBoundApp strategy and AppID", func() {
				BeforeEach(func() {
					serviceBinding = &ServiceBinding{
						AppID:                 string(testAppGUID),
						CustomMetricsStrategy: CustomMetricsBoundApp,
					}
				})

				It("should create binding config with correct values", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(bindingConfig).NotTo(BeNil())
					Expect(bindingConfig.AppGUID).To(Equal(testAppGUID))
					Expect(bindingConfig.GetCustomMetricsStrategy()).To(Equal(CustomMetricsBoundApp))
				})
			})

			Context("with CustomMetricsSameApp strategy and AppID", func() {
				BeforeEach(func() {
					serviceBinding = &ServiceBinding{
						AppID:                 string(testAppGUID),
						CustomMetricsStrategy: CustomMetricsSameApp,
					}
				})

				It("should create binding config with correct values", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(bindingConfig).NotTo(BeNil())
					Expect(bindingConfig.AppGUID).To(Equal(testAppGUID))
					Expect(bindingConfig.GetCustomMetricsStrategy()).To(Equal(CustomMetricsSameApp))
				})
			})

			Context("with only CustomMetricsBoundApp strategy (no AppID)", func() {
				BeforeEach(func() {
					serviceBinding = &ServiceBinding{
						AppID:                 "",
						CustomMetricsStrategy: CustomMetricsBoundApp,
					}
				})

				It("should create binding config with correct strategy", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(bindingConfig).NotTo(BeNil())
					Expect(bindingConfig.AppGUID).To(Equal(GUID("")))
					Expect(bindingConfig.GetCustomMetricsStrategy()).To(Equal(CustomMetricsBoundApp))
				})
			})
		})

		When("invalid parameters are provided", func() {
			Context("with nil service binding", func() {
				BeforeEach(func() {
					serviceBinding = nil
				})

				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("serviceBinding must not be nil"))
					Expect(bindingConfig).To(BeNil())
				})
			})

			Context("with empty strategy", func() {
				BeforeEach(func() {
					serviceBinding = &ServiceBinding{
						CustomMetricsStrategy: "",
					}
				})

				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error: serviceBinding contained unsupported strategy"))
					Expect(bindingConfig).To(BeNil())
				})
			})

			Context("with invalid custom metrics strategy", func() {
				BeforeEach(func() {
					serviceBinding = &ServiceBinding{
						AppID:                 string(testAppGUID),
						CustomMetricsStrategy: "invalid_strategy",
					}
				})

				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("serviceBinding contained unsupported strategy"))
					Expect(bindingConfig).To(BeNil())
				})
			})
		})
	})

})
