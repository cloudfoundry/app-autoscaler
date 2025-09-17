package broker_test

import (
	"encoding/json"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"context"

	"code.cloudfoundry.org/brokerapi/v13/domain"
	"code.cloudfoundry.org/brokerapi/v13/domain/apiresponses"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker", func() {
	var (
		aBroker         *broker.Broker
		err             error
		fakeBindingDB   *fakes.FakeBindingDB
		fakePolicyDB    *fakes.FakePolicyDB
		fakeCredentials *fakes.FakeCredentials
		testLogger      = lagertest.NewTestLogger("test")
	)

	BeforeEach(func() {
		fakeBindingDB = &fakes.FakeBindingDB{}
		fakePolicyDB = &fakes.FakePolicyDB{}
		fakeCredentials = &fakes.FakeCredentials{}
	})
	JustBeforeEach(func() {
		aBroker = broker.New(testLogger, conf, fakeBindingDB, fakePolicyDB, services, fakeCredentials)
	})

	Describe("Services", func() {
		var retrievedServices []domain.Service
		JustBeforeEach(func() {
			retrievedServices, err = aBroker.Services(context.TODO())
		})
		It("returns the retrievedServices", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(retrievedServices).To(Equal(services))
		})
	})

	// 🚧 To-do: Untested functions: “Provision” and “Deprovision”

	Describe("GetInstance", func() {
		var instance domain.GetInstanceDetailsSpec
		var fetchInstanceDetails domain.FetchInstanceDetails
		BeforeEach(func() {
			fetchInstanceDetails = domain.FetchInstanceDetails{
				ServiceID: testServiceID,
				PlanID:    testPlanID,
			}
		})
		JustBeforeEach(func() {
			instance, err = aBroker.GetInstance(context.TODO(), testInstanceId, fetchInstanceDetails)
		})
		Context("when there is no instance", func() {
			BeforeEach(func() {
				fakeBindingDB.GetServiceInstanceReturns(nil, db.ErrDoesNotExist)
			})
			It("returns the appropriate error response", func() {
				By("querying the DB", func() {
					Expect(fakeBindingDB.GetServiceInstanceCallCount()).To(Equal(1))
					ctx, instanceId := fakeBindingDB.GetServiceInstanceArgsForCall(0)
					Expect(ctx).NotTo(BeNil())
					Expect(instanceId).To(Equal(testInstanceId))
				})
				By("returning apiresponses.ErrInstanceDoesNotExist", func() {
					Expect(err).To(Equal(apiresponses.ErrInstanceDoesNotExist))
				})
			})
		})
		Context("when the instance exists", func() {
			Context("without default policy", func() {
				BeforeEach(func() {
					fakeBindingDB.GetServiceInstanceReturns(&models.ServiceInstance{
						ServiceInstanceId: testInstanceId,
						OrgId:             testOrgId,
						SpaceId:           testSpaceId,
						DefaultPolicy:     "",
						DefaultPolicyGuid: "",
					}, nil)
				})
				It("returns the instance without parameters", func() {
					Expect(err).ShouldNot(HaveOccurred())
					Expect(instance).To(Equal(domain.GetInstanceDetailsSpec{
						ServiceID:    testServiceID,
						PlanID:       testPlanID,
						DashboardURL: dashBoardURL + "/manage/" + testInstanceId,
						Parameters:   nil,
						Metadata:     domain.InstanceMetadata{},
					}))
				})
			})
			Context("with default policy", func() {
				BeforeEach(func() {
					fakeBindingDB.GetServiceInstanceReturns(&models.ServiceInstance{
						ServiceInstanceId: testInstanceId,
						OrgId:             testOrgId,
						SpaceId:           testSpaceId,
						DefaultPolicy:     testDefaultPolicy,
						DefaultPolicyGuid: testDefaultGuid,
					}, nil)
				})
				It("returns the instance with parameters", func() {
					Expect(err).ShouldNot(HaveOccurred())
					policy := json.RawMessage(testDefaultPolicy)
					Expect(instance).To(Equal(domain.GetInstanceDetailsSpec{
						ServiceID:    testServiceID,
						PlanID:       testPlanID,
						DashboardURL: dashBoardURL + "/manage/" + testInstanceId,
						Parameters: models.InstanceParameters{
							DefaultPolicy: policy,
						},
						Metadata: domain.InstanceMetadata{},
					}))
				})
			})
		})
	})

	Describe("GetBinding", func() {
		var Binding domain.GetBindingSpec
		var fetchBindingDetails domain.FetchBindingDetails
		BeforeEach(func() {
			fetchBindingDetails = domain.FetchBindingDetails{
				ServiceID: testServiceID,
				PlanID:    testPlanID,
			}
		})
		JustBeforeEach(func() {
			Binding, err = aBroker.GetBinding(context.TODO(), testInstanceId, testBindingId, fetchBindingDetails)
		})
		Context("when there is no binding", func() {
			BeforeEach(func() {
				fakeBindingDB.GetServiceBindingReturns(nil, db.ErrDoesNotExist)
			})
			It("returns the appropriate error response", func() {
				By("querying the DB", func() {
					Expect(fakeBindingDB.GetServiceBindingCallCount()).To(Equal(1))
					ctx, BindingId := fakeBindingDB.GetServiceBindingArgsForCall(0)
					Expect(ctx).NotTo(BeNil())
					Expect(BindingId).To(Equal(testBindingId))
				})
				By("returning apiresponses.ErrBindingDoesNotExist", func() {
					Expect(err).To(Equal(apiresponses.ErrBindingDoesNotExist))
				})
			})
		})
		Context("when the binding exists", func() {
			Context("with default policy and default custom-metrics-strategy", func() {
				BeforeEach(func() {
					fakeBindingDB.GetServiceBindingReturns(&models.ServiceBinding{
						ServiceBindingID:      testBindingId,
						ServiceInstanceID:     testInstanceId,
						AppID:                 testAppId,
						CustomMetricsStrategy: "same_app",
					}, nil)
					fakePolicyDB.GetAppPolicyReturns(nil, nil)
				})
				It("returns the empty binding without parameters", func() {
					By("querying the DB", func() {
						Expect(fakePolicyDB.GetAppPolicyCallCount()).To(Equal(1))
						ctx, appId := fakePolicyDB.GetAppPolicyArgsForCall(0)
						Expect(ctx).NotTo(BeNil())
						Expect(appId).To(Equal(testAppId))
					})
					By("returning an empty response", func() {
						Expect(err).ShouldNot(HaveOccurred())
						serialisedBinding, err := json.Marshal(Binding)
						Expect(err).ShouldNot(HaveOccurred())
						serializedDefaultBinding, err := json.Marshal(domain.GetBindingSpec{})
						Expect(err).ShouldNot(HaveOccurred())
						Expect(serialisedBinding).To(Equal(serializedDefaultBinding))
					})
				})
			})
			Context("with dedicated policy and default custom-metrics-strategy", func() {
				BeforeEach(func() {
					fakeBindingDB.GetServiceBindingReturns(&models.ServiceBinding{
						ServiceBindingID:      testBindingId,
						ServiceInstanceID:     testInstanceId,
						AppID:                 "",
						CustomMetricsStrategy: "same_app",
					}, nil)
					fakePolicyDB.GetAppPolicyReturns(policyDef, nil)
				})
				It("returns the binding with parameters", func() {
					stgy := models.DefaultCustomMetricsStrategy
					responesParams := domain.GetBindingSpec{
						Parameters: models.NewScalingPolicy(stgy, policyDef),
					}

					Expect(err).To(BeNil())
					Expect(Binding).To(Equal(responesParams))
				})
			})
			Context("with non-default-configuration and default-policy", func() {
				BeforeEach(func() {
					fakeBindingDB.GetServiceBindingReturns(&models.ServiceBinding{
						ServiceBindingID:      testBindingId,
						ServiceInstanceID:     testInstanceId,
						AppID:                 testAppId,
						CustomMetricsStrategy: models.CustomMetricsBoundApp.String(),
					}, nil)

					fakePolicyDB.GetAppPolicyReturns(policyDef, nil)
				})
				It("returns the Binding with configs and policy in parameters", func() {
					Expect(err).To(BeNil())
					Expect(Binding).To(Equal(domain.GetBindingSpec{
						Parameters: models.NewScalingPolicy(
							models.CustomMetricsBoundApp, policyDef)}))
				})
			})
			Context("with configuration only", func() {
				BeforeEach(func() {
					fakeBindingDB.GetServiceBindingReturns(&models.ServiceBinding{
						ServiceBindingID:      testBindingId,
						ServiceInstanceID:     testInstanceId,
						AppID:                 testAppId,
						CustomMetricsStrategy: models.CustomMetricsBoundApp.String(),
					}, nil)
					bindingBytes, err := os.ReadFile("testdata/with-configs.json")
					Expect(err).ShouldNot(HaveOccurred())

					_, err = models.ScalingPolicyFromRawJSON(bindingBytes)
					Expect(err).ShouldNot(HaveOccurred())
					fakePolicyDB.GetAppPolicyReturns(nil, nil)
				})
				It("returns no binding configs in parameters", func() {
					Expect(err).To(BeNil())
					Expect(Binding).To(Equal(domain.GetBindingSpec{
						Parameters: models.NewScalingPolicy(models.CustomMetricsBoundApp, nil),
					}))
				})
			})
		})
	})

	Describe("Bind", func() {
		var ctx context.Context
		var instanceID string
		var bindingID string
		var details domain.BindDetails

		BeforeEach(func() {
			ctx = context.Background()
			instanceID = "some_instance-id"
			bindingID = "some_binding-id"
		})
		Context("Create a binding", func() {
			It("Fails when the additional config-parameter “app-guid” is provided", func() {
				var bindingParams = []byte(`
{
  "configuration": {
	"app-guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",
	"custom_metrics": {
	  "metric_submission_strategy": {
		"allow_from": "bound_app"
	  }
	}
  }
}`)
				details = domain.BindDetails{
					AppGUID:   "", // Deprecated field!
					PlanID:    "some_plan-id",
					ServiceID: "some_service-id",
					BindResource: &domain.BindResource{
						AppGuid: "AppGUID_for_bindings",
						//	SpaceGuid          string `json:"space_guid,omitempty"`
						//	Route              string `json:"route,omitempty"`
						//	CredentialClientID string `json:"credential_client_id,omitempty"`
						//	BackupAgent        bool   `json:"backup_agent,omitempty"`
					}, //  *BindResource

					// RawContext: json.RawMessage // `json:"context,omitempty"`
					RawParameters: bindingParams, // `json:"parameters,omitempty"`
				}

				_, err := aBroker.Bind(ctx, instanceID, bindingID, details, false)

				Expect(err).NotTo(BeNil())
				// 🚧 To-do!
			})

			It("Supports provision of an Autoscaler Policy as RawParameters", func() {

			})
			It("Does not require the provision of an Autoscaler Policy as RawParameters", func() {
				// 🚧 To-do: Check usage of default-policy?
			})
		})
		Context("Create a service-key", func() {
			It("Fails when there is an AppGUID in the BindDetails", func() {

			})
		})
	})
}) // End `Describe "Broker"`
