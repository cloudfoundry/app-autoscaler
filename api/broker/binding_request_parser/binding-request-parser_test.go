package binding_request_parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	br "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker/binding_request_parser"
	lp "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker/binding_request_parser/legacy"
	brp_v1 "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker/binding_request_parser/v0_1"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

var _ = Describe("BindingRequestParsers", func() {
	const v0_1SchemaFilePath string = "file://./v0_1/meta.schema.json"
	const legacySchemaFilePath string = "file://./legacy/schema.json"

	const validModernBindingRequestRaw string = `
	{
		"schema-version": "0.1",
		"configuration": {
			  "app_guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",
			  "custom_metrics": {
				  "metric_submission_strategy": {
					  "allow_from": "bound_app"
				  }
			  }
		},
		"instance_min_count": 1,
		"instance_max_count": 5,
		"scaling_rules": [
			  {
				  "metric_type": "memoryused",
				  "threshold": 30,
				  "operator": "<",
				  "adjustment": "-1"
			  }
		]
	}`
	const validLegacyBindingRequestRaw string = `
		{
		 "configuration": {
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

	Describe("v0.1_BindingRequestParser", func() {
		var (
			v0_1Parser brp_v1.BindingRequestParser
			err        error
		)
		var _ = BeforeEach(func() {
			v0_1Parser, err = brp_v1.NewFromFile(v0_1SchemaFilePath, models.X509Certificate)
			Expect(err).NotTo(HaveOccurred())
		})
		Context("When using the new format for binding-requests", func() {
			Context("and parsing a valid and complete one", func() {
				It("should return a correctly populated BindingRequestParameters", func() {
					bindingRequestRaw := validModernBindingRequestRaw
					ccAppGuid := models.GUID("") // Raw request is about creating a service-key;

					bindingRequest, err := v0_1Parser.Parse(bindingRequestRaw, ccAppGuid)

					Expect(err).NotTo(HaveOccurred())
					Expect(bindingRequest.GetConfiguration().GetAppGUID()).To(
						Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
				})
			})
		})
	})

	Describe("LegacyBindingRequestParser", func() {
		var (
			legacyParser lp.BindingRequestParser
			err          error
		)
		var _ = BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			legacyParser, err = lp.New(legacySchemaFilePath, models.X509Certificate)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When using the legacy format for binding-requests", func() {
			It("should return a correctly populated BindingRequestParameters", func() {
				bindingRequestRaw := validLegacyBindingRequestRaw
				appGuidFromCC := models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")

				appScalingConfig, err := legacyParser.Parse(bindingRequestRaw, appGuidFromCC)

				Expect(err).NotTo(HaveOccurred())
				Expect(appScalingConfig.GetConfiguration().GetAppGUID()).
					To(Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
				policy := appScalingConfig.GetScalingPolicy().GetPolicyDefinition()
				Expect(policy.InstanceMin).To(Equal(1))
				Expect(policy.InstanceMax).To(Equal(4))
				Expect(policy.ScalingRules).To(HaveLen(2))
				Expect(policy.ScalingRules[0].MetricType).To(Equal("memoryutil"))
				Expect(policy.ScalingRules[0].BreachDurationSeconds).To(Equal(600))
				Expect(policy.ScalingRules[0].Threshold).To(Equal(30.0))
				Expect(policy.ScalingRules[0].Operator).To(Equal("<"))
				Expect(policy.ScalingRules[0].CoolDownSeconds).To(Equal(300))
				Expect(policy.ScalingRules[0].Adjustment).To(Equal("-1"))
				Expect(policy.ScalingRules[1].MetricType).To(Equal("memoryutil"))
				Expect(policy.ScalingRules[1].BreachDurationSeconds).To(Equal(600))
				Expect(policy.ScalingRules[1].Threshold).To(Equal(90.0))
				Expect(policy.ScalingRules[1].Operator).To(Equal(">="))
				Expect(policy.ScalingRules[1].CoolDownSeconds).To(Equal(300))
				Expect(policy.ScalingRules[1].Adjustment).To(Equal("+1"))
			})
		})
	})

	Describe("CombinedBindingRequestParser", func() {
		var (
			parser br.BindRequestParser
			err    error
		)
		var _ = BeforeEach(func() {
			parser, err = br.New(
				legacySchemaFilePath,
				v0_1SchemaFilePath,
				models.X509Certificate,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When using the v0.1 schema for binding-requests", func() {
			Context("and parsing a valid and complete one", func() {
				It("should return a correctly populated BindingRequestParameters", func() {
					bindingRequestRaw := validModernBindingRequestRaw
					appGuidFromCC := models.GUID("")

					bindingRequest, err := parser.Parse(bindingRequestRaw, appGuidFromCC)

					Expect(err).NotTo(HaveOccurred())
					Expect(bindingRequest.GetConfiguration().GetAppGUID()).To(
						Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
				})
			})
		})

		Context("When using the legacy schema for binding-requests", func() {
			It("should return a correctly populated BindingRequestParameters", func() {
				bindingRequestRaw := validLegacyBindingRequestRaw
				appGuidFromCC := models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")

				bindingRequest, err := parser.Parse(bindingRequestRaw, appGuidFromCC)

				Expect(err).NotTo(HaveOccurred())
				Expect(bindingRequest.GetConfiguration().GetAppGUID()).
					To(Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
				// 🚧 To-do: Add a few more field-comparisons;
			})
		})
	})
})
