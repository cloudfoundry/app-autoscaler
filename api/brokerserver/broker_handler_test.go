package brokerserver_test

import (
	. "autoscaler/api/brokerserver"
	"autoscaler/db"
	"autoscaler/fakes"
	"autoscaler/models"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

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
			handler.CreateServiceInstance(resp, req, map[string]string{"instanceId": "an-instance-id"})
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

	})

	Describe("DeleteServiceInstance", func() {
		JustBeforeEach(func() {
			handler.DeleteServiceInstance(resp, req, map[string]string{"instanceId": "an-instance-id"})
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
			It("fails with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

	})

	Describe("BindServiceInstance", func() {
		var err error
		var bindingRequestBody *models.BindingRequestBody
		var body []byte
		BeforeEach(func() {
			bindingRequestBody = &models.BindingRequestBody{
				AppID: "an-app-id",
				BrokerCommonRequestBody: models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
					PlanID:    "a-plan-id",
				},
				Policy: json.RawMessage(`{
					"instance_max_count":4,
					"instance_min_count":1,
					"scaling_rules":[
					{
						"metric_type":"memoryused",
						"threshold":30,
						"operator":"<",
						"adjustment":"-1"
					}]
				}`),
			}
		})
		JustBeforeEach(func() {
			req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			handler.BindServiceInstance(resp, req, map[string]string{"instanceId": "an-instance-id", "bindingId": "a-binding-id"})
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
			})
			It("succeeds with 201", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated))
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

		Context("When called with valid policy json", func() {
			BeforeEach(func() {
				body, err = json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
			})
			It("succeeds with 201", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated))
			})
		})
	})

	Describe("UnBindServiceInstance", func() {
		var err error
		BeforeEach(func() {
			bindingdb.GetAppIdByBindingIdReturns(testAppId, nil)
		})
		JustBeforeEach(func() {
			handler.UnbindServiceInstance(resp, req, map[string]string{"instanceId": "an-instance-id", "bindingId": "a-binding-id"})
		})
		Context("When mandatory parameters are present", func() {
			BeforeEach(func() {
				req, err = http.NewRequest(http.MethodDelete, "", nil)
				Expect(err).NotTo(HaveOccurred())
			})
			It("succeed with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
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
			})
			It("fails with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone))
				Expect(resp.Body.String()).To(Equal(`{"code":"Gone","message":"Service Binding Doesn't Exist"}`))
			})
		})

		Context("When database DeleteServiceBinding call returns error other than ErrDoesnotExist", func() {
			BeforeEach(func() {
				req, err = http.NewRequest(http.MethodDelete, "", nil)
				Expect(err).NotTo(HaveOccurred())
				bindingdb.DeleteServiceBindingReturns(fmt.Errorf("some sql error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
				Expect(resp.Body.String()).To(Equal(`{"code":"Internal Server Error","message":"Error deleting service binding"}`))
			})
		})
	})
})
