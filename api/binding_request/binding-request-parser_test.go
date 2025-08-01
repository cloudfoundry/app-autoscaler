package binding_request_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	br "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request"
	clp "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request/clean_parser"
	cp "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request/combined_parser"
	lp "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_request/legacy_parser"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

var _ = Describe("BindingRequestParsers", func() {
	const cleanSchemaFilePath string = "file://./binding-request.json"
	const validModernBindingRequestRaw string = `
		{
		  "configuration": {
			"app_guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",
			"custom_metrics": {
			  "metric_submission_strategy": {
				"allow_from": "bound_app"
			  }
			}
		  },
		  "scaling-policy": {
			  "instance_min_count": 1,
			  "instance_max_count": 4,
			  "scaling_rules": [
				{
				  "metric_type": "memoryutil",
				  "breach_duration_secs": 600,
				  "threshold": 30,
				  "operator": "<",
				  "cool_down_secs": 300,
				  "adjustment": "-1"
				},
				{
				  "metric_type": "memoryutil",
				  "breach_duration_secs": 600,
				  "threshold": 90,
				  "operator": ">=",
				  "cool_down_secs": 300,
				  "adjustment": "+1"
				}
			  ]
		  }
		}`
	const validLegacyBindingRequestRaw string = `
		{
		 "configuration": {
			 "app_guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",
			 "custom_metrics": {
			   "metric_submission_strategy": {
				   "allow_from": "bound_app"
			   }
			 }
		 },
		 "instance_min_count": 1,
		 "instance_max_count": 4,
		 "scaling_rules": [
		   {
			 "metric_type": "memoryutil",
			 "breach_duration_secs": 600,
			 "threshold": 30,
			 "operator": "<",
			 "cool_down_secs": 300,
			 "adjustment": "-1"
		   },
		   {
			 "metric_type": "memoryutil",
			 "breach_duration_secs": 600,
			 "threshold": 90,
			 "operator": ">=",
			 "cool_down_secs": 300,
			 "adjustment": "+1"
		   }
		 ]
	   }`

	Describe("CleanBindingRequestParser", func() {
		var (
			cleanParser clp.CleanBindingRequestParser
			err         error
		)
		var _ = BeforeEach(func() {
			cleanParser, err = clp.NewFromFile(cleanSchemaFilePath)
			Expect(err).NotTo(HaveOccurred())
		})
		Context("When using the new format for binding-requests", func() {
			Context("and parsing a valid and complete one", func() {
				It("should return a correctly populated BindingRequestParameters", func() {
					bindingRequestRaw := validModernBindingRequestRaw

					bindingRequest, err := cleanParser.Parse(bindingRequestRaw)

					Expect(err).NotTo(HaveOccurred())
					Expect(bindingRequest.Configuration.AppGUID).To(
						Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
				})
			})
		})
	})

	Describe("LegacyBindingRequestParser", func() {
		var (
			legacyParser lp.LegacyBindingRequestParser
			err          error
		)
		var _ = BeforeEach(func() {
			legacyParser, err = lp.New()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When using the legacy format for binding-requests", func() {
			It("should return a correctly populated BindingRequestParameters", func() {
				bindingRequestRaw := validLegacyBindingRequestRaw

				bindingRequest, err := legacyParser.Parse(bindingRequestRaw)

				Expect(err).NotTo(HaveOccurred())
				Expect(bindingRequest.Configuration.AppGUID).
					To(Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
				// 🚧 To-do: Add a few more field-comparisons;
			})
		})
	})

	Describe("CombinedBindingRequestParser", func() {
		var (
			cleanParser    clp.CleanBindingRequestParser
			legacyParser   lp.LegacyBindingRequestParser
			combinedParser cp.CombinedBindingRequestParser

			err error
		)
		var _ = BeforeEach(func() {
			cleanParser, err = clp.NewFromFile(cleanSchemaFilePath)
			Expect(err).NotTo(HaveOccurred())
			legacyParser, err = lp.New()
			Expect(err).NotTo(HaveOccurred())
			combinedParser = cp.New([]br.Parser{cleanParser, legacyParser})
		})

		Context("When using the new format for binding-requests", func() {
			Context("and parsing a valid and complete one", func() {
				It("should return a correctly populated BindingRequestParameters", func() {
					bindingRequestRaw := validModernBindingRequestRaw

					bindingRequest, err := combinedParser.Parse(bindingRequestRaw)

					Expect(err).NotTo(HaveOccurred())
					Expect(bindingRequest.Configuration.AppGUID).To(
						Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
				})
			})
		})

		Context("When using the legacy format for binding-requests", func() {
			It("should return a correctly populated BindingRequestParameters", func() {
				bindingRequestRaw := validLegacyBindingRequestRaw

				bindingRequest, err := combinedParser.Parse(bindingRequestRaw)

				Expect(err).NotTo(HaveOccurred())
				Expect(bindingRequest.Configuration.AppGUID).
					To(Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
				// 🚧 To-do: Add a few more field-comparisons;
			})
		})
	})
})
