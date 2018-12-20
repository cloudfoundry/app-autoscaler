package policyvalidator_test

import (
	. "autoscaler/api/policyvalidator"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyValidator", func() {
	var (
		policyValidator *PolicyValidator
		err             error
		policyString    string
	)
	BeforeEach(func() {
		policyValidator = NewPolicyValidator("./policy_json.schema.json")
	})
	JustBeforeEach(func() {
		err = policyValidator.ValidatePolicy(policyString)
	})
	Context("Policy Schema &  Validation", func() {
		Context("when invalid json", func() {
			BeforeEach(func() {
				policyString = `{
					"instance_min_count:1,
				}`
			})
			It("should fail", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("invalid character '\\n' in string literal"))
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
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("[{\"context\": \"(root)\", \"description\": \"instance_min_count is required\"}]"))
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
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("[{\"context\": \"(root)\", \"description\": \"instance_max_count is required\"}]"))
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
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("[{\"context\": \"(root).instance_min_count\", \"description\": \"instance_min_count 10 is higher or equal to instance_max_count 4\"}]"))
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
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("[{\"context\": \"(root)\", \"description\": \"Must validate at least one schema (anyOf)\"},{\"context\": \"(root)\", \"description\": \"scaling_rules is required\"}]"))
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
				Expect(err).NotTo(HaveOccurred())
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0\", \"description\": \"metric_type is required\"}]"))
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0.metric_type\", \"description\": \"Does not match pattern '^[a-zA-Z0-9_]+$'\"}]"))
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0\", \"description\": \"threshold is required\"}]"))
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0.threshold\", \"description\": \"Invalid type. Expected: integer, given: number\"}]"))
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0\", \"description\": \"operator is required\"}]"))
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0.operator\", \"description\": \"scaling_rules.0.operator must be one of the following: \"\\u003c\", \"\\u003e\", \"\\u003c=\", \"\\u003e=\"\"}]"))
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0\", \"description\": \"adjustment is required\"}]"))
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0.adjustment\", \"description\": \"Does not match pattern '^[-+][1-9]+[0-9]*$'\"}]"))
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
					Expect(err).NotTo(HaveOccurred())
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0.breach_duration_secs\", \"description\": \"Invalid type. Expected: integer, given: string\"}]"))
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
				It("should succeed", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0.breach_duration_secs\", \"description\": \"Must be greater than or equal to 60\"}]"))
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
				It("should succeed", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0.breach_duration_secs\", \"description\": \"Must be less than or equal to 3600\"}]"))
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
					Expect(err).NotTo(HaveOccurred())
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
				It("should succeed", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0.cool_down_secs\", \"description\": \"Must be greater than or equal to 60\"}]"))
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
				It("should succeed", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).scaling_rules.0.cool_down_secs\", \"description\": \"Must be less than or equal to 3600\"}]"))
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).schedules\", \"description\": \"timezone is required\"}]"))
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
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("[{\"context\": \"(root).schedules\", \"description\": \"Must validate at least one schema (anyOf)\"},{\"context\": \"(root).schedules\", \"description\": \"recurring_schedule is required\"}]"))
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
					Expect(err).NotTo(HaveOccurred())
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"start_time is after end_time\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"start_date is after end_date\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"start_date is before current_date\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"instance_min_count is required\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"instance_max_count is required\"}]"))
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
						Expect(err).ToNot(HaveOccurred())
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
										"instance_max_count":5,
										"initial_min_instance_count":3
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0.instance_min_count\", \"description\": \"instance_min_count 10 is higher or equal to instance_max_count 10\"}]"))
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
										"initial_min_instance_count":7
									}
								]
							}
						}
					`
					})
					It("should fail", func() {
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0.initial_min_instance_count\", \"description\": \"initial_min_instance_count 7 is greater than instance_max_count 2\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"recurring_schedule[0] and recurring_schedule[1] are overlapping\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"Must validate one and only one schema (oneOf)\"}]"))
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
						Expect(err).ToNot(HaveOccurred())
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
						Expect(err).ToNot(HaveOccurred())
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"recurring_schedule[0] and recurring_schedule[1] are overlapping\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"recurring_schedule[0] and recurring_schedule[1] are overlapping\"}]"))
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
						Expect(err).NotTo(HaveOccurred())
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
						Expect(err).NotTo(HaveOccurred())
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.recurring_schedule.0\", \"description\": \"recurring_schedule[0] and recurring_schedule[1] are overlapping\"}]"))
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
					Expect(err).NotTo(HaveOccurred())
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0\", \"description\": \"start_date_time is required\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0.start_date_time\", \"description\": \"Does not match pattern '^2[0-9]{3}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])T(2[0-3]|1[0-9]|0[0-9]):([0-5][0-9])$'\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0\", \"description\": \"end_date_time is required\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0.end_date_time\", \"description\": \"Does not match pattern '^2[0-9]{3}-(0[1-9]|1[0-2])-(0[1-9]|[1-2][0-9]|3[0-1])T(2[0-3]|1[0-9]|0[0-9]):([0-5][0-9])$'\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0\", \"description\": \"instance_min_count is required\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0\", \"description\": \"instance_max_count is required\"}]"))
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
						Expect(err).NotTo(HaveOccurred())
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0.instance_min_count\", \"description\": \"instance_min_count 5 is higher or equal to instance_max_count 2\"}]"))
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
								  "initial_min_instance_count":7
							   }
							]
						 }
					}`
					})
					It("should fail", func() {
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0.initial_min_instance_count\", \"description\": \"initial_min_instance_count 7 is greater than instance_max_count 2\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0\", \"description\": \"start_date_time is before current_date_time\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0\", \"description\": \"start_date_time is after end_date_time\"}]"))
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
						Expect(err).To(HaveOccurred())
						Expect(err).To(MatchError("[{\"context\": \"(root).schedules.specific_date.0\", \"description\": \"specific_date[0]:{start_date_time: 2097-01-04T20:00, end_date_time: 2097-01-04T20:00} and specific_date[1]:{start_date_time: 2090-01-03T20:00, end_date_time: 2090-01-03T20:00} are overlapping\"}]"))
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
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
