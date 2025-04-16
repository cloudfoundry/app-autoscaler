package binding_requests_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	br "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_requests"
	cp "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/binding_requests/clean_parser"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

var _ = Describe("BindingRequestParser", func() {
	const schemaFilePath string = "file://./binding-request.json"
	var (
		err                  error
		bindingRequestParser br.Parser
	)
	var _ = BeforeEach(func() {
		// 🚧 To-do: Use a proper combined parser.
		bindingRequestParser, err = cp.NewFromFile(schemaFilePath)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("When using the new format for binding-requests", func() {
		Context("and parsing a valid and complete one", func() {
			It("should return a correctly populated BindingRequestParameters", func() {
				bindingRequestRaw := `
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

				bindingRequest, err := bindingRequestParser.Parse(bindingRequestRaw)

				Expect(err).NotTo(HaveOccurred())
				Expect(bindingRequest.Configuration.AppGUID).To(Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
			})
		})
	})

	Context("When using the legacy format for binding-requests", func() {
		It("should return a correctly populated BindingRequestParameters", func() {
			bindingRequestRaw := `
{
  "configuration": {
	  "app-guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",
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

			bindingRequest, err := bindingRequestParser.Parse(bindingRequestRaw)

			Expect(err).NotTo(HaveOccurred())
			Expect(bindingRequest.Configuration.AppGUID).
				To(Equal(models.GUID("8d0cee08-23ad-4813-a779-ad8118ea0b91")))
		})
	})
})
