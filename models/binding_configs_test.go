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
		testAppGUID   = GUID("test-app-guid")
	)

	Context("NewBindingConfig", func() {
		Context("with default authentication scheme", func() {
			BeforeEach(func() {
				bindingConfig = NewBindingConfig(testAppGUID, nil)
			})

			It("should create binding config with default auth scheme", func() {
				Expect(bindingConfig).NotTo(BeNil())
				Expect(bindingConfig.GetAppGUID()).To(Equal(testAppGUID))
				Expect(bindingConfig.GetCustomMetricsBindingAuth()).To(BeNil())
			})
		})

		Context("with BindingSecret authentication scheme", func() {
			BeforeEach(func() {
				bindingConfig = NewBindingConfig(testAppGUID, &BindingSecret)
			})

			It("should create binding config with BindingSecret auth scheme", func() {
				Expect(bindingConfig).NotTo(BeNil())
				Expect(bindingConfig.GetAppGUID()).To(Equal(testAppGUID))
				Expect(*bindingConfig.GetCustomMetricsBindingAuth()).To(Equal(BindingSecret))
			})
		})

		Context("with X509Certificate authentication scheme", func() {
			BeforeEach(func() {
				bindingConfig = NewBindingConfig(testAppGUID, &X509Certificate)
			})

			It("should create binding config with X509Certificate auth scheme", func() {
				Expect(bindingConfig).NotTo(BeNil())
				Expect(bindingConfig.GetAppGUID()).To(Equal(testAppGUID))
				Expect(*bindingConfig.GetCustomMetricsBindingAuth()).To(Equal(X509Certificate))
			})
		})
	})

	Context("ToRawJSON", func() {
		var rawJSON json.RawMessage
		var rawJSONString string

		Context("with default authentication scheme", func() {
			BeforeEach(func() {
				bindingConfig = NewBindingConfig(testAppGUID, nil)
				rawJSON, err = bindingConfig.ToRawJSON()
				rawJSONString = string(rawJSON)
			})

			It("should serialize to raw JSON without error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rawJSON).NotTo(BeNil())
				Expect(rawJSONString).To(ContainSubstring(`"app_guid":"test-app-guid"`))
			})

			It("should not include credential-type when using default scheme", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rawJSONString).NotTo(ContainSubstring("credential-type"))
			})
		})

		Context("with BindingSecret authentication scheme", func() {
			BeforeEach(func() {
				bindingConfig = NewBindingConfig(testAppGUID, &BindingSecret)
				rawJSON, err = bindingConfig.ToRawJSON()
				rawJSONString = string(rawJSON)
			})

			It("should serialize to raw JSON without error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rawJSON).NotTo(BeNil())
				Expect(rawJSONString).To(ContainSubstring(`"app_guid":"test-app-guid"`))
				Expect(rawJSONString).To(ContainSubstring(`"credential-type":"binding-secret"`))
			})

			It("should be compliant with our official format", func() {
				correctJSONString := `{"app_guid":"test-app-guid","credential-type":"binding-secret"}`
				Expect(err).NotTo(HaveOccurred())
				Expect(rawJSON).NotTo(BeNil())
				Expect(rawJSONString).To(Equal(correctJSONString))
			})
		})

		Context("with X509Certificate authentication scheme", func() {
			BeforeEach(func() {
				bindingConfig = NewBindingConfig(testAppGUID, &X509Certificate)
				rawJSON, err = bindingConfig.ToRawJSON()
				rawJSONString = string(rawJSON)
			})

			It("should serialize to raw JSON without error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(rawJSON).NotTo(BeNil())
				Expect(rawJSONString).To(ContainSubstring(`"app_guid":"test-app-guid"`))
				Expect(rawJSONString).To(ContainSubstring(`"credential-type":"x509"`))
			})
		})
	})

	Context("BindingConfigFromRawJSON", func() {
		var rawJSON json.RawMessage

		When("Deserialising a complete BindingConfig", func() {
			BeforeEach(func() {
				rawJSON = []byte(`
{
  "app_guid": "550e8400-e29b-41d4-a716-446655440000",
  "credential-type": "binding-secret"
}`)
			})

			It("should deserialize from the correct rawJSONString", func() {
				bindingConfig, err = BindingConfigFromRawJSON(rawJSON)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingConfig).NotTo(BeNil())
				Expect(bindingConfig.GetAppGUID()).To(Equal(GUID("550e8400-e29b-41d4-a716-446655440000")))
				Expect(*bindingConfig.GetCustomMetricsBindingAuth()).To(Equal(BindingSecret))
			})
		})

		Context("with default authentication scheme", func() {
			BeforeEach(func() {
				rawJSON = json.RawMessage(`{"app_guid":"test-app-guid"}`)
			})

			It("should deserialize from raw JSON without error", func() {
				bindingConfig, err = BindingConfigFromRawJSON(rawJSON)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingConfig).NotTo(BeNil())
				Expect(bindingConfig.GetAppGUID()).To(Equal(testAppGUID))
				Expect(bindingConfig.GetCustomMetricsBindingAuth()).To(BeNil())
			})
		})

		Context("with BindingSecret authentication scheme", func() {
			BeforeEach(func() {
				rawJSON = json.RawMessage(`{"app_guid":"test-app-guid","credential-type":"binding-secret"}`)
			})

			It("should deserialize from raw JSON without error", func() {
				bindingConfig, err = BindingConfigFromRawJSON(rawJSON)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingConfig).NotTo(BeNil())
				Expect(bindingConfig.GetAppGUID()).To(Equal(testAppGUID))
				Expect(*bindingConfig.GetCustomMetricsBindingAuth()).To(Equal(BindingSecret))
			})
		})

		Context("with X509Certificate authentication scheme", func() {
			BeforeEach(func() {
				rawJSON = json.RawMessage(`{"app_guid":"test-app-guid","credential-type":"x509"}`)
			})

			It("should deserialize from raw JSON without error", func() {
				bindingConfig, err = BindingConfigFromRawJSON(rawJSON)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingConfig).NotTo(BeNil())
				Expect(bindingConfig.GetAppGUID()).To(Equal(testAppGUID))
				Expect(*bindingConfig.GetCustomMetricsBindingAuth()).To(Equal(X509Certificate))
			})
		})

		Context("with invalid credential type", func() {
			BeforeEach(func() {
				rawJSON = json.RawMessage(`{"app_guid":"test-app-guid","credential-type":"invalid_type"}`)
			})

			It("should return an error", func() {
				bindingConfig, err = BindingConfigFromRawJSON(rawJSON)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unknown credential type"))
				Expect(bindingConfig).To(BeNil())
			})
		})

		Context("with invalid JSON", func() {
			BeforeEach(func() {
				rawJSON = json.RawMessage(`{"invalid_json"}`)
			})

			It("should return an error", func() {
				bindingConfig, err = BindingConfigFromRawJSON(rawJSON)
				Expect(err).To(HaveOccurred())
				Expect(bindingConfig).To(BeNil())
			})
		})

		Context("with empty data", func() {
			BeforeEach(func() {
				rawJSON = json.RawMessage(``)
			})

			It("should return an error", func() {
				bindingConfig, err = BindingConfigFromRawJSON(rawJSON)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("data must not be empty"))
				Expect(bindingConfig).To(BeNil())
			})
		})
	})

	Context("ToRawJSON and BindingConfigFromRawJSON", func() {
		When("executed in succession", func() {
			var bindingConfig1, bindingConfig2 *BindingConfig
			var rawJSON json.RawMessage

			BeforeEach(func() {
				bindingConfig1 = NewBindingConfig(testAppGUID, &X509Certificate)
				rawJSON, err = bindingConfig1.ToRawJSON()
				Expect(err).NotTo(HaveOccurred())
				bindingConfig2, err = BindingConfigFromRawJSON(rawJSON)
			})

			It("should return an equivalent BindingConfig", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(bindingConfig2).NotTo(BeNil())
				Expect(*bindingConfig2).To(Equal(*bindingConfig1))
			})
		})
	})

	Context("String", func() {
		Context("with default authentication scheme", func() {
			BeforeEach(func() {
				bindingConfig = NewBindingConfig(testAppGUID, nil)
			})

			It("should return string representation", func() {
				str := bindingConfig.String()
				Expect(str).To(ContainSubstring("test-app-guid"))
				Expect(str).To(ContainSubstring("useDefaultAuthScheme: true"))
			})
		})

		Context("with custom authentication scheme", func() {
			BeforeEach(func() {
				bindingConfig = NewBindingConfig(testAppGUID, &BindingSecret)
			})

			It("should return string representation", func() {
				str := bindingConfig.String()
				Expect(str).To(ContainSubstring("test-app-guid"))
				Expect(str).To(ContainSubstring("useDefaultAuthScheme: false"))
			})
		})
	})
})

