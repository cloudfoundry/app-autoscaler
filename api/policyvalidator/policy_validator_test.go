package policyvalidator_test

import (
	"encoding/json"
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/policyvalidator"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyValidator", func() {
	var (
		policyValidator        *PolicyValidator
		errResult              []PolicyValidationErrors
		policyString           string
		policy                 *models.ScalingPolicy
		policyJson             string
		lowerCPUThreshold      int
		upperCPUThreshold      int
		lowerCPUUtilThreshold  int
		upperCPUUtilThreshold  int
		lowerDiskUtilThreshold int
		upperDiskUtilThreshold int
		lowerDiskThreshold     int
		upperDiskThreshold     int
	)
	BeforeEach(func() {
		lowerCPUThreshold = 1
		upperCPUThreshold = 15

		lowerCPUThreshold = 1
		upperCPUThreshold = 100

		lowerCPUUtilThreshold = 1
		upperCPUUtilThreshold = 100

		lowerDiskUtilThreshold = 1
		upperDiskUtilThreshold = 100

		lowerDiskThreshold = 1
		upperDiskThreshold = 2 * 1024

		policyValidator = NewPolicyValidator(
			"./policy_json.schema.json",
			lowerCPUThreshold,
			upperCPUThreshold,
			lowerCPUUtilThreshold,
			upperCPUUtilThreshold,
			lowerDiskUtilThreshold,
			upperDiskUtilThreshold,
			lowerDiskThreshold,
			upperDiskThreshold,
		)
	})
	JustBeforeEach(func() {
		policy, errResult = policyValidator.ParseAndValidatePolicy(json.RawMessage(policyString))
		policyBytes, err := json.Marshal(policy)
		Expect(err).ToNot(HaveOccurred())
		policyJson = string(policyBytes)
	})
	Context("Policy Schema &  Validation", func() {
		Context("when invalid json", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_min_count:1,
				}`
			})
			It("should fail", func() {
				Expect(errResult).To(Equal([]PolicyValidationErrors{
					{
						Context:     "(root)",
						Description: "invalid character '\\n' in string literal",
					},
				}))
			})
		})

		Context("when instance_min_count is missing", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_max_count":4,
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"threshold":30,
						"operator":"<",
						"adjustment":"-1"
					}]
				}`
			})
			It("should fail", func() {
				Expect(errResult).To(Equal([]PolicyValidationErrors{
					{
						Context:     "(root)",
						Description: "instance_min_count is required",
					},
				}))
			})
		})
		Context("when instance_min_count is < 1", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_min_count":0,
					"instance_max_count":4,
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"threshold":30,
						"operator":"<",
						"adjustment":"-1"
					}]
				}`
			})
			It("should fail", func() {
				Expect(errResult).To(Equal([]PolicyValidationErrors{
					{
						Context:     "(root).instance_min_count",
						Description: "Must be greater than or equal to 1",
					},
				}))
			})
		})

		Context("when instance_max_count is missing", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_min_count":4,
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"threshold":30,
						"operator":"<",
						"adjustment":"-1"
					}]
				}`
			})
			It("should fail", func() {
				Expect(errResult).To(Equal([]PolicyValidationErrors{
					{
						Context:     "(root)",
						Description: "instance_max_count is required",
					},
				}))
			})
		})

		Context("when instance_min_count is greater than instance_max_count", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_min_count":10,
					"instance_max_count":4,
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"threshold":30,
						"operator":"<",
						"adjustment":"-1"
					}]
				}`
			})
			It("should fail", func() {
				Expect(errResult).To(Equal([]PolicyValidationErrors{
					{
						Context:     "(root).instance_min_count",
						Description: "instance_min_count 10 is higher than instance_max_count 4",
					},
				}))
			})
		})

		Context("when scaling_rules and schedules both are missing", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_min_count":1,
					"instance_max_count":2
				}`
			})
			It("should fail ", func() {
				Expect(errResult).To(Equal([]PolicyValidationErrors{
					{
						Context:     "(root)",
						Description: "Must validate at least one schema (anyOf)",
					},
					{
						Context:     "(root)",
						Description: "scaling_rules is required",
					},
				}))
			})
		})

		Context("when only scaling_rules are present", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
			})
			It("should succeed", func() {
				Expect(policyJson).To(MatchJSON(policyString))
			})
		})

		Context("when additional fields are present", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"stats_window_secs": 600,
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}],
					"is_admin": true,
					"is_sso": true,
					"role": "admin"
				}`
			})
			It("the validation succeed and remove them", func() {
				validPolicyString := `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				Expect(policyJson).To(MatchJSON(validPolicyString))
			})
		})

		Context("Scaling Rules", func() {

			Context("when metric_type is missing", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "metric_type is required",
						},
					}))
				})
			})

			Context("when metric_type is non-alphanum", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type": "%me$ric",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.metric_type",
							Description: "Does not match pattern '^[a-zA-Z0-9_]+$'",
						},
					}))
				})
			})

			Context("when metric_type is too long", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type": "custom_metric_custom_metric_custom_metric_custom_metric_custom_metric_custom_metric_custom_metric_custom_metric",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.metric_type",
							Description: "String length must be less than or equal to 100",
						},
					}))
				})
			})

			Context("when threshold is missing", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "threshold is required",
						},
					}))
				})
			})
			Context("when threshold is not integer", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold": 90.55,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root)",
							Description: "json: cannot unmarshal number 90.55 into Go struct field ScalingRule.scaling_rules.threshold of type int64",
						},
					}))
				})
			})

			Context("when threshold for memoryused is less than 1", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"breach_duration_secs":600,
						"threshold": -90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type memoryused should be greater than or equal 1",
						},
					}))
				})
			})

			Context("when threshold for memoryused is greater than 100", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"breach_duration_secs":600,
						"threshold": 500,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should succeed", func() {
					Expect(errResult).To(BeNil())
					Expect(policyJson).To(MatchJSON(policyString))
				})
			})

			Context("when threshold for memoryutil is less than 0", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold": -90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type memoryutil should be greater than or equal 1 and less than or equal to 100",
						},
					}))
				})
			})

			Context("when threshold for memoryutil is greater than 100", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold": 500,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type memoryutil should be greater than or equal 1 and less than or equal to 100",
						},
					}))
				})
			})

			Context("when threshold for responsetime is less than 1", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"responsetime",
						"breach_duration_secs":600,
						"threshold": -90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type responsetime should be greater than or equal 1",
						},
					}))
				})
			})

			Context("when threshold for responsetime is greater than 100", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"responsetime",
						"breach_duration_secs":600,
						"threshold": 500,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should succeed", func() {
					Expect(policyJson).To(MatchJSON(policyString))
				})
			})

			Context("when threshold for throughput is less than 0", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"throughput",
						"breach_duration_secs":600,
						"threshold": -90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type throughput should be greater than or equal 1",
						},
					}))
				})
			})

			Context("when threshold for throughput is greater than 100", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"throughput",
						"breach_duration_secs":600,
						"threshold": 500,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should succeed", func() {
					Expect(errResult).To(BeNil())
					Expect(policyJson).To(MatchJSON(policyString))
				})
			})

			Context(fmt.Sprintf("when threshold for cpu is less than %d", lowerCPUThreshold), func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"cpu",
						"breach_duration_secs":600,
						"threshold": 0,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: fmt.Sprintf("scaling_rules[0].threshold for metric_type cpu should be greater than or equal %d and less than or equal to %d", lowerCPUThreshold, upperCPUThreshold),
						},
					}))
				})
			})

			Context(fmt.Sprintf("when threshold for cpu is greater than %d", upperCPUThreshold), func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"cpu",
						"breach_duration_secs":600,
						"threshold": 500,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: fmt.Sprintf("scaling_rules[0].threshold for metric_type cpu should be greater than or equal %d and less than or equal to %d", lowerCPUThreshold, upperCPUThreshold),
						},
					}))
				})
			})

			Context(fmt.Sprintf("when threshold for cpuutil is less than %d", lowerCPUUtilThreshold), func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"cpuutil",
						"breach_duration_secs":600,
						"threshold": 0,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: fmt.Sprintf("scaling_rules[0].threshold for metric_type cpuutil should be greater than or equal %d and less than or equal to %d", lowerCPUUtilThreshold, upperCPUUtilThreshold),
						},
					}))
				})
			})

			Context(fmt.Sprintf("when threshold for cpuutil is greater than %d", upperCPUUtilThreshold), func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"cpuutil",
						"breach_duration_secs":600,
						"threshold": 999,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: fmt.Sprintf("scaling_rules[0].threshold for metric_type cpuutil should be greater than or equal %d and less than or equal to %d", lowerCPUUtilThreshold, upperCPUUtilThreshold),
						},
					}))
				})
			})

			Context(fmt.Sprintf("when threshold for diskutil is less than%d", lowerDiskUtilThreshold), func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"diskutil",
						"breach_duration_secs":600,
						"threshold": 0,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: fmt.Sprintf("scaling_rules[0].threshold for metric_type diskutil should be greater than or equal %d and less than or equal to %d", lowerDiskUtilThreshold, upperCPUUtilThreshold),
						},
					}))
				})
			})

			Context(fmt.Sprintf("when threshold for diskutil is greater than %d", upperDiskUtilThreshold), func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"diskutil",
						"breach_duration_secs":600,
						"threshold": 101,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: fmt.Sprintf("scaling_rules[0].threshold for metric_type diskutil should be greater than or equal %d and less than or equal to %d", lowerDiskUtilThreshold, upperDiskUtilThreshold),
						},
					}))
				})
			})

			Context(fmt.Sprintf("when threshold for disk is less than %d", lowerDiskThreshold), func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"disk",
						"breach_duration_secs":600,
						"threshold": 0,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: fmt.Sprintf("scaling_rules[0].threshold for metric_type disk should be greater than or equal %d and less than or equal to %d", lowerDiskThreshold, upperDiskThreshold),
						},
					}))
				})
			})

			Context(fmt.Sprintf("when threshold for disk is greater than %d", upperDiskThreshold), func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"disk",
						"breach_duration_secs":600,
						"threshold": 2049,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: fmt.Sprintf("scaling_rules[0].threshold for metric_type disk should be greater than or equal %d and less than or equal to %d", lowerDiskThreshold, upperDiskThreshold),
						},
					}))
				})
			})

			Context("when operator is missing", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "operator is required",
						},
					}))
				})
			})

			Context("when operator is invalid", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"operator": "abcd",
						"threshold":90,
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.operator",
							Description: "scaling_rules.0.operator must be one of the following: \"\\u003c\", \"\\u003e\", \"\\u003c=\", \"\\u003e=\"",
						},
					}))
				})
			})

			Context("when adjustment is missing", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "adjustment is required",
						},
					}))
				})
			})

			Context("when adjustment is invalid", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment": "5"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.adjustment",
							Description: "Does not match pattern '^[-+][1-9]+[0-9]*%?$'",
						},
					}))
				})
			})

			Context("when adjustment is number type", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment": "+1"
					},{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":"<=",
						"cool_down_secs":300,
						"adjustment": "-2"
					}]
				}`
				})
				It("should succeed", func() {
					Expect(errResult).To(BeNil())
					Expect(policyJson).To(MatchJSON(policyString))
				})
			})

			Context("when adjustment is percentage type", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment": "+100%"
					},{
						"metric_type":"memoryutil",
						"breach_duration_secs":600,
						"threshold":90,
						"operator":"<=",
						"cool_down_secs":300,
						"adjustment": "-200%"
					}]
				}`
				})
				It("should succeed", func() {
					Expect(errResult).To(BeNil())
					Expect(policyJson).To(MatchJSON(policyString))
				})
			})

			Context("when breach_duration_secs is missing", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment": "+1"
					}]
				}`
				})
				It("should succeed", func() {
					Expect(errResult).To(BeNil())
					Expect(policyJson).To(MatchJSON(policyString))
				})
			})

			Context("when breach_duration_secs is not an integer", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs": "a",
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment": "+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root)",
							Description: "json: cannot unmarshal string into Go struct field ScalingRule.scaling_rules.breach_duration_secs of type int",
						},
					}))
				})
			})

			Context("when breach_duration_secs is less than 60", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs": 5,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment": "+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.breach_duration_secs",
							Description: "Must be greater than or equal to 60",
						},
					}))
				})
			})

			Context("when breach_duration_secs is greater than 3600", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs": 55000,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment": "+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.breach_duration_secs",
							Description: "Must be less than or equal to 3600",
						},
					}))
				})
			})

			Context("when cool_down_secs is missing", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs": 300,
						"threshold":90,
						"operator":">=",
						"adjustment": "+1"
					}]
				}`
				})
				It("should succeed", func() {
					Expect(policyJson).To(MatchJSON(policyString))
				})
			})

			Context("when cool_down_secs is less than 60", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs": 300,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":5,
						"adjustment": "+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.cool_down_secs",
							Description: "Must be greater than or equal to 60",
						},
					}))
				})
			})

			Context("when cool_down_secs is greater than 3600", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryutil",
						"breach_duration_secs": 300,
						"threshold":90,
						"operator":">=",
						"cool_down_secs":55000,
						"adjustment": "+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.cool_down_secs",
							Description: "Must be less than or equal to 3600",
						},
					}))
				})
			})
		})
		Context("Schedules", func() {

			Context("when timezone is missing", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"schedules":{
						"specific_date":[
						   {
							  "start_date_time":"2099-01-04T20:00",
							  "end_date_time":"2099-02-19T23:15",
							  "instance_min_count":2,
							  "instance_max_count":5,
							  "initial_min_instance_count":3
						   }
						]
					 }
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).schedules",
							Description: "timezone is required",
						},
					}))
				})
			})

			Context("when timezone is invalid", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"schedules":{
						"timezone":"invalid-timezone",
						"specific_date":[
						   {
							  "start_date_time":"2099-01-04T20:00",
							  "end_date_time":"2099-02-19T23:15",
							  "instance_min_count":2,
							  "instance_max_count":5,
							  "initial_min_instance_count":3
						   }
						]
					 }
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).schedules.timezone",
							Description: "schedules.timezone must be one of the following: \"Etc/GMT+12\", \"Etc/GMT+11\", \"Pacific/Midway\", \"Pacific/Niue\", \"Pacific/Pago_Pago\", \"Pacific/Samoa\", \"US/Samoa\", \"Etc/GMT+10\", \"HST\", \"Pacific/Honolulu\", \"Pacific/Johnston\", \"Pacific/Rarotonga\", \"Pacific/Tahiti\", \"US/Hawaii\", \"Pacific/Marquesas\", \"America/Adak\", \"America/Atka\", \"Etc/GMT+9\", \"Pacific/Gambier\", \"US/Aleutian\", \"America/Anchorage\", \"America/Juneau\", \"America/Metlakatla\", \"America/Nome\", \"America/Sitka\", \"America/Yakutat\", \"Etc/GMT+8\", \"Pacific/Pitcairn\", \"US/Alaska\", \"America/Creston\", \"America/Dawson\", \"America/Dawson_Creek\", \"America/Ensenada\", \"America/Hermosillo\", \"America/Los_Angeles\", \"America/Phoenix\", \"America/Santa_Isabel\", \"America/Tijuana\", \"America/Vancouver\", \"America/Whitehorse\", \"Canada/Pacific\", \"Canada/Yukon\", \"Etc/GMT+7\", \"MST\", \"Mexico/BajaNorte\", \"PST8PDT\", \"US/Arizona\", \"US/Pacific\", \"US/Pacific-New\", \"America/Belize\", \"America/Boise\", \"America/Cambridge_Bay\", \"America/Chihuahua\", \"America/Costa_Rica\", \"America/Denver\", \"America/Edmonton\", \"America/El_Salvador\", \"America/Guatemala\", \"America/Inuvik\", \"America/Managua\", \"America/Mazatlan\", \"America/Ojinaga\", \"America/Regina\", \"America/Shiprock\", \"America/Swift_Current\", \"America/Tegucigalpa\", \"America/Yellowknife\", \"Canada/East-Saskatchewan\", \"Canada/Mountain\", \"Canada/Saskatchewan\", \"Etc/GMT+6\", \"MST7MDT\", \"Mexico/BajaSur\", \"Navajo\", \"Pacific/Galapagos\", \"US/Mountain\", \"America/Atikokan\", \"America/Bahia_Banderas\", \"America/Bogota\", \"America/Cancun\", \"America/Cayman\", \"America/Chicago\", \"America/Coral_Harbour\", \"America/Eirunepe\", \"America/Guayaquil\", \"America/Indiana/Knox\", \"America/Indiana/Tell_City\", \"America/Jamaica\", \"America/Knox_IN\", \"America/Lima\", \"America/Matamoros\", \"America/Menominee\", \"America/Merida\", \"America/Mexico_City\", \"America/Monterrey\", \"America/North_Dakota/Beulah\", \"America/North_Dakota/Center\", \"America/North_Dakota/New_Salem\", \"America/Panama\", \"America/Porto_Acre\", \"America/Rainy_River\", \"America/Rankin_Inlet\", \"America/Resolute\", \"America/Rio_Branco\", \"America/Winnipeg\", \"Brazil/Acre\", \"CST6CDT\", \"Canada/Central\", \"Chile/EasterIsland\", \"EST\", \"Etc/GMT+5\", \"Jamaica\", \"Mexico/General\", \"Pacific/Easter\", \"US/Central\", \"US/Indiana-Starke\", \"America/Caracas\", \"America/Anguilla\", \"America/Antigua\", \"America/Aruba\", \"America/Asuncion\", \"America/Barbados\", \"America/Blanc-Sablon\", \"America/Boa_Vista\", \"America/Campo_Grande\", \"America/Cuiaba\", \"America/Curacao\", \"America/Detroit\", \"America/Dominica\", \"America/Fort_Wayne\", \"America/Grand_Turk\", \"America/Grenada\", \"America/Guadeloupe\", \"America/Guyana\", \"America/Havana\", \"America/Indiana/Indianapolis\", \"America/Indiana/Marengo\", \"America/Indiana/Petersburg\", \"America/Indiana/Vevay\", \"America/Indiana/Vincennes\", \"America/Indiana/Winamac\", \"America/Indianapolis\", \"America/Iqaluit \", \"America/Kentucky/Louisville \", \"America/Kentucky/Monticello\", \"America/Kralendijk\", \"America/La_Paz\", \"America/Louisville \", \"America/Lower_Princes\", \"America/Manaus\", \"America/Marigot\", \"America/Martinique\", \"America/Montreal\", \"America/Montserrat\", \"America/Nassau\", \"America/New_York\", \"America/Nipigon\", \"America/Pangnirtung \", \"America/Port-au-Prince \", \"America/Port_of_Spain\", \"America/Porto_Velho\", \"America/Puerto_Rico \", \"America/Santo_Domingo \", \"America/St_Barthelemy\", \"America/St_Kitts\", \"America/St_Lucia\", \"America/St_Thomas\", \"America/St_Vincent\", \"America/Thunder_Bay\", \"America/Toronto\", \"America/Tortola\", \"America/Virgin\", \"Brazil/West\", \"Canada/Eastern\", \"Cuba\", \"EST5EDT\", \"Etc/GMT+4\", \"US/East-Indiana\", \"US/Eastern\", \"US/Michigan\", \"America/Araguaina \", \"America/Argentina/Buenos_Aires \", \"America/Argentina/Catamarca \", \"America/Argentina/ComodRivadavia \", \"America/Argentina/Cordoba \", \"America/Argentina/Jujuy \", \"America/Argentina/La_Rioja \", \"America/Argentina/Mendoza \", \"America/Argentina/Rio_Gallegos \", \"America/Argentina/Salta \", \"America/Argentina/San_Juan \", \"America/Argentina/San_Luis \", \"America/Argentina/Tucuman \", \"America/Argentina/Ushuaia\", \"America/Bahia\", \"America/Belem\", \"America/Buenos_Aires\", \"America/Catamarca\", \"America/Cayenne\", \"America/Cordoba\", \"America/Fortaleza\", \"America/Glace_Bay\", \"America/Goose_Bay\", \"America/Halifax\", \"America/Jujuy\", \"America/Maceio\", \"America/Mendoza\", \"America/Moncton\", \"America/Montevideo\", \"America/Paramaribo\", \"America/Recife\", \"America/Rosario\", \"America/Santarem\", \"America/Santiago\", \"America/Sao_Paulo\", \"America/Thule\", \"Antarctica/Palmer\", \"Antarctica/Rothera\", \"Atlantic/Bermuda\", \"Atlantic/Stanley\", \"Brazil/East\", \"Canada/Atlantic\", \"Chile/Continental\", \"Etc/GMT+3\", \"America/St_Johns\", \"Canada/Newfoundland\", \"America/Godthab\", \"America/Miquelon\", \"America/Noronha \", \"Atlantic/South_Georgia\", \"Brazil/DeNoronha\", \"Etc/GMT+2\", \"Atlantic/Cape_Verde\", \"Etc/GMT+1\", \"Africa/Abidjan\", \"Africa/Accra\", \"Africa/Bamako\", \"Africa/Banjul\", \"Africa/Bissau\", \"Africa/Conakry\", \"Africa/Dakar\", \"Africa/Freetown\", \"Africa/Lome\", \"Africa/Monrovia\", \"Africa/Nouakchott\", \"Africa/Ouagadougou\", \"Africa/Sao_Tome\", \"Africa/Timbuktu\", \"America/Danmarkshavn\", \"America/Scoresbysund\", \"Atlantic/Azores\", \"Atlantic/Reykjavik\", \"Atlantic/St_Helena\", \"Etc/GMT\", \"Etc/GMT+0\", \"Etc/GMT-0\", \"Etc/GMT0\", \"Etc/Greenwich\", \"Etc/UCT\", \"Etc/UTC\", \"Etc/Universal\", \"Etc/Zulu\", \"GMT\", \"GMT+0\", \"GMT-0\", \"GMT0\", \"Greenwich\", \"Iceland\", \"UCT\", \"UTC\", \"Universal\", \"Zulu\", \"Africa/Algiers\", \"Africa/Bangui\", \"Africa/Brazzaville\", \"Africa/Casablanca\", \"Africa/Douala\", \"Africa/El_Aaiun\", \"Africa/Kinshasa\", \"Africa/Lagos\", \"Africa/Libreville\", \"Africa/Luanda\", \"Africa/Malabo\", \"Africa/Ndjamena\", \"Africa/Niamey\", \"Africa/Porto-Novo\", \"Africa/Tunis\", \"Africa/Windhoek\", \"Atlantic/Canary\", \"Atlantic/Faeroe\", \"Atlantic/Faroe\", \"Atlantic/Madeira\", \"Eire\", \"Etc/GMT-1\", \"Europe/Belfast\", \"Europe/Dublin\", \"Europe/Guernsey\", \"Europe/Isle_of_Man\", \"Europe/Jersey\", \"Europe/Lisbon\", \"Europe/London\", \"GB\", \"GB-Eire\", \"Portugal\", \"WET\", \"Africa/Blantyre\", \"Africa/Bujumbura\", \"Africa/Cairo\", \"Africa/Ceuta\", \"Africa/Gaborone\", \"Africa/Harare\", \"Africa/Johannesburg\", \"Africa/Kigali\", \"Africa/Lubumbashi\", \"Africa/Lusaka\", \"Africa/Maputo\", \"Africa/Maseru\", \"Africa/Mbabane\", \"Africa/Tripoli\", \"Antarctica/Troll\", \"Arctic/Longyearbyen\", \"Atlantic/Jan_Mayen\", \"CET\", \"Egypt\", \"Etc/GMT-2\", \"Europe/Amsterdam\", \"Europe/Andorra\", \"Europe/Belgrade\", \"Europe/Berlin\", \"Europe/Bratislava\", \"Europe/Brussels\", \"Europe/Budapest\", \"Europe/Busingen\", \"Europe/Copenhagen\", \"Europe/Gibraltar\", \"Europe/Kaliningrad\", \"Europe/Ljubljana\", \"Europe/Luxembourg\", \"Europe/Madrid\", \"Europe/Malta\", \"Europe/Monaco\", \"Europe/Oslo\", \"Europe/Paris\", \"Europe/Podgorica\", \"Europe/Prague\", \"Europe/Rome\", \"Europe/San_Marino\", \"Europe/Sarajevo\", \"Europe/Skopje\", \"Europe/Stockholm\", \"Europe/Tirane\", \"Europe/Vaduz\", \"Europe/Vatican\", \"Europe/Vienna\", \"Europe/Warsaw\", \"Europe/Zagreb\", \"Europe/Zurich\", \"Libya\", \"MET\", \"Poland\", \"Africa/Addis_Ababa\", \"Africa/Asmara\", \"Africa/Asmera\", \"Africa/Dar_es_Salaam\", \"Africa/Djibouti\", \"Africa/Juba\", \"Africa/Kampala\", \"Africa/Khartoum\", \"Africa/Mogadishu\", \"Africa/Nairobi\", \"Antarctica/Syowa\", \"Asia/Aden\", \"Asia/Amman\", \"Asia/Baghdad\", \"Asia/Bahrain\", \"Asia/Beirut\", \"Asia/Damascus\", \"Asia/Gaza\", \"Asia/Hebron\", \"Asia/Istanbul\", \"Asia/Jerusalem\", \"Asia/Kuwait\", \"Asia/Nicosia\", \"Asia/Qatar\", \"Asia/Riyadh\", \"Asia/Tel_Aviv\", \"EET\", \"Etc/GMT-3\", \"Europe/Athens\", \"Europe/Bucharest\", \"Europe/Chisinau\", \"Europe/Helsinki\", \"Europe/Istanbul\", \"Europe/Kiev\", \"Europe/Mariehamn\", \"Europe/Minsk\", \"Europe/Moscow\", \"Europe/Nicosia\", \"Europe/Riga\", \"Europe/Simferopol\", \"Europe/Sofia\", \"Europe/Tallinn\", \"Europe/Tiraspol\", \"Europe/Uzhgorod\", \"Europe/Vilnius\", \"Europe/Volgograd\", \"Europe/Zaporozhye\", \"Indian/Antananarivo\", \"Indian/Comoro\", \"Indian/Mayotte\", \"Israel\", \"Turkey\", \"W-SU\", \"Asia/Dubai\", \"Asia/Muscat\", \"Asia/Tbilisi\", \"Asia/Yerevan\", \"Etc/GMT-4\", \"Europe/Samara\", \"Indian/Mahe\", \"Indian/Mauritius\", \"Indian/Reunion\", \"Asia/Kabul\", \"Asia/Tehran\", \"Iran\", \"Antarctica/Mawson\", \"Asia/Aqtau\", \"Asia/Aqtobe\", \"Asia/Ashgabat\", \"Asia/Ashkhabad\", \"Asia/Baku\", \"Asia/Dushanbe\", \"Asia/Karachi\", \"Asia/Oral\", \"Asia/Samarkand\", \"Asia/Tashkent\", \"Asia/Yekaterinburg\", \"Etc/GMT-5\", \"Indian/Kerguelen\", \"Indian/Maldives\", \"Asia/Calcutta\", \"Asia/Colombo\", \"Asia/Kolkata\", \"Asia/Kathmandu\", \"Asia/Katmandu\", \"Antarctica/Vostok\", \"Asia/Almaty\", \"Asia/Bishkek\", \"Asia/Dacca\", \"Asia/Dhaka\", \"Asia/Kashgar\", \"Asia/Novosibirsk\", \"Asia/Omsk\", \"Asia/Qyzylorda\", \"Asia/Thimbu\", \"Asia/Thimphu\", \"Asia/Urumqi\", \"Etc/GMT-6\", \"Indian/Chagos\", \"Asia/Rangoon\", \"Indian/Cocos\", \"Antarctica/Davis\", \"Asia/Bangkok\", \"Asia/Ho_Chi_Minh\", \"Asia/Hovd\", \"Asia/Jakarta\", \"Asia/Krasnoyarsk\", \"Asia/Novokuznetsk\", \"Asia/Phnom_Penh\", \"Asia/Pontianak\", \"Asia/Saigon\", \"Asia/Vientiane\", \"Etc/GMT-7\", \"Indian/Christmas\", \"Antarctica/Casey\", \"Asia/Brunei\", \"Asia/Chita\", \"Asia/Choibalsan\", \"Asia/Chongqing\", \"Asia/Chungking\", \"Asia/Harbin\", \"Asia/Hong_Kong\", \"Asia/Irkutsk\", \"Asia/Kuala_Lumpur\", \"Asia/Kuching\", \"Asia/Macao\", \"Asia/Macau\", \"Asia/Makassar\", \"Asia/Manila\", \"Asia/Shanghai\", \"Asia/Singapore\", \"Asia/Taipei\", \"Asia/Ujung_Pandang\", \"Asia/Ulaanbaatar\", \"Asia/Ulan_Bator\", \"Australia/Perth\", \"Australia/West\", \"Etc/GMT-8\", \"Hongkong\", \"PRC\", \"ROC\", \"Singapore\", \"Australia/Eucla\", \"Asia/Dili\", \"Asia/Jayapura\", \"Asia/Khandyga\", \"Asia/Pyongyang\", \"Asia/Seoul\", \"Asia/Tokyo\", \"Asia/Yakutsk\", \"Etc/GMT-9\", \"Japan\", \"Pacific/Palau\", \"ROK\", \"Australia/Adelaide \", \"Australia/Broken_Hill\", \"Australia/Darwin\", \"Australia/North\", \"Australia/South\", \"Australia/Yancowinna \", \"Antarctica/DumontDUrville\", \"Asia/Magadan\", \"Asia/Sakhalin\", \"Asia/Ust-Nera\", \"Asia/Vladivostok\", \"Australia/ACT\", \"Australia/Brisbane\", \"Australia/Canberra\", \"Australia/Currie\", \"Australia/Hobart\", \"Australia/Lindeman\", \"Australia/Melbourne\", \"Australia/NSW\", \"Australia/Queensland\", \"Australia/Sydney\", \"Australia/Tasmania\", \"Australia/Victoria\", \"Etc/GMT-10\", \"Pacific/Chuuk\", \"Pacific/Guam\", \"Pacific/Port_Moresby\", \"Pacific/Saipan\", \"Pacific/Truk\", \"Pacific/Yap\", \"Australia/LHI\", \"Australia/Lord_Howe\", \"Antarctica/Macquarie\", \"Asia/Srednekolymsk\", \"Etc/GMT-11\", \"Pacific/Bougainville\", \"Pacific/Efate\", \"Pacific/Guadalcanal\", \"Pacific/Kosrae\", \"Pacific/Noumea\", \"Pacific/Pohnpei\", \"Pacific/Ponape\", \"Pacific/Norfolk\", \"Antarctica/McMurdo\", \"Antarctica/South_Pole\", \"Asia/Anadyr\", \"Asia/Kamchatka\", \"Etc/GMT-12\", \"Kwajalein\", \"NZ\", \"Pacific/Auckland\", \"Pacific/Fiji\", \"Pacific/Funafuti\", \"Pacific/Kwajalein\", \"Pacific/Majuro\", \"Pacific/Nauru\", \"Pacific/Tarawa\", \"Pacific/Wake\", \"Pacific/Wallis\", \"NZ-CHAT\", \"Pacific/Chatham\", \"Etc/GMT-13\", \"Pacific/Apia\", \"Pacific/Enderbury\", \"Pacific/Fakaofo\", \"Pacific/Tongatapu\", \"Etc/GMT-14\", \"Pacific/Kiritimati\"",
						},
					}))
				})
			})

			Context("when recurring_schedule and specific_date schedules both are missing", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"schedules":{
						"timezone":"Asia/Kolkata"
					 }
				}`
				})
				It("should fail", func() {
					Expect(errResult).To(Equal([]PolicyValidationErrors{
						{
							Context:     "(root).schedules",
							Description: "Must validate at least one schema (anyOf)",
						},
						{
							Context:     "(root).schedules",
							Description: "recurring_schedule is required",
						},
					}))
				})
			})

			Context("when only recurring_schedules are present", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"schedules":{
						"timezone":"Asia/Kolkata",
						"recurring_schedule":[
							{
							   "start_time":"10:00",
							   "end_time":"18:00",
							   "days_of_week":[
								  1,
								  2,
								  3
							   ],
							   "instance_min_count":1,
							   "instance_max_count":10,
							   "initial_min_instance_count":5
							}
						]
					 }
				}`
				})
				It("should succeed", func() {
					Expect(errResult).To(BeNil())
					Expect(policyJson).To(MatchJSON(policyString))
				})
			})

			Context("recurring_schedule", func() {
				Context("when start_time is after end_time", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"08:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_max_count":5,
										"instance_min_count":1,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "recurring_schedule[0].start_time is same or after recurring_schedule[0].end_time",
							},
						}))
					})
				})
				Context("when only start_date is present", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date":"2099-01-01",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_max_count":5,
										"instance_min_count":1,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should succeed", func() {
						Expect(errResult).To(BeNil())
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})
				Context("when only end_date is present", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"end_date":"2099-01-01",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_max_count":5,
										"instance_min_count":1,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should succeed", func() {
						Expect(errResult).To(BeNil())
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})
				Context("when only start_date is present and is same as current date", func() {
					BeforeEach(func() {
						timezone := "Asia/Kolkata"
						location, _ := time.LoadLocation(timezone)

						policyString = fmt.Sprintf(`{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"%v",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date":"%v",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_max_count":5,
										"instance_min_count":1,
										"initial_min_instance_count":3
									}
								]
							}
						}`, timezone, time.Now().In(location).Format(DateLayout))
					})
					It("should succeed", func() {
						Expect(errResult).To(BeNil())
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})
				Context("when only end_date is present and is same as current date", func() {
					BeforeEach(func() {
						timezone := "Asia/Kolkata"
						location, _ := time.LoadLocation(timezone)

						policyString = fmt.Sprintf(`{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"%v",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"end_date":"%v",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_max_count":5,
										"instance_min_count":1,
										"initial_min_instance_count":3
									}
								]
							}
						}`, timezone, time.Now().In(location).Format(DateLayout))
					})
					It("should succeed", func() {
						Expect(errResult).To(BeNil())
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})
				Context("when start_date is after end_date", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date":"2099-01-01",
										"end_date":"2098-01-01",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_max_count":5,
										"instance_min_count":1,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "recurring_schedule[0].start_date is after recurring_schedule[0].end_date",
							},
						}))
					})
				})
				Context("when start_date is before current_date", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date":"2015-01-01",
										"end_date":"2098-01-01",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_max_count":5,
										"instance_min_count":1,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "recurring_schedule[0].start_date is before recurring_schedule[0].current_date",
							},
						}))
					})
				})
				Context("when instance_min_count is missing", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_max_count":5,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "instance_min_count is required",
							},
						}))
					})
				})
				Context("when instance_max_count is missing", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "instance_max_count is required",
							},
						}))
					})
				})
				Context("when initial_min_instance_count is missing", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"instance_max_count":5
									}
								]
							}
						}
					`
					})
					It("should succeed", func() {
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})
				Context("when instance_min_count is greater than instance_max_count", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":10,
										"instance_max_count":5
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0.instance_min_count",
								Description: "recurring_schedule[0].instance_min_count 10 is higher than recurring_schedule[0].instance_max_count 5",
							},
						}))
					})
				})
				Context("when initial_min_instance_count is smaller than instance_min_count", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":1
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0.initial_min_instance_count",
								Description: "recurring_schedule[0].initial_min_instance_count 1 is smaller than recurring_schedule[0].instance_min_count 2",
							},
						}))
					})
				})
				Context("when initial_min_instance_count is greater than instance_max_count", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":8
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0.initial_min_instance_count",
								Description: "recurring_schedule[0].initial_min_instance_count 8 is greater than recurring_schedule[0].instance_max_count 5",
							},
						}))
					})
				})
				Context("when overlapping time range in overlapping days_of_week", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"days_of_week":[
											2,
											4,
											6
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "recurring_schedule[0] and recurring_schedule[1] are overlapping",
							},
						}))
					})
				})
				Context("when both days_of_week and days_of_month are present", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"days_of_month":[
											1,
											2,
											3
										],
										"instance_max_count":5,
										"instance_min_count":1,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "Must validate one and only one schema (oneOf)",
							},
						}))
					})
				})
				Context("when overlapping time range in non-overlapping days_of_week", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"days_of_week":[
											4,
											6
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should succeed", func() {
						Expect(errResult).To(BeNil())
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})
				Context("when overlapping time range in overlapping days_of_week in non-overlapping date range", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date": "2091-01-01",
										"end_date": "2092-02-02",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"start_date": "2098-01-01",
										"end_date": "2099-02-02",
										"days_of_week":[
											2,
											4,
											6
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should succeed", func() {
						Expect(errResult).To(BeNil())
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})
				Context("when overlapping time range in overlapping days_of_week in overlapping date range", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date": "2091-01-01",
										"end_date": "2099-03-02",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"start_date": "2098-01-01",
										"end_date": "2099-02-02",
										"days_of_week":[
											2,
											4,
											6
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "recurring_schedule[0] and recurring_schedule[1] are overlapping",
							},
						}))
					})
				})

				Context("when overlapping time range in overlapping days_of_week in start_date date of second schedule is before end_date of second schedule with no end_date", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date": "2091-01-01",
										"end_date": "2099-03-02",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"start_date": "2080-01-01",
										"days_of_week":[
											2,
											4,
											6
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "recurring_schedule[0] and recurring_schedule[1] are overlapping",
							},
						}))
					})
				})

				Context("when overlapping time range in overlapping days_of_week with both schedules having no end_date", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date": "2091-01-01",
										"days_of_week":[
											1,
											2,
											3
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"start_date": "2080-01-01",
										"days_of_week":[
											2,
											4,
											6
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "recurring_schedule[0] and recurring_schedule[1] are overlapping",
							},
						}))
					})
				})

				Context("when overlapping time range in overlapping days_of_month", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_month":[
											11,
											22
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"days_of_month":[
											22,
											23
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "recurring_schedule[0] and recurring_schedule[1] are overlapping",
							},
						}))
					})
				})

				Context("when overlapping time range in non-overlapping days_of_month", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"days_of_month":[
											11,
											12
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"days_of_month":[
											22,
											23
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should succeed", func() {
						Expect(errResult).To(BeNil())
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})

				Context("when overlapping time range in overlapping days_of_month in non-overlapping date range", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date": "2091-01-01",
										"end_date": "2092-02-02",
										"days_of_month":[
											11,
											12
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"start_date": "2098-01-01",
										"end_date": "2099-02-02",
										"days_of_month":[
											12,
											23
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should succeed", func() {
						Expect(errResult).To(BeNil())
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})

				Context("when overlapping time range in overlapping days_of_month in overlapping date range", func() {
					BeforeEach(func() {
						policyString = `{
							"instance_max_count":4,
							"instance_min_count":1,
							"schedules":{
								"timezone":"Asia/Kolkata",
								"recurring_schedule":[
									{
										"start_time":"10:00",
										"end_time":"18:00",
										"start_date": "2091-01-01",
										"end_date": "2099-02-02",
										"days_of_month":[
											11,
											12
										],
										"instance_min_count":2,
										"instance_max_count":5,
										"initial_min_instance_count":3
									},
									{
										"start_time":"08:00",
										"end_time":"20:00",
										"start_date": "2098-01-01",
										"end_date": "2099-03-02",
										"days_of_month":[
											12,
											23
										],
										"instance_min_count":2,
										"instance_max_count":7,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0",
								Description: "recurring_schedule[0] and recurring_schedule[1] are overlapping",
							},
						}))
					})
				})

			})

			Context("when only specific_date schedules are present", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"schedules":{
						"timezone":"Asia/Kolkata",
						"specific_date":[
						   {
							  "start_date_time":"2099-01-04T20:00",
							  "end_date_time":"2099-02-19T23:15",
							  "instance_min_count":2,
							  "instance_max_count":5,
							  "initial_min_instance_count":3
						   }
						]
					 }
				}`
				})
				It("should succeed", func() {
					Expect(errResult).To(BeNil())
					Expect(policyJson).To(MatchJSON(policyString))
				})
			})

			Context("specific_date schedule", func() {

				Context("when start_date_time is missing", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "end_date_time":"2099-02-19T23:15",
								  "instance_min_count":2,
								  "instance_max_count":5,
								  "initial_min_instance_count":3
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0",
								Description: "start_date_time is required",
							},
						}))
					})
				})
				Context("when start_date_time is not in correct format", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2099-01-04T20:00+530",
								  "end_date_time":"2099-02-19T23:15",
								  "instance_min_count":2,
								  "instance_max_count":5,
								  "initial_min_instance_count":3
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0.start_date_time",
								Description: "Does not match pattern '^2[0-9]{3}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])T(2[0-3]|1[0-9]|0[0-9]):([0-5][0-9])$'",
							},
						}))
					})
				})

				Context("when end_date_time is missing", func() {
					BeforeEach(func() {
						policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"schedules":{
						"timezone":"Asia/Kolkata",
						"specific_date":[
						   {
							  "start_date_time":"2099-02-19T23:15",
							  "instance_min_count":2,
							  "instance_max_count":5,
							  "initial_min_instance_count":3
						   }
						]
					 }
				}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0",
								Description: "end_date_time is required",
							},
						}))
					})
				})
				Context("when end_date_time is not in correct format", func() {
					BeforeEach(func() {
						policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"schedules":{
						"timezone":"Asia/Kolkata",
						"specific_date":[
						   {
							  "start_date_time":"2099-01-04T20:00",
							  "end_date_time":"2099-02-19T23:15+530",
							  "instance_min_count":2,
							  "instance_max_count":5,
							  "initial_min_instance_count":3
						   }
						]
					 }
				}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0.end_date_time",
								Description: "Does not match pattern '^2[0-9]{3}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])T(2[0-3]|1[0-9]|0[0-9]):([0-5][0-9])$'",
							},
						}))
					})
				})

				Context("when instance_min_count is missing", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2099-01-04T20:00",
								  "end_date_time":"2099-02-19T23:15",
								  "instance_max_count":5,
								  "initial_min_instance_count":3
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0",
								Description: "instance_min_count is required",
							},
						}))
					})
				})

				Context("when instance_max_count is missing", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2099-01-04T20:00",
								  "end_date_time":"2099-02-19T23:15",
								  "instance_min_count":2,
								  "initial_min_instance_count":3
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0",
								Description: "instance_max_count is required",
							},
						}))
					})
				})

				Context("when initial_instance_min_count is missing", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2099-01-04T20:00",
								  "end_date_time":"2099-02-19T23:15",
								  "instance_min_count":2,
								  "instance_max_count":5
							   }
							]
						 }
					}`
					})
					It("should succeed", func() {
						Expect(errResult).To(BeNil())
						Expect(policyJson).To(MatchJSON(policyString))
					})
				})

				Context("when instance_min_count is greater than instance_max_count", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2099-01-04T20:00",
								  "end_date_time":"2099-02-19T23:15",
								  "instance_min_count":5,
								  "instance_max_count":2
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0.instance_min_count",
								Description: "specific_date[0].instance_min_count 5 is higher than specific_date[0].instance_max_count 2",
							},
						}))
					})
				})

				Context("when initial_instance_min_count is smaller than instance_min_count", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2099-01-04T20:00",
								  "end_date_time":"2099-02-19T23:15",
								  "instance_min_count":2,
								  "instance_max_count":5,
								  "initial_min_instance_count":1
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0.initial_min_instance_count",
								Description: "specific_date[0].initial_min_instance_count 1 is smaller than specific_date[0].instance_min_count 2",
							},
						}))
					})
				})

				Context("when inItial_instance_min_count is greater than instance_max_count", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2099-01-04T20:00",
								  "end_date_time":"2099-02-19T23:15",
								  "instance_min_count":2,
								  "instance_max_count":5,
								  "initial_min_instance_count":8
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0.initial_min_instance_count",
								Description: "specific_date[0].initial_min_instance_count 8 is greater than specific_date[0].instance_max_count 5",
							},
						}))
					})
				})

				Context("when start_date_time is before current_date_time", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2010-01-04T20:00",
								  "end_date_time":"2099-02-19T23:15",
								  "instance_min_count":2,
								  "instance_max_count":5,
								  "initial_min_instance_count":3
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0",
								Description: "specific_date[0].start_date_time is before current date time",
							},
						}))
					})
				})
				Context("when end_date_time is before start_date_time", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2099-01-04T20:00",
								  "end_date_time":"2098-02-19T23:15",
								  "instance_min_count":2,
								  "instance_max_count":5,
								  "initial_min_instance_count":3
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0",
								Description: "specific_date[0].start_date_time is after specific_date[0].end_date_time",
							},
						}))
					})
				})

				Context("when there's an overlap between two specific_date schedules", func() {
					BeforeEach(func() {
						policyString = `{
						"instance_max_count":4,
						"instance_min_count":1,
						"schedules":{
							"timezone":"Asia/Kolkata",
							"specific_date":[
							   {
								  "start_date_time":"2097-01-04T20:00",
								  "end_date_time":"2098-02-19T23:15",
								  "instance_min_count":2,
								  "instance_max_count":5,
								  "initial_min_instance_count":3
							   },
							   {
								"start_date_time":"2090-01-03T20:00",
								"end_date_time":"2099-02-19T23:15",
								"instance_min_count":5,
								"instance_max_count":10,
								"initial_min_instance_count":7
							 	}
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(errResult).To(Equal([]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0",
								Description: "specific_date[0]:{start_date_time: 2097-01-04T20:00, end_date_time: 2097-01-04T20:00} and specific_date[1]:{start_date_time: 2090-01-03T20:00, end_date_time: 2090-01-03T20:00} are overlapping",
							},
						}))
					})
				})
			})

		})
	})
	Context("Binding Configuration with custom metrics strategy", func() {
		When("custom_metrics is missing", func() {
			BeforeEach(func() {
				policyString = `{
                "instance_max_count":4,
                "instance_min_count":1,
                "scaling_rules":[
                    {
                        "metric_type":"memoryutil",
                        "breach_duration_secs": 300,
                        "threshold":90,
                        "operator":">=",
                        "adjustment": "+1"
                    }
                ]
            }`
			})
			It("should not fail", func() {
				Expect(errResult).To(BeNil())
			})
		})
		When("allow_from is missing in metric_submission_strategy", func() {
			BeforeEach(func() {
				policyString = `{
                "instance_max_count":4,
                "instance_min_count":1,
                "scaling_rules":[
                    {
                        "metric_type":"memoryutil",
                        "breach_duration_secs": 300,
                        "threshold":90,
                        "operator":">=",
                        "adjustment": "+1"
                    }
                ],
                "configuration": {
                    "custom_metrics": {
                        "metric_submission_strategy": {}
                    }
                }
            }`
			})
			It("should fail", func() {
				Expect(errResult).To(Equal([]PolicyValidationErrors{
					{
						Context:     "(root).configuration.custom_metrics.metric_submission_strategy",
						Description: "allow_from is required",
					},
				}))
			})
		})
		When("allow_from is invalid in metric_submission_strategy", func() {
			BeforeEach(func() {
				policyString = `{
                "instance_max_count":4,
                "instance_min_count":1,
                "scaling_rules":[
                    {
                        "metric_type":"memoryutil",
                        "breach_duration_secs": 300,
                        "threshold":90,
                        "operator":">=",
                        "adjustment": "+1"
                    }
                ],
                "configuration": {
                    "custom_metrics": {
                        "metric_submission_strategy": {
                            "allow_from": "invalid_value"
                        }
                    }
                }
            }`
			})
			It("should fail", func() {
				Expect(errResult).To(Equal([]PolicyValidationErrors{
					{
						Context:     "(root).configuration.custom_metrics.metric_submission_strategy.allow_from",
						Description: "configuration.custom_metrics.metric_submission_strategy.allow_from must be one of the following: \"bound_app\"",
					},
				}))
			})
		})
		When("allow_from is valid in metric_submission_strategy", func() {
			BeforeEach(func() {
				policyString = `{
                "instance_max_count":4,
                "instance_min_count":1,
                "scaling_rules":[
                    {
                        "metric_type":"memoryutil",
                        "breach_duration_secs": 300,
                        "threshold":90,
                        "operator":">=",
                        "adjustment": "+1"
                    }
                ],
                "configuration": {
                    "custom_metrics": {
                        "metric_submission_strategy": {
                            "allow_from": "bound_app"
                        }
                    }
                }
            }`
			})
			It("should succeed", func() {

				Expect(errResult).To(BeNil())
			})
		})
	})

	Context("If the policy string is valid json with required parameters", func() {
		BeforeEach(func() {
			policyString = `
			{
				"instance_min_count":1,
				"instance_max_count":4,
				"scaling_rules":[
				   {
					  "metric_type":"memoryused",
					  "breach_duration_secs":600,
					  "threshold":30,
					  "operator":"<",
					  "cool_down_secs":300,
					  "adjustment":"-1"
				   },
				   {
					  "metric_type":"memoryutil",
					  "breach_duration_secs":600,
					  "threshold":90,
					  "operator":">=",
					  "cool_down_secs":300,
					  "adjustment":"+1"
				   },
				   {
					"metric_type":"jobquelength",
					"breach_duration_secs":600,
					"threshold":90,
					"operator":">=",
					"cool_down_secs":300,
					"adjustment":"+1"
				 }
				],
				"schedules":{
				   "timezone":"Asia/Kolkata",
				   "recurring_schedule":[
					  {
						 "start_time":"10:00",
						 "end_time":"18:00",
						 "days_of_week":[
							1,
							2,
							3
						 ],
						 "instance_min_count":1,
						 "instance_max_count":10,
						 "initial_min_instance_count":5
					  },
					  {
						 "start_date":"2099-06-27",
						 "end_date":"2099-07-23",
						 "start_time":"11:00",
						 "end_time":"19:30",
						 "days_of_month":[
							5,
							15,
							25
						 ],
						 "instance_min_count":3,
						 "instance_max_count":10,
						 "initial_min_instance_count":5
					  },
					  {
						 "start_time":"10:00",
						 "end_time":"18:00",
						 "days_of_week":[
							4,
							5,
							6
						 ],
						 "instance_min_count":1,
						 "instance_max_count":10
					  },
					  {
						 "start_time":"11:00",
						 "end_time":"19:30",
						 "days_of_month":[
							10,
							20,
							30
						 ],
						 "instance_min_count":1,
						 "instance_max_count":10
					  }
				   ],
				   "specific_date":[
					  {
						 "start_date_time":"2099-06-02T10:00",
						 "end_date_time":"2099-06-15T13:59",
						 "instance_min_count":1,
						 "instance_max_count":4,
						 "initial_min_instance_count":2
					  },
					  {
						 "start_date_time":"2098-01-04T20:00",
						 "end_date_time":"2098-02-19T23:15",
						 "instance_min_count":2,
						 "instance_max_count":5,
						 "initial_min_instance_count":3
					  }
				   ]
				}
			  }
			`
		})
		It("It should succeed", func() {
			Expect(errResult).To(BeNil())
			Expect(policyJson).To(MatchJSON(policyString))
		})
	})
})
