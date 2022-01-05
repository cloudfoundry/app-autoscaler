package brokerserver_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/brokerserver"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"github.com/onsi/gomega/ghttp"

	"golang.org/x/oauth2"

	"github.com/pivotal-cf/brokerapi/domain"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BrokerHandler", func() {
	var (
		fakecfClient    *fakes.FakeCFClient
		bindingdb       *fakes.FakeBindingDB
		policydb        *fakes.FakePolicyDB
		fakeCredentials *fakes.FakeCredentials

		handler *BrokerHandler
		resp    *httptest.ResponseRecorder
		req     *http.Request
	)
	BeforeEach(func() {
		bindingdb = &fakes.FakeBindingDB{}
		policydb = &fakes.FakePolicyDB{}
		resp = httptest.NewRecorder()
		installQuotaAPIHandlers()
		fakecfClient = &fakes.FakeCFClient{}
		fakeCredentials = &fakes.FakeCredentials{}
	})

	JustBeforeEach(func() {
		handler = NewBrokerHandler(lagertest.NewTestLogger("test"), conf, bindingdb, policydb, []domain.Service{{
			ID:   "a-service-id",
			Name: "autoscaler",
			Plans: []domain.ServicePlan{{
				ID:   "a-plan-id",
				Name: "standard",
			}},
		}}, fakecfClient, fakeCredentials)
	})

	Describe("GetBrokerCatalog", func() {
		JustBeforeEach(func() {
			handler.GetBrokerCatalog(resp, req, map[string]string{})
		})
		Context("When getBrokerCatalog is called", func() {
			It("gets the catalog json", func() {
				Expect(resp.Code).To(Equal(http.StatusOK), DebugTestInfo())
				Expect(resp.Body.Bytes()).To(Equal(catalogBytes))
			})
		})
	})

	Describe("CreateServiceInstance", func() {
		var err error
		var instanceCreationReqBody *models.InstanceCreationRequestBody
		var body []byte
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			handler.CreateServiceInstance(resp, req, map[string]string{"instanceId": testInstanceId})
		})
		BeforeEach(func() {
			instanceCreationReqBody = &models.InstanceCreationRequestBody{
				OrgGUID:   "an-org-guid",
				SpaceGUID: "an-space-guid",
				BrokerCommonRequestBody: models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
					PlanID:    "a-plan-id",
				},
			}
		})
		Context("When request body is not a valid json", func() {
			BeforeEach(func() {
				body = []byte("")
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Invalid request body format"}`))
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
					Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
			})
			Context("When SpaceGUID is not provided", func() {
				BeforeEach(func() {
					instanceCreationReqBody.SpaceGUID = ""
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
			})
			Context("When ServiceID is not provided", func() {
				BeforeEach(func() {
					instanceCreationReqBody.ServiceID = ""
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
			})
			Context("When PlanID is not provided", func() {
				BeforeEach(func() {
					instanceCreationReqBody.PlanID = ""
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
			})

		})

		Context("When all parameters are present", func() {
			BeforeEach(func() {
				installQuotaAPIHandlers()
			})
			AfterEach(func() {
				Expect(tokenServer.ReceivedRequests()).To(HaveLen(1))
				Expect(quotaServer.ReceivedRequests()).To(HaveLen(1))
			})
			Context("When database CreateServiceInstance call returns ErrAlreadyExists", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
					bindingdb.CreateServiceInstanceReturns(db.ErrAlreadyExists)
					conf.DashboardRedirectURI = ""
				})
				It("succeeds with 200", func() {
					Expect(resp.Code).To(Equal(http.StatusOK), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{}`))
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
					Expect(resp.Code).To(Equal(http.StatusOK), DebugTestInfo())
					Expect(resp.Body.Bytes()).To(Equal([]byte("{\"dashboard_url\":\"https://service-dashboard-url.com/manage/an-instance-id\"}")))
				})
			})

			Context("When database CreateServiceInstance call returns ErrConflict", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
					bindingdb.CreateServiceInstanceReturns(db.ErrConflict)
				})
				It("fails with 409", func() {
					Expect(resp.Code).To(Equal(http.StatusConflict), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Conflict","message":"Service instance with instance_id \"an-instance-id\" already exists with different parameters"}`))
				})
			})

			Context("When database CreateServiceInstance call returns error other than ErrAlreadyExists", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
					bindingdb.CreateServiceInstanceReturns(fmt.Errorf("some sql error"))
				})
				It("fails with 500", func() {
					Expect(resp.Code).To(Equal(http.StatusInternalServerError), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error creating service instance"}`))
				})
			})

			Context("When dashboard redirect uri is present in config", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
					conf.DashboardRedirectURI = "https://service-dashboard-url.com"
				})
				It("succeeds with 201 and returns dashboard_url", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated), DebugTestInfo())
					Expect(resp.Body.Bytes()).To(Equal([]byte("{\"dashboard_url\":\"https://service-dashboard-url.com/manage/an-instance-id\"}")))
				})
			})
			Context("When all mandatory parameters are present", func() {
				BeforeEach(func() {
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("succeeds with 201", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated), DebugTestInfo())
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
					instanceCreationReqBody = &models.InstanceCreationRequestBody{
						OrgGUID:   "an-org-guid",
						SpaceGUID: "an-space-guid",
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "a-service-id",
							PlanID:    "a-plan-id",
						},
						Parameters: models.InstanceParameters{
							DefaultPolicy: &m,
						},
					}
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
					bodyBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(bodyBytes)).To(Equal(`[{"context":"(root)","description":"instance_max_count is required"}]`))
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
					instanceCreationReqBody = &models.InstanceCreationRequestBody{
						OrgGUID:   "an-org-guid",
						SpaceGUID: "an-space-guid",
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "a-service-id",
							PlanID:    "a-plan-id",
						},
						Parameters: models.InstanceParameters{
							DefaultPolicy: &m,
						},
					}
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					bodyBytes, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(bodyBytes)).To(Equal(`{"code":"Bad Request","message":"Too many scaling rules: Found 2 scaling rules, but a maximum of 1 scaling rules are allowed for this service plan. "}`))
				})
			})
			Context("When a default policy is present", func() {
				BeforeEach(func() {
					d := json.RawMessage(testDefaultPolicy)
					instanceCreationReqBody = &models.InstanceCreationRequestBody{
						OrgGUID:   "an-org-guid",
						SpaceGUID: "an-space-guid",
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "a-service-id",
							PlanID:    "a-plan-id",
						},
						Parameters: models.InstanceParameters{
							DefaultPolicy: &d,
						},
					}
					body, err = json.Marshal(instanceCreationReqBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("succeeds with 201 and saves the default policy", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated), DebugTestInfo())
					Expect(bindingdb.CreateServiceInstanceCallCount()).To(Equal(1))
					serviceInstance := bindingdb.CreateServiceInstanceArgsForCall(0)
					Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
					Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
					Expect(serviceInstance.DefaultPolicyGuid).To(HaveLen(36))
				})
			})

		})
	})

	Describe("Quota checks", func() {
		var err error
		var instanceCreationReqBody *models.InstanceCreationRequestBody
		var body []byte
		//var quota int
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			handler.CreateServiceInstance(resp, req, map[string]string{"instanceId": testInstanceId})
		})
		BeforeEach(func() {
			instanceCreationReqBody = &models.InstanceCreationRequestBody{
				OrgGUID:   "an-org-guid",
				SpaceGUID: "an-space-guid",
				BrokerCommonRequestBody: models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
					PlanID:    "a-plan-id",
				},
			}
			installQuotaAPIHandlers()
		})

		Context("When all mandatory parameters are present", func() {
			BeforeEach(func() {
				d := json.RawMessage(testDefaultPolicy)
				instanceCreationReqBody = &models.InstanceCreationRequestBody{
					OrgGUID:   "an-org-guid",
					SpaceGUID: "an-space-guid",
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					Parameters: models.InstanceParameters{
						DefaultPolicy: &d,
					},
				}
				body, err = json.Marshal(instanceCreationReqBody)
				Expect(err).NotTo(HaveOccurred())
				//quota = 0
			})
			It("succeeds with 201", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated), DebugTestInfo())
				Expect(bindingdb.CreateServiceInstanceCallCount()).To(Equal(1))
				serviceInstance := bindingdb.CreateServiceInstanceArgsForCall(0)
				Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
				Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
				Expect(serviceInstance.DefaultPolicyGuid).To(HaveLen(36))
			})
		})
	})

	Describe("UpdateServiceInstance", func() {
		var err error
		var instanceUpdateRequestBody *models.InstanceUpdateRequestBody
		var body []byte
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			handler.UpdateServiceInstance(resp, req, map[string]string{"instanceId": testInstanceId})
		})
		BeforeEach(func() {
			instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
				BrokerCommonRequestBody: models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
				},
			}
		})
		Context("When request body is not a valid json", func() {
			BeforeEach(func() {
				body = []byte("")
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Invalid request body format"}`))
			})
		})
		Context("When mandatory parameters are not provided", func() {
			Context("When ServiceID is not provided", func() {
				BeforeEach(func() {
					instanceUpdateRequestBody.ServiceID = ""
					body, err = json.Marshal(instanceUpdateRequestBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
			})
		})
		Context("When no default policy & service plan update is performed", func() {
			BeforeEach(func() {
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
			})
			It("fails with 422", func() {
				Expect(resp.Code).To(Equal(http.StatusUnprocessableEntity), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Unprocessable Entity","message":"Failed to update service instance: Only default policy and service plan updates are allowed"}`))
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
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
					},
					Parameters: &models.InstanceParameters{
						DefaultPolicy: &m,
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{}, nil)
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(Equal(`[{"context":"(root)","description":"instance_max_count is required"}]`))
			})
		})
		Context("When the service instance to be updated does not exist", func() {
			BeforeEach(func() {
				emptyPolicyParameter := json.RawMessage("\n{\t}\n")
				parameters := models.InstanceParameters{DefaultPolicy: &emptyPolicyParameter}
				instanceUpdateRequestBody.Parameters = &parameters
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetServiceInstanceReturns(nil, db.ErrDoesNotExist)
			})
			It("retrieves the service instance", func() {
				Expect(bindingdb.GetServiceInstanceCallCount()).To(Equal(1))
				Expect(bindingdb.GetServiceInstanceArgsForCall(0)).To(Equal(testInstanceId))
			})
			It("fails with 404", func() {
				Expect(resp.Code).To(Equal(http.StatusNotFound), DebugTestInfo())
			})
		})
		Context("When all mandatory parameters are present", func() {
			BeforeEach(func() {
				emptyPolicyParameter := json.RawMessage("\n{\t}\n")
				parameters := models.InstanceParameters{DefaultPolicy: &emptyPolicyParameter}
				instanceUpdateRequestBody.Parameters = &parameters
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				fakecfClient.GetServicePlanReturns("a-plan-id", nil)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{}, nil)
			})
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK), DebugTestInfo())
			})
			It("retrieves the service instance", func() {
				Expect(bindingdb.GetServiceInstanceCallCount()).To(Equal(1))
				Expect(bindingdb.GetServiceInstanceArgsForCall(0)).To(Equal(testInstanceId))
			})
		})
		Context("When a default policy is present and there was previously not a default policy", func() {
			BeforeEach(func() {
				d := json.RawMessage(testDefaultPolicy)
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					Parameters: &models.InstanceParameters{
						DefaultPolicy: &d,
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
				}, nil)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-1", "app-id-2"}, nil)
				policydb.SetOrUpdateDefaultAppPolicyReturns([]string{"app-id-2"}, nil)
				verifyScheduleIsUpdatedInScheduler("app-id-2", testDefaultPolicy)
				fakecfClient.GetServicePlanReturns("a-plan-id", nil)
			})
			It("succeeds with 200, saves the default policy, and sets the default policy on the already bound apps", func() {
				By("returning 200")
				Expect(resp.Code).To(Equal(http.StatusOK), DebugTestInfo())

				By("saving the default policy")
				Expect(bindingdb.UpdateServiceInstanceCallCount()).To(Equal(1))
				serviceInstance := bindingdb.UpdateServiceInstanceArgsForCall(0)
				Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
				Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
				Expect(serviceInstance.DefaultPolicyGuid).To(HaveLen(36))

				By("setting the default policy on the already bound apps")
				Expect(bindingdb.GetAppIdsByInstanceIdCallCount()).To(Equal(1))
				lookedUpInstance := bindingdb.GetAppIdsByInstanceIdArgsForCall(0)
				Expect(lookedUpInstance).To(Equal(testInstanceId))
				Expect(policydb.SetOrUpdateDefaultAppPolicyCallCount()).To(Equal(1))
				appsUpdated, oldPolicyGuid, policySet, policySetGuid := policydb.SetOrUpdateDefaultAppPolicyArgsForCall(0)
				Expect(oldPolicyGuid).To(BeEmpty())
				Expect(policySetGuid).To(Equal(serviceInstance.DefaultPolicyGuid))
				Expect(policySet).To(Equal(serviceInstance.DefaultPolicy))
				Expect(appsUpdated).To(Equal([]string{"app-id-1", "app-id-2"}))

				By("updating the scheduler")
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
		Context("When a default policy is present and there was previously a default policy", func() {
			BeforeEach(func() {
				d := json.RawMessage(testDefaultPolicy)
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					Parameters: &models.InstanceParameters{
						DefaultPolicy: &d,
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
					DefaultPolicy:     "a-default-policy",
					DefaultPolicyGuid: "a-default-policy-guid",
				}, nil)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-1", "app-id-2"}, nil)
				policydb.SetOrUpdateDefaultAppPolicyReturns([]string{"app-id-2"}, nil)
				verifyScheduleIsUpdatedInScheduler("app-id-2", testDefaultPolicy)
				fakecfClient.GetServicePlanReturns("a-plan-id", nil)
			})
			It("succeeds with 200, saves the default policy, and updates the default policy", func() {
				By("returning 200")
				Expect(resp.Code).To(Equal(http.StatusOK))

				By("saving the default policy")
				Expect(bindingdb.UpdateServiceInstanceCallCount()).To(Equal(1))
				serviceInstance := bindingdb.UpdateServiceInstanceArgsForCall(0)
				Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
				Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
				Expect(serviceInstance.DefaultPolicyGuid).To(HaveLen(36))

				By("setting the default policy on the already bound apps")
				Expect(policydb.SetOrUpdateDefaultAppPolicyCallCount()).To(Equal(1))
				appToUpdate, oldPolicyGuid, newPolicy, newPolicyGuid := policydb.SetOrUpdateDefaultAppPolicyArgsForCall(0)
				Expect(appToUpdate).To(Equal([]string{"app-id-1", "app-id-2"}))
				Expect(oldPolicyGuid).To(Equal("a-default-policy-guid"))
				Expect(newPolicyGuid).To(Equal(serviceInstance.DefaultPolicyGuid))
				Expect(newPolicy).To(Equal(serviceInstance.DefaultPolicy))

				By("updating the scheduler")
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
		Context("When the default is set to be removed and there was previously a default policy", func() {
			BeforeEach(func() {
				emptyJsonObject := json.RawMessage("\n{\n}\n")
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
					},
					Parameters: &models.InstanceParameters{
						DefaultPolicy: &emptyJsonObject,
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
					DefaultPolicy:     "a-default-policy",
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
				fakecfClient.GetServicePlanReturns("a-plan-id", nil)
			})
			It("succeeds with 200 and removes the default policy", func() {
				By("returning 200")
				Expect(resp.Code).To(Equal(http.StatusOK), DebugTestInfo())

				By("removing the default policy")
				Expect(bindingdb.UpdateServiceInstanceCallCount()).To(Equal(1))
				serviceInstance := bindingdb.UpdateServiceInstanceArgsForCall(0)
				Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
				Expect(serviceInstance.DefaultPolicy).To(Equal(""))
				Expect(serviceInstance.DefaultPolicyGuid).To(Equal(""))

				By("removing the default policy on the already bound apps")
				Expect(policydb.DeletePoliciesByPolicyGuidCallCount()).To(Equal(1))
				removedPolicy := policydb.DeletePoliciesByPolicyGuidArgsForCall(0)
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
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					Parameters: &models.InstanceParameters{
						DefaultPolicy: &m,
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
				}, nil)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-2", "app-id-1"}, nil)
				fakecfClient.GetServicePlanReturns("a-plan-id", nil)
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(Equal(`{"code":"Bad Request","message":"Too many scaling rules: Found 2 scaling rules, but a maximum of 1 scaling rules are allowed for this service plan. "}`))
			})
		})

		Context("When the service plan is updatable", func() {
			BeforeEach(func() {
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id-not-updatable",
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				fakecfClient.GetServicePlanReturns("a-plan-id", nil)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
				}, nil)
			})
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("The service plan is updated and a default policy was present previously", func() {
			BeforeEach(func() {
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id-2",
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				fakecfClient.GetServicePlanReturns("a-plan-id", nil)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
					DefaultPolicy:     testDefaultPolicy,
					DefaultPolicyGuid: "default-policy-guid",
				}, nil)
			})

			It("Succeeds and leaves the old default policy in place", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				serviceInstance := bindingdb.UpdateServiceInstanceArgsForCall(0)
				Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
				Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
				Expect(serviceInstance.DefaultPolicyGuid).To(Equal("default-policy-guid"))
			})
		})

		Context("When the service plan is not updatable", func() {
			BeforeEach(func() {
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				fakecfClient.GetServicePlanReturns("a-plan-id-not-updatable", nil)
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
				}, nil)
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"The plan is not updatable"}`))
			})
		})
		Context("Update service plan and policy both are updated together", func() {
			BeforeEach(func() {
				d := json.RawMessage(testDefaultPolicy)
				instanceUpdateRequestBody = &models.InstanceUpdateRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id-not-updatable",
					},
					Parameters: &models.InstanceParameters{
						DefaultPolicy: &d,
					},
				}
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{
					ServiceInstanceId: testInstanceId,
					DefaultPolicy:     "a-default-policy",
					DefaultPolicyGuid: "a-default-policy-guid",
				}, nil)
				policydb.SetOrUpdateDefaultAppPolicyReturns([]string{"app-id-2"}, nil)
				verifyScheduleIsUpdatedInScheduler("app-id-2", testDefaultPolicy)
				fakecfClient.GetServicePlanReturns("a-plan-id", nil)
				bindingdb.GetAppIdsByInstanceIdReturns([]string{"app-id-1", "app-id-2"}, nil)
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				By("saving the default policy")
				Expect(bindingdb.UpdateServiceInstanceCallCount()).To(Equal(1))
				serviceInstance := bindingdb.UpdateServiceInstanceArgsForCall(0)
				Expect(serviceInstance.ServiceInstanceId).To(Equal(testInstanceId))
				Expect(serviceInstance.DefaultPolicy).To(MatchJSON(testDefaultPolicy))
				Expect(serviceInstance.DefaultPolicyGuid).To(HaveLen(36))

				By("setting the default policy on the already bound apps")
				Expect(policydb.SetOrUpdateDefaultAppPolicyCallCount()).To(Equal(1))
				appToUpdate, oldPolicyGuid, newPolicy, newPolicyGuid := policydb.SetOrUpdateDefaultAppPolicyArgsForCall(0)
				Expect(appToUpdate).To(Equal([]string{"app-id-1", "app-id-2"}))
				Expect(oldPolicyGuid).To(Equal("a-default-policy-guid"))
				Expect(newPolicyGuid).To(Equal(serviceInstance.DefaultPolicyGuid))
				Expect(newPolicy).To(Equal(serviceInstance.DefaultPolicy))

				By("updating the scheduler")
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
	})

	Describe("DeleteServiceInstance", func() {
		JustBeforeEach(func() {
			handler.DeleteServiceInstance(resp, req, map[string]string{"instanceId": testInstanceId})
		})

		Context("When database DeleteServiceInstance call returns ErrDoesnotExist", func() {
			BeforeEach(func() {
				instanceDeletionRequestBody := &models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
					PlanID:    "a-plan-id",
				}
				body, err := json.Marshal(instanceDeletionRequestBody)
				Expect(err).NotTo(HaveOccurred())
				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())

				bindingdb.DeleteServiceInstanceReturns(db.ErrDoesNotExist)
			})
			It("fails with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Gone","message":"Service Instance Doesn't Exist"}`))
			})
		})

		Context("When database DeleteServiceInstance call returns error other than ErrDoesnotExist", func() {
			BeforeEach(func() {
				instanceDeletionRequestBody := &models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
					PlanID:    "a-plan-id",
				}
				body, err := json.Marshal(instanceDeletionRequestBody)
				Expect(err).NotTo(HaveOccurred())
				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())

				bindingdb.DeleteServiceInstanceReturns(fmt.Errorf("error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error deleting service instance"}`))
			})
		})

		Context("When all mandatory parameters are present", func() {
			BeforeEach(func() {
				instanceDeletionRequestBody := &models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
					PlanID:    "a-plan-id",
				}
				body, err := json.Marshal(instanceDeletionRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())
			})
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK), DebugTestInfo())
			})
		})

		Context("When service bindings are present", func() {
			BeforeEach(func() {
				var bindingIds []string
				bindingIds = append(bindingIds, testBindingId)

				bindingdb.GetBindingIdsByInstanceIdReturns(bindingIds, nil)
				bindingdb.GetAppIdByBindingIdReturnsOnCall(0, testAppId, nil)
				instanceDeletionRequestBody := &models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
					PlanID:    "a-plan-id",
				}
				body, err := json.Marshal(instanceDeletionRequestBody)
				Expect(err).NotTo(HaveOccurred())
				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
				Expect(err).ToNot(HaveOccurred())
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("if it has been deleted", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				Expect(bindingdb.DeleteServiceBindingCallCount()).To(Equal(1))
				Expect(bindingdb.DeleteServiceBindingArgsForCall(0)).To(Equal(testBindingId))
				Expect(bindingdb.DeleteServiceInstanceCallCount()).To(Equal(1))
				Expect(policydb.DeletePolicyCallCount(), 1)
				Expect(policydb.DeletePolicyArgsForCall(0), testAppId)
			})
		})

		Context("When service bindings are present", func() {
			BeforeEach(func() {
				var bindingIds []string
				bindingIds = append(bindingIds, testBindingId)

				bindingdb.GetBindingIdsByInstanceIdReturns(bindingIds, nil)
				bindingdb.GetAppIdByBindingIdReturnsOnCall(0, testAppId, nil)
				instanceDeletionRequestBody := &models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
					PlanID:    "a-plan-id",
				}
				body, err := json.Marshal(instanceDeletionRequestBody)
				Expect(err).NotTo(HaveOccurred())
				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("if it has been deleted", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				Expect(bindingdb.DeleteServiceBindingCallCount()).To(Equal(1))
				Expect(bindingdb.DeleteServiceBindingArgsForCall(0)).To(Equal(testBindingId))
				Expect(bindingdb.DeleteServiceInstanceCallCount()).To(Equal(1))
				Expect(policydb.DeletePolicyCallCount(), 1)
				Expect(policydb.DeletePolicyArgsForCall(0), testAppId)
			})
		})

	})

	Describe("BindServiceInstance", func() {
		var (
			err                error
			bindingRequestBody *models.BindingRequestBody
			bindingPolicy      string
			body               []byte
		)
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
					}]
				}`
			bindingRequestBody = &models.BindingRequestBody{
				AppID: "an-app-id",
				BrokerCommonRequestBody: models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
					PlanID:    "a-plan-id",
				},
				Policy: json.RawMessage(bindingPolicy),
			}
		})
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			handler.BindServiceInstance(resp, req, map[string]string{"instanceId": testInstanceId, "bindingId": "a-binding-id"})
		})
		Context("When request body is not a valid json", func() {
			BeforeEach(func() {
				body = []byte("")
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Invalid request body format"}`))
			})
		})
		Context("When mandatory parameters are not provided", func() {
			Context("When AppID is not provided", func() {
				BeforeEach(func() {
					bindingRequestBody.AppID = ""
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
			})

			Context("When ServiceID is not provided", func() {
				BeforeEach(func() {
					bindingRequestBody.ServiceID = ""
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
			})

			Context("When PlanID is not provided", func() {
				BeforeEach(func() {
					bindingRequestBody.PlanID = ""
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
				})
				It("fails with 400", func() {
					Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
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
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					Policy: json.RawMessage(bindingPolicy),
				}

				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Too many scaling rules: Found 2 scaling rules, but a maximum of 1 scaling rules are allowed for this service plan. "}`))
			})
		})
		Context("When mandatory parameters are present", func() {
			BeforeEach(func() {
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
				verifyScheduleIsUpdatedInScheduler(testAppId, bindingPolicy)
			})
			It("succeeds with 201", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated), DebugTestInfo())

				By("updating the scheduler")
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
			It("returns the correct binding parameters", func() {
				creds := &models.CredentialResponse{}
				responseString := resp.Body.String()
				err := json.Unmarshal([]byte(responseString), creds)
				Expect(err).NotTo(HaveOccurred())
				Expect(creds.Credentials.CustomMetrics.URL).To(Equal("someURL"))
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
							}]
						}`
				BeforeEach(func() {
					bindingRequestBody = &models.BindingRequestBody{
						AppID: testAppId,
						BrokerCommonRequestBody: models.BrokerCommonRequestBody{
							ServiceID: "a-service-id",
							PlanID:    "a-plan-id",
						},
						Policy: json.RawMessage(testBindingPolicy),
					}
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())

					verifyScheduleIsUpdatedInScheduler(testAppId, testBindingPolicy)
				})
				It("succeeds with 201 and saves the binding's policy", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated), DebugTestInfo())
					Expect(policydb.SaveAppPolicyCallCount()).To(Equal(1))
					appID, policy, _ := policydb.SaveAppPolicyArgsForCall(0)
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
							ServiceID: "a-service-id",
							PlanID:    "a-plan-id",
						},
					}
					body, err = json.Marshal(bindingRequestBody)
					Expect(err).NotTo(HaveOccurred())
					verifyScheduleIsUpdatedInScheduler(testAppId, testDefaultPolicy)
				})
				It("succeeds with 201 and saves the default policy", func() {
					Expect(resp.Code).To(Equal(http.StatusCreated), DebugTestInfo())
					Expect(policydb.SaveAppPolicyCallCount()).To(Equal(1))
					appID, policy, _ := policydb.SaveAppPolicyArgsForCall(0)
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
				Expect(resp.Code).To(Equal(http.StatusConflict), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Conflict","message":"An autoscaler service instance is already bound to the application. Multiple bindings are not supported."}`))
			})
		})

		Context("When database CreateServiceBinding call returns error other than ErrAlreadyExists", func() {
			BeforeEach(func() {
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.CreateServiceBindingReturns(fmt.Errorf("some sql error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error creating service binding"}`))
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
				Expect(resp.Code).To(Equal(http.StatusInternalServerError), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error creating service binding"}`))
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
				Expect(resp.Code).To(Equal(http.StatusBadRequest), DebugTestInfo())
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(Equal(`[{"context":"(root)","description":"instance_min_count is required"}]`))
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
		var err error
		BeforeEach(func() {
			bindingdb.GetAppIdByBindingIdReturns(testAppId, nil)
		})
		JustBeforeEach(func() {
			handler.UnbindServiceInstance(resp, req, map[string]string{"instanceId": testInstanceId, "bindingId": "a-binding-id"})
		})
		Context("When mandatory parameters are present", func() {
			BeforeEach(func() {
				req, err = http.NewRequest(http.MethodDelete, "", nil)
				Expect(err).NotTo(HaveOccurred())
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK), DebugTestInfo())
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
		Context("When there is no app with the bindingId", func() {
			BeforeEach(func() {
				req, err = http.NewRequest(http.MethodDelete, "", nil)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetAppIdByBindingIdReturns("", sql.ErrNoRows)
			})
			AfterEach(func() {
				bindingdb.GetAppIdByBindingIdReturns(testAppId, nil)
			})
			It("succeed with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Gone","message":"Failed to unbind service: Service binding does not exist"}`))
			})
		})
		Context("When failed to get appId by bindingId", func() {
			BeforeEach(func() {
				req, err = http.NewRequest(http.MethodDelete, "", nil)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetAppIdByBindingIdReturns("", errors.New("some error"))
			})
			AfterEach(func() {
				bindingdb.GetAppIdByBindingIdReturns(testAppId, nil)
			})
			It("succeed with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Failed to unbind service: Error deleting service binding"}`))
			})
		})
		Context("When database DeleteServiceBinding call returns ErrDoesnotExist", func() {
			BeforeEach(func() {
				req, err = http.NewRequest(http.MethodDelete, "", nil)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.DeleteServiceBindingReturns(db.ErrDoesNotExist)
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("fails with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Gone","message":"Failed to unbind service: Service binding does not exist"}`))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})

		Context("When database DeleteServiceBinding call returns error other than ErrDoesnotExist", func() {
			BeforeEach(func() {
				req, err = http.NewRequest(http.MethodDelete, "", nil)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.DeleteServiceBindingReturns(fmt.Errorf("some sql error"))
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError), DebugTestInfo())
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Failed to unbind service: Error deleting service binding"}`))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
	})
})

