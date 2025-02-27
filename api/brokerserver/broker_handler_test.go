package brokerserver_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker"

	"code.cloudfoundry.org/brokerapi/v13/handlers"
	"github.com/go-logr/logr"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"github.com/onsi/gomega/ghttp"

	"code.cloudfoundry.org/lager/v3/lagertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BrokerHandler", func() {
	var (
		bindingdb       *fakes.FakeBindingDB
		policydb        *fakes.FakePolicyDB
		fakeCredentials *fakes.FakeCredentials
		fakePlanChecker *fakes.FakePlanChecker

		handler          handlers.APIHandler
		autoscalerBroker *broker.Broker
		resp             *httptest.ResponseRecorder
		req              *http.Request
	)

	BeforeEach(func() {
		bindingdb = &fakes.FakeBindingDB{}
		policydb = &fakes.FakePolicyDB{}
		resp = httptest.NewRecorder()
		fakeCredentials = &fakes.FakeCredentials{}
		fakePlanChecker = nil
	})

	JustBeforeEach(func() {
		autoscalerBroker = broker.New(lagertest.NewTestLogger("testbroker"), conf, bindingdb, policydb, services, fakeCredentials)
		handler = handlers.NewApiHandler(autoscalerBroker, slog.New(logr.ToSlogHandler(GinkgoLogr)))
		if fakePlanChecker != nil {
			autoscalerBroker.PlanChecker = fakePlanChecker
		}
	})

	Describe("GetBrokerCatalog", func() {
		JustBeforeEach(func() {
			req, _ = http.NewRequest(http.MethodGet, "", bytes.NewReader([]byte("")))
			handler.Catalog(resp, req)
		})
		Context("When getBrokerCatalog is called", func() {
			It("gets the catalog json", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(resp.Body.Bytes()).To(MatchJSON(catalogBytes))
			})
		})
	})

	Describe("CreateServiceInstance", func() {
		var err error
		var instanceCreationReqBody *models.InstanceCreationRequestBody
		var body []byte
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			req.SetPathValue("instance_id", testInstanceId)
			handler.Provision(resp, req)
		})
		BeforeEach(func() {
			instanceCreationReqBody = &models.InstanceCreationRequestBody{
				OrgGUID:   testOrgId,
				SpaceGUID: testSpaceId,
				BrokerCommonRequestBody: models.BrokerCommonRequestBody{
					ServiceID: "autoscaler-guid",
					PlanID:    "autoscaler-free-plan-id",
					BrokerContext: models.BrokerContext{
						OrgGUID:   testOrgId,
						SpaceGUID: testSpaceId,
					},
				},
			}
		})
		Context("When request body is not a valid json", func() {
			BeforeEach(func() {
				body = []byte("")
			})
			It("fails with 422", func() {
				Expect(resp.Code).To(Equal(http.StatusUnprocessableEntity))
			})
		})
		Context("When mandatory parameters are not provided", func() {
			Context("When OrgGUID is not provided", func() {
				BeforeEach(func() {
					instanceCreationReqBody.OrgGUID = ""
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})
			Context("When SpaceGUID is not provided", func() {
				BeforeEach(func() {
					instanceCreationReqBody.SpaceGUID = ""
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})
			Context("When ServiceID is not provided", func() {
				BeforeEach(func() {
					instanceCreationReqBody.ServiceID = ""
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})
			Context("When PlanID is not provided", func() {
				BeforeEach(func() {
					instanceCreationReqBody.PlanID = ""
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})

		})

		Context("When all parameters are present", func() {
			Context("When database CreateServiceInstance call returns ErrAlreadyExists", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
					bindingdb.CreateServiceInstanceReturns(db.ErrAlreadyExists)
					conf.DashboardRedirectURI = ""
				})
				It("succeeds with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
					Expect(resp.Body.String()).To(MatchJSON(`{}`))
				})
			})

			Context("When dashboard redirect uri is present in config and database CreateServiceInstance call returns ErrAlreadyExists", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
					bindingdb.CreateServiceInstanceReturns(db.ErrAlreadyExists)
					conf.DashboardRedirectURI = "https://service-dashboard-url.com"
				})
				It("succeeds with 200 and returns dashboard_url", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
					Expect(resp.Body.Bytes()).To(MatchJSON("{\"dashboard_url\":\"https://service-dashboard-url.com/manage/an-instance-id\"}"))
				})
			})

			Context("When database CreateServiceInstance call returns ErrConflict", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
					bindingdb.CreateServiceInstanceReturns(db.ErrConflict)
				})
				It("fails with 409", func() {
					Expect(resp.Code).To(Equal(http.StatusConflict))
				})
			})

			Context("When database CreateServiceInstance call returns error other than ErrAlreadyExists", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
					bindingdb.CreateServiceInstanceReturns(fmt.Errorf("some sql error"))
				})
				It("fails with 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError))
					Expect(resp.Body.String()).To(MatchJSON(`{"description":"error creating service instance"}`))
				})
			})

			Context("When dashboard redirect uri is present in config", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
					conf.DashboardRedirectURI = "https://service-dashboard-url.com"
				})
				It("succeeds with 201 and returns dashboard_url", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated))
					Expect(resp.Body.Bytes()).To(MatchJSON(`{"dashboard_url":"https://service-dashboard-url.com/manage/an-instance-id"}`))
				})
			})
			Context("When all mandatory parameters are present", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("succeeds with 201", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated))
				})
			})
			Context("When an invalid default policy is present", func() {
				BeforeEach(func() {
					invalidDefaultPolicyWithMissingInstanceMaxCount := `
						{
							"instance_min_count":1,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":30,
								"operator":"<",
								"adjustment":"-1"
							}]
						}`
					body = createInstanceCreationRequestBody(invalidDefaultPolicyWithMissingInstanceMaxCount)
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					bodyBytes, err := io.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(bodyBytes)).To(ContainSubstring(`instance_max_count is required`))
				})
			})
			Context("When a default policy with too many rules is present", func() {
				BeforeEach(func() {
					invalidDefaultPolicyWithTooManyRules := `
						{	"instance_max_count":4,
							"instance_min_count":1,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":30,
								"operator":"<",
								"adjustment":"-1"
							},
							{
								"metric_type":"memoryused",
								"threshold":30,
								"operator":"<",
								"adjustment":"-1"
							}]
						}`
					body = createInstanceCreationRequestBody(invalidDefaultPolicyWithTooManyRules)
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					bodyBytes, err := io.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(bodyBytes)).To(MatchJSON(`{"description": "error: policy did not adhere to plan: Too many scaling rules: Found 2 scaling rules, but a maximum of 1 scaling rules are allowed for this service plan. "}`))
				})
			})
			Context("When a default policy is present", func() {
				BeforeEach(func() {
					d := json.RawMessage(testDefaultPolicy)
					instanceCreationReqBody = &models.InstanceCreationRequestBody{
						OrgGUID:   "an-org-guid",
						SpaceGUID: "an-space-guid",
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "autoscaler-guid",
							PlanID:    "autoscaler-free-plan-id",
						},
						Parameters: models.InstanceParameters{
							DefaultPolicy: d,
						},
					}
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("succeeds with 201 and saves the default policy", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated))
					Expect(bindingdb.CreateServiceInstanceCallCount()).To(Equal(1))
					ctx, serviceInstance := bindingdb.CreateServiceInstanceArgsForCall(0)
					Expect(ctx).To(Not(BeNil()))
					Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
					Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
					Expect(serviceInstance.DefaultPolicyGuid).To(HaveLen(36))
				})
			})

		})
	})

	Describe("UpdateServiceInstance", func() {
		var err error
		var instanceUpdateRequestBody *models.InstanceUpdateRequestBody
		var body []byte
		servicePlanGuid := "autoscaler-free-plan-id"
		callUpdateServiceInstance := func() {
			req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			Expect(err).NotTo(HaveOccurred())
			req.SetPathValue("instance_id", testInstanceId)
			handler.Update(resp, req)
		}
		updatePlanAndDefaultPolicy := func(fromPlan string, targetPlan string, defaultPolicy json.RawMessage) {
			instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
				BrokerCommonRequestBody: models.BrokerCommonRequestBody{
					ServiceID:      "autoscaler-guid",
					PlanID:         targetPlan,
					PreviousValues: models.PreviousValues{PlanID: fromPlan},
				},
			}

			if defaultPolicy != nil {
				instanceUpdateRequestBody.Parameters = &models.InstanceParameters{DefaultPolicy: defaultPolicy}
			}

			body, err = json.Marshal(instanceUpdateRequestBody)
			Expect(err).NotTo(HaveOccurred())
		}

		updatePlan := func(fromPlan string, targetPlan string) {
			updatePlanAndDefaultPolicy(fromPlan, targetPlan, nil)
		}

		updateDefaultPolicy := func(defaultPolicy json.RawMessage) {
			updatePlanAndDefaultPolicy(servicePlanGuid, "", defaultPolicy)
		}

		JustBeforeEach(callUpdateServiceInstance)

		Context("When request body is not a valid json", func() {
			BeforeEach(func() {
				body = []byte("")
			})
			It("fails with 422", func() {
				Expect(resp.Code).To(Equal(http.StatusUnprocessableEntity))
			})
		})
		Context("When mandatory parameters are not provided", func() {
			Context("When ServiceID is not provided", func() {
				BeforeEach(func() {
					instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{},
					}
					body, err = json.Marshal(instanceUpdateRequestBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})
		})
		Context("When an invalid default policy is present", func() {
			BeforeEach(func() {
				invalidDefaultPolicy := `
							{
								"instance_min_count":1,
								"scaling_rules":[
								{
									"metric_type":"memoryused",
									"threshold":30,
									"operator":"<",
									"adjustment":"-1"
								}]
							}`
				m := json.RawMessage(invalidDefaultPolicy)
				updateDefaultPolicy(m)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{}, nil)
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				bodyBytes, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(ContainSubstring(`instance_max_count is required`))
			})
		})
		Context("When the service instance to be updated does not exist", func() {
			BeforeEach(func() {
				emptyPolicyParameter := json.RawMessage("\n{\t}\n")
				updateDefaultPolicy(emptyPolicyParameter)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{}, db.ErrDoesNotExist)
			})
			It("retrieves the service instance", func() {
				Expect(bindingdb.GetServiceInstanceCallCount()).To(Equal(1))
				ctx, appid := bindingdb.GetServiceInstanceArgsForCall(0)
				Expect(ctx).ToNot(BeNil())
				Expect(appid).To(Equal(testInstanceId))
			})
			It("fails with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone))
			})
		})
		Context("When all mandatory parameters are present", func() {

			BeforeEach(func() {
				emptyPolicyParameter := json.RawMessage("\n{\t}\n")
				updateDefaultPolicy(emptyPolicyParameter)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{}, nil)
			})
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
			It("retrieves the service instance", func() {
				Expect(bindingdb.GetServiceInstanceCallCount()).To(Equal(1))
				ctx, appid := bindingdb.GetServiceInstanceArgsForCall(0)
				Expect(ctx).ToNot(BeNil())
				Expect(appid).To(Equal(testInstanceId))
			})
		})
		Context("When a default policy is present and there was previously not a default policy", func() {

			BeforeEach(func() {
				d := json.RawMessage(testDefaultPolicy)
				updateDefaultPolicy(d)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
				}, nil)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-1", "app-id-2"}, nil)
				policydb.SetOrUpdateDefaultAppPolicyReturns([]string{"app-id-2"}, nil)
				verifyScheduleIsUpdatedInScheduler("app-id-2", testDefaultPolicy)
			})
			Context("successfully", func() {
				It("succeeds with 200, saves the default policy, and sets the default policy on the already bound apps", func() {
					By("returning 200")
					Expect(resp.Code).To(Equal(http.StatusOK))

					By("saving the default policy")
					Expect(bindingdb.UpdateServiceInstanceCallCount()).To(Equal(1))
					ctx, serviceInstance := bindingdb.UpdateServiceInstanceArgsForCall(0)
					Expect(ctx).To(Not(BeNil()))
					Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
					Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
					Expect(serviceInstance.DefaultPolicyGuid).To(HaveLen(36))

					By("setting the default policy on the already bound apps")
					Expect(bindingdb.GetAppIdsByInstanceIdCallCount()).To(Equal(1))
					ctx, lookedUpInstance := bindingdb.GetAppIdsByInstanceIdArgsForCall(0)
					Expect(ctx).To(Not(BeNil()))
					Expect(lookedUpInstance).To(Equal(testInstanceId))
					Expect(policydb.SetOrUpdateDefaultAppPolicyCallCount()).To(Equal(1))
					ctx, appsUpdated, oldPolicyGuid, policy, policySetGuid := policydb.SetOrUpdateDefaultAppPolicyArgsForCall(0)
					Expect(ctx).To(Not(BeNil()))
					Expect(oldPolicyGuid).To(BeEmpty())
					Expect(policySetGuid).To(Equal(serviceInstance.DefaultPolicyGuid))
					Expect(policy).To(MatchJSON(serviceInstance.DefaultPolicy))
					Expect(appsUpdated).To(Equal([]string{"app-id-1", "app-id-2"}))

					By("updating the scheduler")
					Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				})
			})
			Context("with a fake plan checker", func() {
				BeforeEach(func() { fakePlanChecker = &fakes.FakePlanChecker{} })
				It("it uses the correct plan id", func() {
					_, s := fakePlanChecker.CheckPlanArgsForCall(0)
					Expect(s).To(Equal("autoscaler-free-plan-id"))
				})
			})
		})
		Context("When a default policy is present and there was previously a default policy", func() {
			BeforeEach(func() {
				d := json.RawMessage(testDefaultPolicy)
				updateDefaultPolicy(d)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
					DefaultPolicy:     "{}",
					DefaultPolicyGuid: "a-default-policy-guid",
				}, nil)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-1", "app-id-2"}, nil)
				policydb.SetOrUpdateDefaultAppPolicyReturns([]string{"app-id-2"}, nil)
				verifyScheduleIsUpdatedInScheduler("app-id-2", testDefaultPolicy)
			})
			It("succeeds with 200, saves the default policy, and updates the default policy", func() {
				By("returning 200")
				all, err2 := io.ReadAll(resp.Body)
				Expect(err2).ToNot(HaveOccurred())
				Expect(resp.Code).To(Equal(http.StatusOK), "error response: '%s'", all)

				By("saving the default policy")
				Expect(bindingdb.UpdateServiceInstanceCallCount()).To(Equal(1))
				ctx, serviceInstance := bindingdb.UpdateServiceInstanceArgsForCall(0)
				Expect(ctx).To(Not(BeNil()))
				Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
				Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
				Expect(serviceInstance.DefaultPolicyGuid).To(HaveLen(36))

				By("setting the default policy on the already bound apps")
				Expect(policydb.SetOrUpdateDefaultAppPolicyCallCount()).To(Equal(1))
				ctx, appToUpdate, oldPolicyGuid, newPolicy, newPolicyGuid := policydb.SetOrUpdateDefaultAppPolicyArgsForCall(0)
				newPolicyStr, err := json.Marshal(newPolicy)
				Expect(err).ToNot(HaveOccurred())
				Expect(ctx).To(Not(BeNil()))
				Expect(appToUpdate).To(Equal([]string{"app-id-1", "app-id-2"}))
				Expect(oldPolicyGuid).To(Equal("a-default-policy-guid"))
				Expect(newPolicyGuid).To(Equal(serviceInstance.DefaultPolicyGuid))

				Expect(newPolicyStr).To(MatchJSON(serviceInstance.DefaultPolicy))

				By("updating the scheduler")
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
		Context("When the default is set to be removed and there was previously a default policy", func() {
			BeforeEach(func() {
				emptyJsonObject := json.RawMessage("\n{\n}\n")
				updateDefaultPolicy(emptyJsonObject)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
					DefaultPolicy:     "{}",
					DefaultPolicyGuid: "a-default-policy-guid",
				}, nil)
				policydb.DeletePoliciesByPolicyGuidReturns([]string{"app-id-2"}, nil)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-2", "app-id-1"}, nil)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-1", "app-id-2"}, nil)

				var encodedTestDefaultPolicy models.ScalingPolicy
				err = json.Unmarshal([]byte(testDefaultPolicy), &encodedTestDefaultPolicy)
				Expect(err).To(BeNil())
				policydb.GetAppPolicyReturns(&encodedTestDefaultPolicy, nil)
				verifyScheduleIsDeletedInScheduler("app-id-2")
			})
			It("succeeds with 200 and removes the default policy", func() {
				By("returning 200")
				all, anErr := io.ReadAll(resp.Body)
				Expect(anErr).ToNot(HaveOccurred())
				Expect(resp.Code).To(Equal(http.StatusOK), "Failed with resp:'%s'", all)

				By("removing the default policy")
				Expect(bindingdb.UpdateServiceInstanceCallCount()).To(Equal(1))
				ctx, serviceInstance := bindingdb.UpdateServiceInstanceArgsForCall(0)
				Expect(ctx).To(Not(BeNil()))
				Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
				Expect(serviceInstance.DefaultPolicy).To(Equal(""))
				Expect(serviceInstance.DefaultPolicyGuid).To(Equal(""))

				By("removing the default policy on the already bound apps")
				Expect(policydb.DeletePoliciesByPolicyGuidCallCount()).To(Equal(1))
				ctx, removedPolicy := policydb.DeletePoliciesByPolicyGuidArgsForCall(0)
				Expect(ctx).To(Not(BeNil()))
				Expect(removedPolicy).To(Equal("a-default-policy-guid"))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})

		Context("When a default policy with too many rules is present", func() {
			BeforeEach(func() {
				invalidDefaultPolicy := `
						{	"instance_max_count":4,
							"instance_min_count":1,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":30,
								"operator":"<",
								"adjustment":"-1"
							},
							{
								"metric_type":"memoryused",
								"threshold":30,
								"operator":"<",
								"adjustment":"-1"
							}]
						}`
				m := json.RawMessage(invalidDefaultPolicy)
				updateDefaultPolicy(m)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
				}, nil)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-2", "app-id-1"}, nil)
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				bodyBytes, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(MatchJSON(`{"description":"error: policy did not adhere to plan: Too many scaling rules: Found 2 scaling rules, but a maximum of 1 scaling rules are allowed for this service plan. "}`))
			})
		})

		Context("When the service plan is updatable", func() {

			Context("and the target plan is available", func() {
				BeforeEach(func() {
					updatePlan("autoscaler-free-plan-id", "a-plan-id-not-updatable")
					bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
						ServiceInstanceId: testInstanceId,
					}, nil)
				})

				It("succeeds with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK))
				})
			})

			Context("and the target plan is not part of the catalog", func() {
				BeforeEach(func() {
					updatePlan("autoscaler-free-plan-id", "unknown-plan")
					bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
						ServiceInstanceId: testInstanceId,
					}, nil)
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})
		})

		Context("The service plan is updated and a default policy was present previously", func() {
			BeforeEach(func() {
				updatePlan("autoscaler-free-plan-id", "a-plan-id-not-updatable")

				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-1", "app-id-2"}, nil)

				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
					DefaultPolicy:     testDefaultPolicy,
					DefaultPolicyGuid: "default-policy-guid",
				}, nil)
				policydb.GetAppPolicyReturns(&models.ScalingPolicy{}, nil)
			})

			It("Succeeds and leaves the old default policy in place", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))

				By("not touching service instance and leaving the policies unchanged", func() {
					Expect(bindingdb.UpdateServiceInstanceCallCount()).To(Equal(0))
					Expect(policydb.SetOrUpdateDefaultAppPolicyCallCount()).To(Equal(0))
				})
			})
		})

		Context("When the service plan is not updatable", func() {
			BeforeEach(func() {
				updatePlan("a-plan-id-not-updatable", "autoscaler-free-plan-id")
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
				}, nil)
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusUnprocessableEntity))
				Expect(resp.Body.String()).To(MatchJSON(`{"error": "PlanChangeNotSupported", "description":"The requested plan migration cannot be performed"}`))
			})
		})
		Context("When the service instance does not exist", func() {
			BeforeEach(func() {
				updatePlan("autoscaler-free-plan-id", "a-plan-id-not-updatable")
				bindingdb.GetServiceInstanceReturns(nil, db.ErrDoesNotExist)
			})
			It("fails with 404", func() {
				Expect(resp.Code).To(Equal(http.StatusGone))
			})
		})
		Context("Update service plan and policy both are updated together", func() {
			BeforeEach(func() {
				d := json.RawMessage(testDefaultPolicy)
				updatePlanAndDefaultPolicy("autoscaler-free-plan-id", "a-plan-id-not-updatable", d)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
					DefaultPolicy:     "{}",
					DefaultPolicyGuid: "a-default-policy-guid",
				}, nil)
				policydb.SetOrUpdateDefaultAppPolicyReturns([]string{"app-id-2"}, nil)
				verifyScheduleIsUpdatedInScheduler("app-id-2", testDefaultPolicy)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-1", "app-id-2"}, nil)
			})

			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				By("saving the default policy")
				Expect(bindingdb.UpdateServiceInstanceCallCount()).To(Equal(1))
				ctx, serviceInstance := bindingdb.UpdateServiceInstanceArgsForCall(0)
				Expect(ctx).To(Not(BeNil()))
				Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
				Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
				Expect(serviceInstance.DefaultPolicyGuid).To(HaveLen(36))

				By("setting the default policy on the already bound apps")
				Expect(policydb.SetOrUpdateDefaultAppPolicyCallCount()).To(Equal(1))
				ctx, appToUpdate, oldPolicyGuid, newPolicy, newPolicyGuid := policydb.SetOrUpdateDefaultAppPolicyArgsForCall(0)
				Expect(ctx).To(Not(BeNil()))
				Expect(appToUpdate).To(Equal([]string{"app-id-1", "app-id-2"}))
				Expect(oldPolicyGuid).To(Equal("a-default-policy-guid"))
				Expect(newPolicyGuid).To(Equal(serviceInstance.DefaultPolicyGuid))
				Expect(newPolicy).To(MatchJSON(serviceInstance.DefaultPolicy))

				By("updating the scheduler")
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
	})

	Describe("DeleteServiceInstance", func() {
		JustBeforeEach(func() {
			req, _ = http.NewRequest(http.MethodDelete, "", nil)
			req.SetPathValue("instance_id", testInstanceId)
			values := url.Values{}
			values.Set("service_id", "autoscaler-guid")
			values.Set("plan_id", "autoscaler-free-plan-id")
			req.URL.RawQuery = values.Encode()

			handler.Deprovision(resp, req)
		})

		Context("When database DeleteServiceInstance call returns ErrDoesnotExist", func() {
			BeforeEach(func() {
				bindingdb.DeleteServiceInstanceReturns(db.ErrDoesNotExist)
			})
			It("fails with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone))
			})
		})

		Context("When database DeleteServiceInstance call returns error other than ErrDoesnotExist", func() {
			BeforeEach(func() {
				bindingdb.DeleteServiceInstanceReturns(fmt.Errorf("error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(MatchJSON(`{"description":"error deleting service instance"}`))
			})
		})

		Context("When all mandatory parameters are present", func() {
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When service bindings are present", func() {
			BeforeEach(func() {
				var bindingIds []string
				bindingIds = append(bindingIds, testBindingId)

				bindingdb.GetBindingIdsByInstanceIdReturns(bindingIds, nil)
				bindingdb.GetAppIdByBindingIdReturnsOnCall(0, testAppId, nil)
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("if it has been deleted", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				Expect(bindingdb.DeleteServiceBindingCallCount()).To(Equal(1))
				ctx, bindingid := bindingdb.DeleteServiceBindingArgsForCall(0)
				Expect(ctx).ToNot(BeNil())
				Expect(bindingid).To(Equal(testBindingId))
				Expect(bindingdb.DeleteServiceInstanceCallCount()).To(Equal(1))
				Expect(policydb.DeletePolicyCallCount(), 1)
				ctx, appid := policydb.DeletePolicyArgsForCall(0)
				Expect(ctx).To(Not(BeNil()))
				Expect(appid, testAppId)
			})
		})

		Context("When service bindings are present", func() {
			BeforeEach(func() {
				var bindingIds []string
				bindingIds = append(bindingIds, testBindingId)

				bindingdb.GetBindingIdsByInstanceIdReturns(bindingIds, nil)
				bindingdb.GetAppIdByBindingIdReturnsOnCall(0, testAppId, nil)
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("if it has been deleted", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				Expect(bindingdb.DeleteServiceBindingCallCount()).To(Equal(1))
				ctx, bindingid := bindingdb.DeleteServiceBindingArgsForCall(0)
				Expect(ctx).ToNot(BeNil())
				Expect(bindingid).To(Equal(testBindingId))
				Expect(bindingdb.DeleteServiceInstanceCallCount()).To(Equal(1))
				Expect(policydb.DeletePolicyCallCount(), 1)
				ctx, appid := policydb.DeletePolicyArgsForCall(0)
				Expect(ctx).To(Not(BeNil()))
				Expect(appid, testAppId)
			})
		})

	})

	Describe("BindServiceInstance", func() {
		var (
			err                   error
			bindingRequestBody    *models.BindingRequestBody
			schedulerExpectedJSON string
			bindingPolicy         string
			body                  []byte
		)
		BeforeEach(func() {
			bindingPolicy = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"schedules": {
								"timezone": "Asia/Shanghai",
								"recurring_schedule": [{
									  "start_time": "10:00",
									  "end_time": "18:00",
									  "days_of_week": [
										1,
										2,
										3
									  ],
									  "instance_min_count": 1,
									  "instance_max_count": 10,
									  "initial_min_instance_count": 5
									}]
							},
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"threshold":30,
						"operator":"<",
						"adjustment":"-1"
					}]
				}`
			bindingRequestBody = &models.BindingRequestBody{
				AppID: "an-app-id",
				BrokerCommonRequestBody: models.BrokerCommonRequestBody{
					ServiceID: "autoscaler-guid",
					PlanID:    "autoscaler-free-plan-id",
				},
				Policy: json.RawMessage(bindingPolicy),
			}
		})
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			handler.Bind(resp, req)
		})
		Context("When request body is not a valid json", func() {
			BeforeEach(func() {
				body = []byte("")
			})
			It("fails with 422", func() {
				Expect(resp.Code).To(Equal(http.StatusUnprocessableEntity))
			})
		})
		Context("When mandatory parameters are not provided", func() {
			Context("When AppID is not provided", func() {
				BeforeEach(func() {
					bindingRequestBody.AppID = ""
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 422", func() {
					Expect(resp.Code).To(Equal(http.StatusUnprocessableEntity))
					Expect(resp.Body.String()).To(MatchJSON(`{"error": "RequiresApp", "description": "error: service must be bound to an application - service key creation is not supported"}`))
				})
			})

			Context("When ServiceID is not provided", func() {
				BeforeEach(func() {
					bindingRequestBody.ServiceID = ""
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})

			Context("When PlanID is not provided", func() {
				BeforeEach(func() {
					bindingRequestBody.PlanID = ""
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})

		})
		Context("When a policy with too many rules is provided", func() {
			BeforeEach(func() {
				bindingPolicy = `{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"threshold":30,
						"operator":"<",
						"adjustment":"-1"
					},
					{
						"metric_type":"memoryused",
						"threshold":30,
						"operator":"<",
						"adjustment":"-1"
					}]
				}`
				bindingRequestBody = &models.BindingRequestBody{
					AppID: "an-app-id",
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "autoscaler-guid",
						PlanID:    "autoscaler-free-plan-id",
					},
					Policy: json.RawMessage(bindingPolicy),
				}

				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(MatchJSON(`{"description":"error: policy did not adhere to plan: Too many scaling rules: Found 2 scaling rules, but a maximum of 1 scaling rules are allowed for this service plan. "}`))
			})
		})
		Context("When mandatory parameters are present", func() {
			BeforeEach(func() {
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
				verifyScheduleIsUpdatedInScheduler(testAppId, bindingPolicy)
			})
			It("succeeds with 201", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated))

				By("updating the scheduler")
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
			It("returns the correct binding parameters", func() {
				verifyCredentialsGenerated(resp)
			})
		})
		Context("Binding configurations", func() {
			When("invalid custom strategy provided in the binding parameters", func() {
				BeforeEach(func() {
					bindingPolicy = `{
				  "configuration": {
					"custom_metrics": {
					  "metric_submission_strategy": {
						"allow_from": "same_app"
					  }
					}
				  },
				  "instance_max_count":4,
				  "instance_min_count":1,
				  "schedules": {
					"timezone": "Asia/Shanghai",
					"recurring_schedule": [{
					  "start_time": "10:00",
					  "end_time": "18:00",
					  "days_of_week": [
						1,
						2,
						3
					  ],
					  "instance_min_count": 1,
					  "instance_max_count": 10,
					  "initial_min_instance_count": 5
					}]
				  },
				  "scaling_rules":[
					{
					  "metric_type":"memoryused",
					  "threshold":30,
					  "operator":"<",
					  "adjustment":"-1"
					}]
				}`
					bindingRequestBody.Policy = json.RawMessage(bindingPolicy)
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
					bindingPolicy = `{
				  "instance_max_count":4,
				  "instance_min_count":1,
				  "schedules": {
					"timezone": "Asia/Shanghai",
					"recurring_schedule": [{
					  "start_time": "10:00",
					  "end_time": "18:00",
					  "days_of_week": [
						1,
						2,
						3
					  ],
					  "instance_min_count": 1,
					  "instance_max_count": 10,
					  "initial_min_instance_count": 5
					}]
				  },
				  "scaling_rules":[
					{
					  "metric_type":"memoryused",
					  "threshold":30,
					  "operator":"<",
					  "adjustment":"-1"
					}]
				}`
					verifyScheduleIsUpdatedInScheduler(testAppId, bindingPolicy)
				})
				It("should fail with 400", func() {
					Expect(resp.Body.String()).To(ContainSubstring("{\"description\":\"invalid policy provided: [{\\\"context\\\":\\\"(root).configuration.custom_metrics.metric_submission_strategy.allow_from\\\",\\\"description\\\":\\\"configuration.custom_metrics.metric_submission_strategy.allow_from must be one of the following: \\\\\\\"bound_app\\\\\\\"\\\"}]\"}"))
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
				})
			})
			When("are empty", func() {
				BeforeEach(func() {
					bindingPolicy = `{
				  "instance_max_count":4,
				  "instance_min_count":1,
				  "schedules": {
					"timezone": "Asia/Shanghai",
					"recurring_schedule": [{
					  "start_time": "10:00",
					  "end_time": "18:00",
					  "days_of_week": [
						1,
						2,
						3
					  ],
					  "instance_min_count": 1,
					  "instance_max_count": 10,
					  "initial_min_instance_count": 5
					}]
				  },
				  "scaling_rules":[
					{
					  "metric_type":"memoryused",
					  "threshold":30,
					  "operator":"<",
					  "adjustment":"-1"
					}]
				}`
					bindingRequestBody.Policy = json.RawMessage(bindingPolicy)
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())

					verifyScheduleIsUpdatedInScheduler(testAppId, bindingPolicy)
				})
				It("set the default custom metrics strategy", func() {
					_, _, _, _, customMetricsStrategy := bindingdb.CreateServiceBindingArgsForCall(0)
					Expect(customMetricsStrategy).To(Equal(models.CustomMetricsSameApp))

				})
				It("succeeds with 201", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated))
					By("updating the scheduler")
					Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
					Expect(bindingdb.CreateServiceBindingCallCount()).To(Equal(1))
					verifyCredentialsGenerated(resp)
				})
			})
			When("policy and binding configuration are present", func() {
				BeforeEach(func() {
					bindingPolicy = `{
				  "configuration": {
					"custom_metrics": {
					  "metric_submission_strategy": {
						"allow_from": "bound_app"
					  }
					}
				  },
				  "instance_max_count":4,
				  "instance_min_count":1,
				  "schedules": {
					"timezone": "Asia/Shanghai",
					"recurring_schedule": [{
					  "start_time": "10:00",
					  "end_time": "18:00",
					  "days_of_week": [
						1,
						2,
						3
					  ],
					  "instance_min_count": 1,
					  "instance_max_count": 10,
					  "initial_min_instance_count": 5
					}]
				  },
				  "scaling_rules":[
					{
					  "metric_type":"memoryused",
					  "threshold":30,
					  "operator":"<",
					  "adjustment":"-1"
					}]
				}`
					bindingRequestBody.Policy = json.RawMessage(bindingPolicy)
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
					bindingPolicy = `{
				  "instance_max_count":4,
				  "instance_min_count":1,
				  "schedules": {
					"timezone": "Asia/Shanghai",
					"recurring_schedule": [{
					  "start_time": "10:00",
					  "end_time": "18:00",
					  "days_of_week": [
						1,
						2,
						3
					  ],
					  "instance_min_count": 1,
					  "instance_max_count": 10,
					  "initial_min_instance_count": 5
					}]
				  },
				  "scaling_rules":[
					{
					  "metric_type":"memoryused",
					  "threshold":30,
					  "operator":"<",
					  "adjustment":"-1"
					}]
				}`
					verifyScheduleIsUpdatedInScheduler(testAppId, bindingPolicy)
				})
				It("succeeds with 201", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated))
					By("updating the scheduler")
					Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
					Expect(bindingdb.CreateServiceBindingCallCount()).To(Equal(1))
					verifyCredentialsGenerated(resp)
				})
			})
		})

		Context("credential-type is provided while binding", func() {
			BeforeEach(func() {
				schedulerExpectedJSON = `{
							"instance_max_count":3,
							"instance_min_count":1,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":99,
								"operator":"<",
								"adjustment":"-1"
							}],
							"schedules": {
								"timezone": "Asia/Shanghai",
								"recurring_schedule": [{
									  "start_time": "10:00",
									  "end_time": "18:00",
									  "days_of_week": [
										1,
										2,
										3
									  ],
									  "instance_min_count": 1,
									  "instance_max_count": 10,
									  "initial_min_instance_count": 5
									}]
							}
						}`
			})
			Context("credential-type is set with invalid value", func() {
				const testBindingPolicy = `{
							"credential-type": "invalid-binding-secret",
							"instance_max_count":3,
							"instance_min_count":1,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":99,
								"operator":"<",
								"adjustment":"-1"
							}],
							"schedules": {
								"timezone": "Asia/Shanghai",
								"recurring_schedule": [{
									  "start_time": "10:00",
									  "end_time": "18:00",
									  "days_of_week": [
										1,
										2,
										3
									  ],
									  "instance_min_count": 1,
									  "instance_max_count": 10,
									  "initial_min_instance_count": 5
									}]
							}
						}`
				BeforeEach(func() {
					bindingRequestBody = &models.BindingRequestBody{
						AppID: testAppId,
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "autoscaler-guid",
							PlanID:    "autoscaler-free-plan-id",
						},
						Policy: json.RawMessage(testBindingPolicy),
					}
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())

					verifyScheduleIsUpdatedInScheduler(testAppId, testBindingPolicy)
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					Expect(resp.Body.String()).To(MatchJSON(`{"error": "validate-credential-type","description": "invalid credential type provided: allowed values are [binding-secret, x509]"}`))
				})
			})
			Context("credential-type is set to binding-secret", func() {
				const testBindingPolicy = `{
							"credential-type": "binding-secret",
							"instance_max_count":3,
							"instance_min_count":1,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":99,
								"operator":"<",
								"adjustment":"-1"
							}],
							"schedules": {
								"timezone": "Asia/Shanghai",
								"recurring_schedule": [{
									  "start_time": "10:00",
									  "end_time": "18:00",
									  "days_of_week": [
										1,
										2,
										3
									  ],
									  "instance_min_count": 1,
									  "instance_max_count": 10,
									  "initial_min_instance_count": 5
									}]
							}
						}`
				BeforeEach(func() {
					bindingRequestBody = &models.BindingRequestBody{
						AppID: testAppId,
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "autoscaler-guid",
							PlanID:    "autoscaler-free-plan-id",
						},
						Policy: json.RawMessage(testBindingPolicy),
					}

					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
					verifyScheduleIsUpdatedInScheduler(testAppId, schedulerExpectedJSON)

					fakeCredentials.CreateReturns(&models.Credential{
						Username: "test-username",
						Password: "test-password",
					}, nil)
				})
				It("should return 201 response code", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated))

					By("updating the scheduler")
					Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				})
				It("should create the correct credentials and return them in response", func() {
					creds := &models.CredentialResponse{}
					responseString := resp.Body.String()
					err := json.Unmarshal([]byte(responseString), creds)
					Expect(err).NotTo(HaveOccurred())
					Expect(creds.Credentials.CustomMetrics.Username).To(Equal("test-username"))
					Expect(creds.Credentials.CustomMetrics.Password).To(Equal("test-password"))
					Expect(*creds.Credentials.CustomMetrics.URL).To(Equal("someURL"))
					Expect(creds.Credentials.CustomMetrics.MtlsUrl).To(Equal("Mtls-someURL"))
				})
			})
			Context("credential-type is set as x509", func() {
				const testBindingPolicy = `{
							"instance_max_count":3,
							"instance_min_count":1,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":99,
								"operator":"<",
								"adjustment":"-1"
							}],
							"schedules": {
								"timezone": "Asia/Shanghai",
								"recurring_schedule": [{
									  "start_time": "10:00",
									  "end_time": "18:00",
									  "days_of_week": [
										1,
										2,
										3
									  ],
									  "instance_min_count": 1,
									  "instance_max_count": 10,
									  "initial_min_instance_count": 5
									}]
							},
							"credential-type": "x509"
						}`
				BeforeEach(func() {
					bindingRequestBody = &models.BindingRequestBody{
						AppID: testAppId,
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "autoscaler-guid",
							PlanID:    "autoscaler-free-plan-id",
						},
						Policy: json.RawMessage(testBindingPolicy),
					}
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
					verifyScheduleIsUpdatedInScheduler(testAppId, schedulerExpectedJSON)
				})
				It("should return 201 response code", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated))
					Expect(policydb.SaveAppPolicyCallCount()).To(Equal(1))
					ctx, appID, policy, _ := policydb.SaveAppPolicyArgsForCall(0)
					Expect(ctx).To(Not(BeNil()))
					Expect(appID).To(Equal(testAppId))
					Expect(policy).NotTo(MatchJSON(testBindingPolicy))
				})
				It("should not contains username/password/url but contains mtls_url in the bind response", func() {
					creds := &models.CredentialResponse{}
					responseString := resp.Body.String()
					err := json.Unmarshal([]byte(responseString), creds)
					Expect(err).NotTo(HaveOccurred())
					Expect(creds.Credentials.CustomMetrics.Credential).To(BeNil())
					Expect(creds.Credentials.CustomMetrics.URL).To(BeNil())
					Expect(creds.Credentials.CustomMetrics.MtlsUrl).To(Equal("Mtls-someURL"))
				})
			})
		})
		Context("credential-type is not provided as part of binding request parameters", func() {
			const testBindingPolicy = `{
							"instance_max_count":3,
							"instance_min_count":1,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":99,
								"operator":"<",
								"adjustment":"-1"
							}],
							"schedules": {
								"timezone": "Asia/Shanghai",
								"recurring_schedule": [{
									  "start_time": "10:00",
									  "end_time": "18:00",
									  "days_of_week": [
										1,
										2,
										3
									  ],
									  "instance_min_count": 1,
									  "instance_max_count": 10,
									  "initial_min_instance_count": 5
									}]
							}
						}`
			BeforeEach(func() {
				bindingRequestBody = &models.BindingRequestBody{
					AppID: testAppId,
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "autoscaler-guid",
						PlanID:    "autoscaler-free-plan-id",
					},
					Policy: json.RawMessage(testBindingPolicy),
				}
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				fakeCredentials.CreateReturns(&models.Credential{
					Username: "test-username",
					Password: "test-password",
				}, nil)

				verifyScheduleIsUpdatedInScheduler(testAppId, testBindingPolicy)
			})
			It("should consider binding-secret as default credential-type and create credentials", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated))
				creds := &models.CredentialResponse{}
				responseString := resp.Body.String()
				err := json.Unmarshal([]byte(responseString), creds)
				Expect(err).NotTo(HaveOccurred())
				Expect(creds.Credentials.CustomMetrics.Username).To(Equal("test-username"))
				Expect(creds.Credentials.CustomMetrics.Password).To(Equal("test-password"))
				Expect(*creds.Credentials.CustomMetrics.URL).To(Equal("someURL"))
				Expect(creds.Credentials.CustomMetrics.MtlsUrl).To(Equal("Mtls-someURL"))
			})
		})

		Context("When a default policy was provided when creating the service instance", func() {
			BeforeEach(func() {
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{testInstanceId, testOrgId, testSpaceId, testDefaultPolicy, testDefaultGuid}, nil)
			})

			Context("and a policy is provided while binding to it", func() {
				const testBindingPolicy = `{
							"instance_max_count":3,
							"instance_min_count":1,
							"scaling_rules":[
							{
								"metric_type":"memoryused",
								"threshold":99,
								"operator":"<",
								"adjustment":"-1"
							}],
							"schedules": {
								"timezone": "Asia/Shanghai",
								"recurring_schedule": [{
									  "start_time": "10:00",
									  "end_time": "18:00",
									  "days_of_week": [
										1,
										2,
										3
									  ],
									  "instance_min_count": 1,
									  "instance_max_count": 10,
									  "initial_min_instance_count": 5
									}]
							}
						}`
				BeforeEach(func() {
					bindingRequestBody = &models.BindingRequestBody{
						AppID: testAppId,
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "autoscaler-guid",
							PlanID:    "autoscaler-free-plan-id",
						},
						Policy: json.RawMessage(testBindingPolicy),
					}
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())

					verifyScheduleIsUpdatedInScheduler(testAppId, testBindingPolicy)
				})
				It("succeeds with 201 and saves the binding's policy", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated))
					Expect(policydb.SaveAppPolicyCallCount()).To(Equal(1))
					ctx, appID, policy, _ := policydb.SaveAppPolicyArgsForCall(0)
					Expect(ctx).To(Not(BeNil()))
					Expect(appID).To(Equal(testAppId))
					Expect(policy).NotTo(MatchJSON(testDefaultPolicy))
					Expect(policy).To(MatchJSON(testBindingPolicy))

					By("updating the scheduler")
					Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				})
			})

			Context("and no policy is provided while binding to it", func() {
				BeforeEach(func() {
					bindingRequestBody = &models.BindingRequestBody{
						AppID: testAppId,
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "autoscaler-guid",
							PlanID:    "autoscaler-free-plan-id",
						},
					}
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
					verifyScheduleIsUpdatedInScheduler(testAppId, testDefaultPolicy)
				})
				It("succeeds with 201 and saves the default policy", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated))
					Expect(policydb.SaveAppPolicyCallCount()).To(Equal(1))
					ctx, appID, policy, _ := policydb.SaveAppPolicyArgsForCall(0)
					Expect(ctx).To(Not(BeNil()))
					Expect(appID).To(Equal(testAppId))
					Expect(policy).To(MatchJSON(testDefaultPolicy))

					By("updating the scheduler")
					Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				})
			})
		})

		Context("When database CreateServiceBinding call returns ErrAlreadyExists", func() {
			BeforeEach(func() {
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.CreateServiceBindingReturns(db.ErrAlreadyExists)
			})
			It("fails with 409", func() {
				Expect(resp.Code).To(Equal(http.StatusConflict))
				Expect(resp.Body.String()).To(MatchJSON(`{"description":"error: an autoscaler service instance is already bound to the application and multiple bindings are not supported"}`))
			})
		})

		Context("When database CreateServiceBinding call returns error other than ErrAlreadyExists", func() {
			BeforeEach(func() {
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.CreateServiceBindingReturns(fmt.Errorf("some sql error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(MatchJSON(`{"description":"error creating service binding"}`))
			})
		})
		Context("When failed to create credential", func() {
			BeforeEach(func() {
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
				fakeCredentials.CreateReturns(nil, fmt.Errorf("some internal error"))
			})
			It("fails with 500", func() {
				Expect(policydb.GetCredentialCallCount()).To(Equal(0))
				Expect(policydb.SaveCredentialCallCount()).To(Equal(0))
				Expect(fakeCredentials.CreateCallCount()).To(Equal(1))
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(MatchJSON(`{"description":"error creating service binding"}`))
			})
		})

		Context("When called with invalid policy json", func() {
			BeforeEach(func() {
				bindingRequestBody.Policy = json.RawMessage(`{
					"instance_max_count":4,
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"threshold":30,
						"operator":"<",
						"adjustment":"-1"
					}]
				}`)
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				bodyBytes, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(ContainSubstring(`instance_min_count is required`))
			})
		})

		Context("When service bindings are present", func() {
			bindingIds := []string{testBindingId}
			BeforeEach(func() {
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				bindingdb.GetBindingIdsByInstanceIdReturns(bindingIds, nil)
				bindingdb.GetAppIdByBindingIdReturns(testAppId, nil)
				verifyScheduleIsDeletedUpdatedInScheduler(testAppId, bindingPolicy)
			})
			It("delete existing bindings before creating the new one", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated))

				requestsRecievedByTheScheduler := schedulerServer.ReceivedRequests()
				Expect(requestsRecievedByTheScheduler).To(HaveLen(2))
				Expect(requestsRecievedByTheScheduler[0].Method).To(Equal(http.MethodDelete))
				Expect(requestsRecievedByTheScheduler[0].URL).To(MatchRegexp(testAppId))
				Expect(requestsRecievedByTheScheduler[1].Method).To(Equal(http.MethodPut))
				Expect(requestsRecievedByTheScheduler[1].URL).To(MatchRegexp(testAppId))
			})
		})
	})

	Describe("UnBindServiceInstance", func() {
		BeforeEach(func() {
			req, _ = http.NewRequest(http.MethodDelete, "", nil)
			req.SetPathValue("instance_id", testInstanceId)
			req.SetPathValue("binding_id", testBindingId)
			values := url.Values{}
			values.Set("service_id", "autoscaler-guid")
			values.Set("plan_id", "autoscaler-free-plan-id")
			req.URL.RawQuery = values.Encode()

			bindingdb.GetAppIdByBindingIdReturns(testAppId, nil)
		})
		JustBeforeEach(func() {
			handler.Unbind(resp, req)
		})
		Context("When mandatory parameters are present", func() {
			BeforeEach(func() {
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
		Context("When there is no app with the bindingId", func() {
			BeforeEach(func() {
				bindingdb.GetAppIdByBindingIdReturns("", sql.ErrNoRows)
			})
			AfterEach(func() {
				bindingdb.GetAppIdByBindingIdReturns(testAppId, nil)
			})
			It("succeed with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone))
			})
		})
		Context("When failed to get appId by bindingId", func() {
			BeforeEach(func() {
				bindingdb.GetAppIdByBindingIdReturns("", errors.New("some error"))
			})
			AfterEach(func() {
				bindingdb.GetAppIdByBindingIdReturns(testAppId, nil)
			})
			It("succeed with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(MatchJSON(`{"description":"unbind failed: error deleting service binding"}`))
			})
		})
		Context("When database DeleteServiceBinding call returns ErrDoesnotExist", func() {
			BeforeEach(func() {
				bindingdb.DeleteServiceBindingReturns(db.ErrDoesNotExist)
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("fails with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})

		Context("When database DeleteServiceBinding call returns error other than ErrDoesnotExist", func() {
			BeforeEach(func() {
				bindingdb.DeleteServiceBindingReturns(fmt.Errorf("some sql error"))
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(MatchJSON(`{"description":"unbind failed: error deleting service binding"}`))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
	})

	Describe("GetBinding", func() {
		var (
			err           error
			bindingPolicy string
		)

		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodGet, "", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.GetBinding(resp, req)
		})

		Context("Binding configurations are exist", func() {
			BeforeEach(func() {
				bindingPolicy = `{
				  "configuration": {
					"custom_metrics": {
					  "metric_submission_strategy": {
						"allow_from": "bound_app"
					  }
					}
				  },
				  "instance_max_count":4,
				  "instance_min_count":1,
				   "scaling_rules":[
					{
					  "metric_type":"memoryused",
					  "threshold":30,
					  "operator":"<",
					  "adjustment":"-1"
					}]
				}`
				Expect(bindingPolicy).NotTo(BeEmpty())

			})
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusPreconditionFailed))
			})
		})
	})
})

func verifyCredentialsGenerated(resp *httptest.ResponseRecorder) {
	creds := &models.CredentialResponse{}
	responseString := resp.Body.String()
	err := json.Unmarshal([]byte(responseString), creds)
	Expect(err).NotTo(HaveOccurred())
	Expect(*creds.Credentials.CustomMetrics.URL).To(Equal("someURL"))
	Expect(creds.Credentials.CustomMetrics.MtlsUrl).To(Equal("Mtls-someURL"))
}

func createInstanceCreationRequestBody(defaultPolicy string) []byte {
	m := json.RawMessage(defaultPolicy)
	instanceCreationReqBody := &models.InstanceCreationRequestBody{
		OrgGUID:   "an-org-guid",
		SpaceGUID: "an-space-guid",
		BrokerCommonRequestBody: models.BrokerCommonRequestBody{
			ServiceID: "autoscaler-guid",
			PlanID:    "autoscaler-free-plan-id",
		},
		Parameters: models.InstanceParameters{
			DefaultPolicy: m,
		},
	}
	body, err := json.Marshal(instanceCreationReqBody)
	Expect(err).NotTo(HaveOccurred())
	return body
}

func verifyScheduleIsUpdatedInScheduler(appId string, policy string) {
	updateSchedulePath, err := routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", appId)
	Expect(err).NotTo(HaveOccurred())
	schedulerServer.Reset()
	schedulerServer.AppendHandlers(ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", updateSchedulePath.String()),
		ghttp.VerifyJSON(policy),
	))
}

func verifyScheduleIsDeletedInScheduler(appId string) {
	deleteSchedulePath, err := routes.SchedulerRoutes().Get(routes.DeleteScheduleRouteName).URLPath("appId", appId)
	Expect(err).NotTo(HaveOccurred())
	schedulerServer.Reset()
	schedulerServer.AppendHandlers(ghttp.CombineHandlers(
		ghttp.VerifyRequest("DELETE", deleteSchedulePath.String()),
	))
}

func verifyScheduleIsDeletedUpdatedInScheduler(appId string, policy string) {
	deleteSchedulePath, err := routes.SchedulerRoutes().Get(routes.DeleteScheduleRouteName).URLPath("appId", appId)
	Expect(err).NotTo(HaveOccurred())
	updateSchedulePath, err := routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", appId)
	Expect(err).NotTo(HaveOccurred())
	schedulerServer.Reset()
	schedulerServer.AppendHandlers(
		ghttp.VerifyRequest("DELETE", deleteSchedulePath.String()),
		ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", updateSchedulePath.String()),
			ghttp.VerifyJSON(policy),
		))
}
