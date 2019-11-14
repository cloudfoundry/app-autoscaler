package brokerserver_test

import (
	. "autoscaler/api/brokerserver"
	"autoscaler/db"
	"autoscaler/fakes"
	"autoscaler/models"
	"autoscaler/routes"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/onsi/gomega/ghttp"

	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BrokerHandler", func() {
	var (
		bindingdb *fakes.FakeBindingDB
		policydb  *fakes.FakePolicyDB

		handler *BrokerHandler
		resp    *httptest.ResponseRecorder
		req     *http.Request
	)
	BeforeEach(func() {
		bindingdb = &fakes.FakeBindingDB{}
		policydb = &fakes.FakePolicyDB{}
		resp = httptest.NewRecorder()

		handler = NewBrokerHandler(lagertest.NewTestLogger("test"), conf, bindingdb, policydb)

	})

	Describe("GetBrokerCatalog", func() {
		JustBeforeEach(func() {
			handler.GetBrokerCatalog(resp, req, map[string]string{})
		})
		Context("When getBrokerCatalog is called", func() {
			It("gets the catalog json", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
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
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
			})

		})
		Context("When database CreateServiceInstance call returns ErrAlreadyExists", func() {
			BeforeEach(func() {
				body, err = json.Marshal(instanceCreationReqBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.CreateServiceInstanceReturns(db.ErrAlreadyExists)
				conf.DashboardRedirectURI = ""
			})
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
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
				Expect(resp.Code).To(Equal(http.StatusOK))
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
				Expect(resp.Code).To(Equal(http.StatusConflict))
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
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
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
				Expect(resp.Code).To(Equal(http.StatusCreated))
				Expect(resp.Body.Bytes()).To(Equal([]byte("{\"dashboard_url\":\"https://service-dashboard-url.com/manage/an-instance-id\"}")))
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
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(Equal(`[{"context":"(root)","description":"instance_max_count is required"}]`))
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
				Expect(resp.Code).To(Equal(http.StatusCreated))
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
					PlanID:    "a-plan-id",
				},
			}
		})
		Context("When request body is not a valid json", func() {
			BeforeEach(func() {
				body = []byte("")
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
			})
		})
		Context("When no default policy update is performed", func() {
			BeforeEach(func() {
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
			})
			It("fails with 422", func() {
				Expect(resp.Code).To(Equal(http.StatusUnprocessableEntity))
				Expect(resp.Body.String()).To(Equal(`{"code":"Unprocessable Entity","message":"Failed to update service instance: Only default_policy updates allowed"}`))
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
						PlanID:    "a-plan-id",
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
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
				Expect(resp.Code).To(Equal(http.StatusNotFound))
			})
		})
		Context("When all mandatory parameters are present", func() {
			BeforeEach(func() {
				emptyPolicyParameter := json.RawMessage("\n{\t}\n")
				parameters := models.InstanceParameters{DefaultPolicy: &emptyPolicyParameter}
				instanceUpdateRequestBody.Parameters = &parameters
				body, err = json.Marshal(instanceUpdateRequestBody)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.GetServiceInstanceReturns(&models.ServiceInstance{}, nil)
			})
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
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
			})
			It("succeeds with 200, saves the default policy, and sets the default policy on the already bound apps", func() {
				By("returning 200")
				Expect(resp.Code).To(Equal(http.StatusOK))

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
						PlanID:    "a-plan-id",
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
				verifyScheduleIsDeletedInScheduler("app-id-2")
			})
			It("succeeds with 200 and removes the default policy", func() {
				By("returning 200")
				Expect(resp.Code).To(Equal(http.StatusOK))

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

				bindingdb.DeleteServiceInstanceReturns(db.ErrDoesNotExist)
			})
			It("fails with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone))
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

				bindingdb.DeleteServiceInstanceReturns(fmt.Errorf("error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
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
			})
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
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
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
					Expect(resp.Code).To(Equal(http.StatusBadRequest))
					Expect(resp.Body.String()).To(Equal(`{"code":"Bad Request","message":"Malformed or missing mandatory data"}`))
				})
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
					Expect(resp.Code).To(Equal(http.StatusCreated))
					Expect(policydb.SaveAppPolicyCallCount()).To(Equal(1))
					appID, policy, _ := policydb.SaveAppPolicyArgsForCall(0)
					Expect(appID).To(Equal(testAppId))
					Expect(policy).NotTo(MatchJSON(string(testDefaultPolicy)))
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
					Expect(resp.Code).To(Equal(http.StatusCreated))
					Expect(policydb.SaveAppPolicyCallCount()).To(Equal(1))
					appID, policy, _ := policydb.SaveAppPolicyArgsForCall(0)
					Expect(appID).To(Equal(testAppId))
					Expect(policy).To(MatchJSON(string(testDefaultPolicy)))

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
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error creating service binding"}`))
			})
		})
		Context("When failed to create credential", func() {
			BeforeEach(func() {
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
				policydb.GetCredentialReturns(nil, sql.ErrNoRows)
				policydb.SaveCredentialReturns(fmt.Errorf("some sql error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
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
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				bodyBytes, err := ioutil.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(bodyBytes)).To(Equal(`[{"context":"(root)","description":"instance_min_count is required"}]`))
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
				Expect(resp.Code).To(Equal(http.StatusOK))
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
				Expect(resp.Code).To(Equal(http.StatusGone))
				Expect(resp.Body.String()).To(Equal(`{"code":"Gone","message":"Binding does not exist"}`))
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
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error deleting service binding"}`))
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
				Expect(resp.Code).To(Equal(http.StatusGone))
				Expect(resp.Body.String()).To(Equal(`{"code":"Gone","message":"Service Binding Doesn't Exist"}`))
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
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error deleting service binding"}`))
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
			})
		})
	})
})

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