func DebugTestInfo() string {
	quota := debugMock("quota", quotaServer)
	token := debugMock("token", tokenServer)
	return fmt.Sprintf("%s\n%s\n", quota, token)
}

func debugMock(name string, server *ghttp.Server) string {
	handler := server.GetHandler(0)
	s := "nope"
	if handler != nil {
		s = "yup"
	}
	return fmt.Sprintf("State of %s mock server handler handler 0:%s url:%s requests:%+v", name, s, server.URL(), server.ReceivedRequests())
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

func installQuotaAPIHandlers() {
	/// only provide them for tests using them, or reset them if not used?
	tokenServer.Reset()
	tokenServer.AppendHandlers(ghttp.CombineHandlers(
		ghttp.VerifyBasicAuth("client-id", "client-secret"),
		ghttp.VerifyRequest("POST", "/"),
		ghttp.RespondWithJSONEncoded(http.StatusOK, oauth2.Token{
			AccessToken:  "secret-token",
			TokenType:    "bearer",
			RefreshToken: "refresh-token",
		}),
	))
	quotaServer.Reset()
	quotaServer.AppendHandlers(ghttp.CombineHandlers(
		ghttp.VerifyHeaderKV("Authorization", "Bearer secret-token"),
		ghttp.VerifyRequest("GET", "/api/v2.0/orgs/an-org-guid/services/autoscaler/plan/standard"),
		ghttp.RespondWithJSONEncoded(http.StatusOK, struct {
			Quota int `json:"quota"`
		}{Quota: -1}),
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
