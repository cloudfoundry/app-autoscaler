package broker_test

import (
	"encoding/json"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/broker"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"context"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
)

var _ = Describe("Broker", func() {
	var (
		broker          *Broker
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
		broker = New(testLogger, conf, fakeBindingDB, fakePolicyDB, services, fakeCredentials)
	})

	Describe("Services", func() {
		var retrievedServices []domain.Service
		JustBeforeEach(func() {
			retrievedServices, err = broker.Services(context.TODO())
		})
		It("returns the retrievedServices", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(retrievedServices).To(Equal(services))
		})
	})

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
			instance, err = broker.GetInstance(context.TODO(), testInstanceId, fetchInstanceDetails)
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
							DefaultPolicy: &policy,
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
			Binding, err = broker.GetBinding(context.TODO(), testInstanceId, testBindingId, fetchBindingDetails)
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
			BeforeEach(func() {
				fakeBindingDB.GetServiceBindingReturns(&models.ServiceBinding{
					ServiceBindingID:  testBindingId,
					ServiceInstanceID: testInstanceId,
					AppID:             testAppId,
				}, nil)
			})
			Context("without policy", func() {
				BeforeEach(func() {
					fakePolicyDB.GetAppPolicyReturns(nil, db.ErrDoesNotExist)
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
						Expect(Binding).To(Equal(domain.GetBindingSpec{}))
					})
				})
			})
			Context("with policy", func() {
				BeforeEach(func() {
					fakePolicyDB.GetAppPolicyReturns(scalingPolicy, nil)
				})
				It("returns the Binding with parameters", func() {
					Expect(err).To(BeNil())
					Expect(Binding).To(Equal(domain.GetBindingSpec{
						Parameters: scalingPolicy,
					}))
				})
			})
		})
	})

})