var _ = Describe("CustomMetricsBindingAuthScheme", func() {
	Context("String", func() {
		It("should return correct string for BindingSecret", func() {
			Expect(BindingSecret.String()).To(Equal("binding-secret"))
		})

		It("should return correct string for X509Certificate", func() {
			Expect(X509Certificate.String()).To(Equal("x509"))
		})
	})

	Context("JSON marshaling", func() {
		It("should marshal BindingSecret correctly", func() {
			data, err := json.Marshal(BindingSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal(`"binding-secret"`))
		})

		It("should marshal X509Certificate correctly", func() {
			data, err := json.Marshal(X509Certificate)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(Equal(`"x509"`))
		})
	})

	Context("JSON unmarshaling", func() {
		var scheme CustomMetricsBindingAuthScheme
		var err error

		Context("with valid BindingSecret value", func() {
			BeforeEach(func() {
				err = json.Unmarshal([]byte(`"binding-secret"`), &scheme)
			})

			It("should unmarshal correctly", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(scheme).To(Equal(BindingSecret))
			})
		})

		Context("with valid X509Certificate value", func() {
			BeforeEach(func() {
				err = json.Unmarshal([]byte(`"x509"`), &scheme)
			})

			It("should unmarshal correctly", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(scheme).To(Equal(X509Certificate))
			})
		})

		Context("with invalid value", func() {
			BeforeEach(func() {
				err = json.Unmarshal([]byte(`"invalid"`), &scheme)
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unknown credential type"))
			})
		})

		Context("with invalid JSON", func() {
			BeforeEach(func() {
				err = json.Unmarshal([]byte(`{invalid}`), &scheme)
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("ParseCustomMetricsBindingAuthScheme", func() {
		var scheme *CustomMetricsBindingAuthScheme
		var err error

		Context("with valid BindingSecret string", func() {
			BeforeEach(func() {
				scheme, err = ParseCustomMetricsBindingAuthScheme("binding-secret")
			})

			It("should parse correctly", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(*scheme).To(Equal(BindingSecret))
			})
		})

		Context("with valid X509Certificate string", func() {
			BeforeEach(func() {
				scheme, err = ParseCustomMetricsBindingAuthScheme("x509")
			})

			It("should parse correctly", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(*scheme).To(Equal(X509Certificate))
			})
		})

		Context("with invalid string", func() {
			BeforeEach(func() {
				scheme, err = ParseCustomMetricsBindingAuthScheme("invalid")
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unknown credential type"))
				Expect(scheme).To(BeNil())
			})
		})
	})
})
