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
