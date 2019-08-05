package brokerserver_test

import (
	. "autoscaler/api/brokerserver"
	"autoscaler/db"
	"autoscaler/fakes"
	"autoscaler/models"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager"

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

		handler = NewBrokerHandler(lager.NewLogger("test"), conf, bindingdb, policydb)

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
		JustBeforeEach(func() {
			handler.CreateServiceInstance(resp, req, map[string]string{"instanceId": "an-instance-id"})
		})
		Context("When mandatory parameters are not provided", func() {
			BeforeEach(func() {
				instanceCreationReqBody := &models.InstanceCreationRequestBody{
					OrgGUID: "an-org-guid",
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
				}
				body, err := json.Marshal(instanceCreationReqBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("When all mandatory parameters are present", func() {
			BeforeEach(func() {
				instanceCreationReqBody := &models.InstanceCreationRequestBody{
					OrgGUID:   "an-org-guid",
					SpaceGUID: "a-space-guid",
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
				}
				body, err := json.Marshal(instanceCreationReqBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			})
			It("succeeds with 201", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated))
			})
		})

		Context("When database CreateServiceInstance call returns ErrAlreadyExists", func() {
			BeforeEach(func() {
				instanceCreationReqBody := &models.InstanceCreationRequestBody{
					OrgGUID:   "an-org-guid",
					SpaceGUID: "a-space-guid",
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
				}
				body, err := json.Marshal(instanceCreationReqBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))

				bindingdb.CreateServiceInstanceReturns(db.ErrAlreadyExists)
			})
			It("succeeds with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When database CreateServiceInstance call returns error other than ErrAlreadyExists", func() {
			BeforeEach(func() {
				instanceCreationReqBody := &models.InstanceCreationRequestBody{
					OrgGUID:   "an-org-guid",
					SpaceGUID: "a-space-guid",
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
				}
				body, err := json.Marshal(instanceCreationReqBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))

				bindingdb.CreateServiceInstanceReturns(fmt.Errorf("some sql error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("When dashboard redirect uri is present in config", func() {
			BeforeEach(func() {
				instanceCreationReqBody := &models.InstanceCreationRequestBody{
					OrgGUID:   "an-org-guid",
					SpaceGUID: "a-space-guid",
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
				}
				body, err := json.Marshal(instanceCreationReqBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
				conf.DashboardRedirectURI = "https://service-dashboard-url.com"
			})
			It("succeeds with 201 and returns dashboard_url", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated))
				Expect(resp.Body.Bytes()).To(Equal([]byte("{\"dashboard_url\":\"https://service-dashboard-url.com/manage/an-instance-id\"}")))
			})
		})
	})

	Describe("DeleteServiceInstance", func() {
		JustBeforeEach(func() {
			handler.DeleteServiceInstance(resp, req, map[string]string{"instanceId": "an-instance-id"})
		})
		Context("When mandatory parameters are not provided", func() {
			BeforeEach(func() {
				instanceDeletionRequestBody := &models.BrokerCommonRequestBody{
					ServiceID: "a-service-id",
				}
				body, err := json.Marshal(instanceDeletionRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
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
			})
		})

	})

	Describe("BindServiceInstance", func() {
		JustBeforeEach(func() {
			handler.BindServiceInstance(resp, req, map[string]string{"instanceId": "an-instance-id", "bindingId": "a-binding-id"})
		})
		Context("When mandatory parameters are not provided", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.BindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
					},
					AppID: "an-app-id",
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("When appId is not present", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.BindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("When mandatory parameters are present", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.BindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					AppID: "an-app-id",
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			})
			It("succeeds with 201", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated))
			})
		})

		Context("When database CreateServiceBinding call returns ErrAlreadyExists", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.BindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					AppID: "an-app-id",
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))

				bindingdb.CreateServiceBindingReturns(db.ErrAlreadyExists)
			})
			It("fails with 409", func() {
				Expect(resp.Code).To(Equal(http.StatusConflict))
			})
		})

		Context("When database CreateServiceBinding call returns error other than ErrAlreadyExists", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.BindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					AppID: "an-app-id",
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))

				bindingdb.CreateServiceBindingReturns(fmt.Errorf("some sql error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})
		Context("When failed to create custom metrics credential", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.BindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					AppID: "an-app-id",
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
				policydb.GetCustomMetricsCredsReturns(nil, sql.ErrNoRows)
				policydb.SaveCustomMetricsCredReturns(fmt.Errorf("some sql error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("When called with invalid policy json", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.BindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					AppID: testAppId,
					Policy: `{
						"instance_max_count":4,
						"scaling_rules":[
						{
							"metric_type":"memoryused",
							"threshold":30,
							"operator":"<",
							"adjustment":"-1"
						}]
					}`,
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
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
				bindingRequestBody := &models.BindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					AppID: testAppId,
					Policy: `{
						"instance_max_count":4,
						"instance_min_count":1,
						"scaling_rules":[
						{
							"metric_type":"memoryused",
							"threshold":30,
							"operator":"<",
							"adjustment":"-1"
						}]
					}`,
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())
				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			})
			It("succeeds with 201", func() {
				Expect(resp.Code).To(Equal(http.StatusCreated))
			})
		})
	})

	Describe("UnBindServiceInstance", func() {
		JustBeforeEach(func() {
			handler.UnbindServiceInstance(resp, req, map[string]string{"instanceId": "an-instance-id", "bindingId": "a-binding-id"})
		})
		Context("When mandatory parameters are not provided", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.UnbindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
					},
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			})
			It("fails with 400", func() {
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("When mandatory parameters are present", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.UnbindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					AppID: testAppId,
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
			})
			It("fails with 200", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When database DeleteServiceBinding call returns ErrDoesnotExist", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.UnbindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					AppID: testAppId,
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))

				bindingdb.DeleteServiceBindingReturns(db.ErrDoesNotExist)
			})
			It("fails with 410", func() {
				Expect(resp.Code).To(Equal(http.StatusGone))
			})
		})

		Context("When database DeleteServiceBinding call returns error other than ErrDoesnotExist", func() {
			BeforeEach(func() {
				bindingRequestBody := &models.UnbindingRequestBody{
					BrokerCommonRequestBody: models.BrokerCommonRequestBody{
						ServiceID: "a-service-id",
						PlanID:    "a-plan-id",
					},
					AppID: testAppId,
				}
				body, err := json.Marshal(bindingRequestBody)
				Expect(err).NotTo(HaveOccurred())

				req, err = http.NewRequest(http.MethodPut, "", bytes.NewReader(body))

				bindingdb.DeleteServiceBindingReturns(fmt.Errorf("some sql error"))
			})
			It("fails with 500", func() {
				Expect(resp.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})
})
