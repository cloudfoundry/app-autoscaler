package policyvalidator_test

import (
	. "autoscaler/api/policyvalidator"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyValidator", func() {
	var (
		policyValidator *PolicyValidator
		errResult       *[]PolicyValidationErrors
		valid           bool
		policyString    string
	)
	BeforeEach(func() {
		policyValidator = NewPolicyValidator("./policy_json.schema.json")
	})
	JustBeforeEach(func() {
		errResult, valid = policyValidator.ValidatePolicy(policyString)
	})
	Context("Policy Schema &  Validation", func() {
		Context("when invalid json", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_min_count:1,
				}`
			})
			It("should fail", func() {
				Expect(valid).To(BeFalse())
				Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
				Expect(valid).To(BeFalse())
				Expect(errResult).To(Equal(&[]PolicyValidationErrors{
					{
						Context:     "(root)",
						Description: "instance_min_count is required",
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
				Expect(valid).To(BeFalse())
				Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
				Expect(valid).To(BeFalse())
				Expect(errResult).To(Equal(&[]PolicyValidationErrors{
					{
						Context:     "(root).instance_min_count",
						Description: "instance_min_count 10 is higher or equal to instance_max_count 4",
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
				Expect(valid).To(BeFalse())
				Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
				Expect(valid).To(BeTrue())
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.metric_type",
							Description: "Does not match pattern '^[a-zA-Z0-9_]+$'",
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.threshold",
							Description: "Invalid type. Expected: integer, given: number",
						},
					}))
				})
			})

			Context("when threshold for memoryused is less than 0", func() {
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type memoryused should be greater than 0",
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
					Expect(valid).To(BeTrue())
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type memoryutil should be greater than 0 and less than equal to 100",
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type memoryutil should be greater than 0 and less than equal to 100",
						},
					}))
				})
			})

			Context("when threshold for responsetime is less than 0", func() {
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type responsetime should be greater than 0",
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
					Expect(valid).To(BeTrue())
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type throughput should be greater than 0",
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
					Expect(valid).To(BeTrue())
				})
			})

			Context("when threshold for cpu is less than 0", func() {
				BeforeEach(func() {
					policyString = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"cpu",
						"breach_duration_secs":600,
						"threshold": -90,
						"operator":">=",
						"cool_down_secs":300,
						"adjustment":"+1"
					}]
				}`
				})
				It("should fail", func() {
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type cpu should be greater than 0 and less than equal to 100",
						},
					}))
				})
			})

			Context("when threshold for cpu is greater than 100", func() {
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0",
							Description: "scaling_rules[0].threshold for metric_type cpu should be greater than 0 and less than equal to 100",
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.adjustment",
							Description: "Does not match pattern '^[-+][1-9]+[0-9]*$'",
						},
					}))
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
					Expect(valid).To(BeTrue())
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).scaling_rules.0.breach_duration_secs",
							Description: "Invalid type. Expected: integer, given: string",
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeTrue())
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
						{
							Context:     "(root).schedules",
							Description: "timezone is required",
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
					Expect(valid).To(BeFalse())
					Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
							{
								Context:     "(root).schedules.recurring_schedule.0.instance_min_count",
								Description: "recurring_schedule[0].instance_min_count 10 is higher or equal to recurring_schedule[0].instance_max_count 5",
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
					Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeTrue())
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0.instance_min_count",
								Description: "specific_date[0].instance_min_count 5 is higher or equal to specific_date[0].instance_max_count 2",
							},
						}))
					})
				})

				Context("when intial_instance_min_count is smaller than instance_min_count", func() {
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
							{
								Context:     "(root).schedules.specific_date.0.initial_min_instance_count",
								Description: "specific_date[0].initial_min_instance_count 1 is smaller than specific_date[0].instance_min_count 2",
							},
						}))
					})
				})

				Context("when intial_instance_min_count is greater than instance_max_count", func() {
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
						Expect(valid).To(BeFalse())
						Expect(errResult).To(Equal(&[]PolicyValidationErrors{
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
			Expect(valid).To(BeTrue())
		})
	})
})
