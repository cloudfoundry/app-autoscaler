package models_test

import (
	"encoding/json"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BindingConfigs", func() {

	var (
		bindingConfig *BindingConfig
		err           error
		testAppGUID   GUID = GUID("test-app-guid")
	)

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
						CustomMetricsStrategy: CustomMetricsBoundApp.String(),
					}
				})

				It("should create binding config with correct values", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(bindingConfig).NotTo(BeNil())
					Expect(bindingConfig.GetAppGUID()).To(Equal(testAppGUID))
					Expect(bindingConfig.GetCustomMetricStrategy()).To(Equal(CustomMetricsBoundApp))
				})
			})

			Context("with CustomMetricsSameApp strategy and AppID", func() {
				BeforeEach(func() {
					serviceBinding = &ServiceBinding{
						AppID:                 string(testAppGUID),
						CustomMetricsStrategy: CustomMetricsSameApp.String(),
					}
				})

				It("should create binding config with correct values", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(bindingConfig).NotTo(BeNil())
					Expect(bindingConfig.GetAppGUID()).To(Equal(testAppGUID))
					Expect(bindingConfig.GetCustomMetricStrategy()).To(Equal(CustomMetricsSameApp))
				})
			})

			Context("with only CustomMetricsBoundApp strategy (no AppID)", func() {
				BeforeEach(func() {
					serviceBinding = &ServiceBinding{
						AppID:                 "",
						CustomMetricsStrategy: CustomMetricsBoundApp.String(),
					}
				})

				It("should create binding config with correct strategy", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(bindingConfig).NotTo(BeNil())
					Expect(bindingConfig.GetAppGUID()).To(Equal(GUID("")))
					Expect(bindingConfig.GetCustomMetricStrategy()).To(Equal(CustomMetricsBoundApp))
				})
			})

			Context("with empty strategy", func() {
				BeforeEach(func() {
					serviceBinding = &ServiceBinding{
						CustomMetricsStrategy: "",
					}
				})

				It("should create a binding-config with default-strategy", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(bindingConfig.GetCustomMetricStrategy()).To(Equal(DefaultCustomMetricsStrategy))
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

	Context("ToRawJSON", func() {
		var strategy CustomMetricsStrategy = CustomMetricsBoundApp
		var err error
		var rawJSON json.RawMessage
		var rawJSONString string

		BeforeEach(func() {
			bindingConfig = NewBindingConfig(testAppGUID, strategy)
			rawJSON, err = bindingConfig.ToRawJSON()
			rawJSONString = string(rawJSON)
		})

		It("should serialize to raw JSON without error", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(rawJSON).NotTo(BeNil())
			Expect(string(rawJSON)).To(ContainSubstring(`"app_guid":"test-app-guid"`))
			Expect(string(rawJSON)).To(ContainSubstring(`"allow_from":"bound_app"`))
		})

		It("should be compliant with our official format", func() {
			correctJSONString := `{"app_guid":"test-app-guid","custom_metrics":{"metric_submission_strategy":{"allow_from":"bound_app"}}}`
			Expect(err).NotTo(HaveOccurred())
			Expect(rawJSON).NotTo(BeNil())
			Expect(rawJSONString).To(Equal(string(correctJSONString)))
		})
	})

	Context("FromRawJSON", func() {
		var rawJSON json.RawMessage
		var err error

		BeforeEach(func() {
			rawJSON = json.RawMessage(`{"app_guid":"test-app-guid","custom_metrics":{"metric_submission_strategy":{"allow_from":"bound_app"}}}`)
		})

		It("should deserialize from raw JSON without error", func() {
			bindingConfig, err = BindingConfigFromRawJSON(rawJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(bindingConfig).NotTo(BeNil())
			Expect(bindingConfig.GetAppGUID()).To(Equal(testAppGUID))
			Expect(bindingConfig.GetCustomMetricStrategy()).To(Equal(CustomMetricsBoundApp))
		})

		It("should return an error for invalid JSON", func() {
			rawJSON = json.RawMessage(`{"invalid_json"}`)
			bindingConfig, err = BindingConfigFromRawJSON(rawJSON)
			Expect(err).To(HaveOccurred())
			Expect(bindingConfig).To(BeNil())
		})
	})
})

// Ich möchte die Tests oben ersetzen durch aktualisierte Tests, die die neuen, öffentlich sichtbaren Funktionen und geänderten Inhalte überprüfen.
