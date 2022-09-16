package brokerserver

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/pivotal-cf/brokerapi/v8/handlers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/lager"

	"github.com/pivotal-cf/brokerapi/v8/domain"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/lagertest"

	"github.com/onsi/gomega/ghttp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	bindingdb             *fakes.FakeBindingDB
	policydb              *fakes.FakePolicyDB
	fakecfClient          *fakes.FakeCFClient
	fakeCredentials       *fakes.FakeCredentials
	broker                *Broker
	handler               handlers.APIHandler
	conf                  *config.Config
	schedulerServer       = ghttp.NewServer()
	port                  = 10000 + GinkgoParallelProcess()
	testBindingId         = "a-binding-id"
	testServiceInstanceId = "a-service-instance-id"
	testAppId             = "an-app-id"
	catalog               []domain.Service
)

var _ = Describe("BrokerHandler", func() {

	BeforeEach(func() {
		urlPath, err := routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", testAppId)
		Expect(err).ToNot(HaveOccurred())
		schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
		schedulerServer = ghttp.NewServer()

		brokerCred1 := config.BrokerCredentialsConfig{
			BrokerUsername:     "broker_username",
			BrokerUsernameHash: []byte("$2a$10$WNO1cPko4iDAT6MkhaDojeJMU8ZdNH6gt.SapsFOsC0OF4cQ9qQwu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_username")'
			BrokerPassword:     "broker_password",
			BrokerPasswordHash: []byte("$2a$10$evLviRLcIPKnWQqlBl3DJOvBZir9vJ4gdEeyoGgvnK/CGBnxIAFRu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_password")'
		}
		brokerCred2 := config.BrokerCredentialsConfig{
			BrokerUsername:     "broker_username2",
			BrokerUsernameHash: []byte("$2a$10$NK76ms9n/oeD1.IumovhIu2fiiQ/4FIVc81o4rdNS8beJMxYvhTqG"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_username2")'
			BrokerPassword:     "broker_password2",
			BrokerPasswordHash: []byte("$2a$10$HZOfLweDfjNfe2h3KItdg.26BxNU6TVKMDwhJMNPPIWpj7T2HCVbW"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_password2")'
		}
		var brokerCreds []config.BrokerCredentialsConfig
		brokerCreds = append(brokerCreds, brokerCred1, brokerCred2)
		conf = &config.Config{
			BrokerServer: config.ServerConfig{
				Port: port,
			},
			BrokerCredentials: brokerCreds,
			PlanCheck: &config.PlanCheckConfig{
				PlanDefinitions: map[string]config.PlanDefinition{
					"a-plan-id": {
						PlanCheckEnabled:  true,
						PlanUpdateable:    true,
						SchedulesCount:    1,
						ScalingRulesCount: 1,
					},
					"a-plan-id-not-updatable": {
						PlanCheckEnabled:  true,
						SchedulesCount:    1,
						ScalingRulesCount: 1,
					},
				},
			},
			CatalogPath:       "../exampleconfig/catalog-example.json",
			CatalogSchemaPath: "../schemas/catalog.schema.json",
			PolicySchemaPath:  "../policyvalidator/policy_json.schema.json",
			Scheduler: config.SchedulerConfig{
				SchedulerURL: schedulerServer.URL(),
			},
			InfoFilePath: "../exampleconfig/info-file.json",
			RateLimit: models.RateLimitConfig{
				MaxAmount:     2,
				ValidDuration: 2 * time.Second,
			},
		}
		bindingdb = &fakes.FakeBindingDB{}
		policydb = &fakes.FakePolicyDB{}
		fakecfClient = &fakes.FakeCFClient{}
		fakeCredentials = &fakes.FakeCredentials{}
		plans := []domain.ServicePlan{{
			ID:   "a-plan-id",
			Name: "standard",
		}}

		catalog = []domain.Service{{
			ID:    "a-service-id",
			Name:  "autoscaler",
			Plans: plans,
		}}

	})

	JustBeforeEach(func() {
		broker = NewBroker(lagertest.NewTestLogger(("testbroker")), conf, bindingdb, policydb, catalog, fakecfClient, fakeCredentials)
		handler = handlers.NewApiHandler(broker, lagertest.NewTestLogger(("testhandler")))
	})
	Describe("test delete binding", func() {
		var err error
		Context("when there is no binding to delete", func() {
			BeforeEach(func() {
				bindingdb.GetAppIdByBindingIdReturns("", sql.ErrNoRows)
			})
			JustBeforeEach(func() {
				err = deleteBinding(broker, testBindingId, testServiceInstanceId)
				Expect(err).To(HaveOccurred())
			})

			It("sql.ErrNoRows error occurs", func() {
				Expect(err.Error()).To(Equal("service binding does not exist"))
				Expect(policydb.DeletePolicyCallCount()).To(Equal(0))
				Expect(bindingdb.DeleteServiceBindingCallCount()).To(Equal(0))
			})
		})

		Context("when app is bound to the service instance", func() {
			BeforeEach(func() {
				bindingdb.GetAppIdByBindingIdReturns(testAppId, nil)

				_, err = http.NewRequest(http.MethodDelete, "", nil)
				Expect(err).NotTo(HaveOccurred())
				verifyScheduleIsDeletedInScheduler(testAppId)
			})
			JustBeforeEach(func() {
				err = deleteBinding(broker, testBindingId, testServiceInstanceId)
			})

			It("service binding should be deleted", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				Expect(policydb.DeletePolicyCallCount()).To(Equal(1))
				ctx, appid := policydb.DeletePolicyArgsForCall(0)
				Expect(ctx).NotTo(BeNil())
				Expect(appid).To(Equal(testAppId))
				Expect(bindingdb.DeleteServiceBindingCallCount()).To(Equal(1))
				ctx, bindingid := bindingdb.DeleteServiceBindingArgsForCall(0)
				Expect(ctx).ToNot(BeNil())
				Expect(bindingid).To(Equal(testBindingId))
			})

		})
	})
})

func verifyScheduleIsDeletedInScheduler(appId string) {
	deleteSchedulePath, err := routes.SchedulerRoutes().Get(routes.DeleteScheduleRouteName).URLPath("appId", appId)
	Expect(err).NotTo(HaveOccurred())
	schedulerServer.Reset()
	schedulerServer.AppendHandlers(ghttp.CombineHandlers(
		ghttp.VerifyRequest("DELETE", deleteSchedulePath.String()),
	))
}

func deleteBinding(h *Broker, bindingId string, serviceInstanceId string) error {
	appId, err := h.bindingdb.GetAppIdByBindingId(context.Background(), bindingId)
	if errors.Is(err, sql.ErrNoRows) {
		h.logger.Info("binding does not exist", nil, lager.Data{"instanceId": serviceInstanceId, "bindingId": bindingId})
		return ErrBindingDoesNotExist
	}
	if err != nil {
		h.logger.Error("failed to get appId by bindingId", err, lager.Data{"instanceId": serviceInstanceId, "bindingId": bindingId})
		return ErrDeleteServiceBinding
	}
	h.logger.Info("deleting policy json", lager.Data{"appId": appId})
	err = h.policydb.DeletePolicy(context.Background(), appId)
	if err != nil {
		h.logger.Error("failed to delete policy for unbinding", err, lager.Data{"appId": appId})
		return ErrDeletePolicyForUnbinding
	}

	h.logger.Info("deleting schedules", lager.Data{"appId": appId})
	err = h.schedulerUtil.DeleteSchedule(context.Background(), appId)
	if err != nil {
		h.logger.Info("failed to delete schedules for unbinding", lager.Data{"appId": appId})
		return ErrDeleteSchedulesForUnbinding
	}
	err = h.bindingdb.DeleteServiceBinding(context.Background(), bindingId)
	if err != nil {
		h.logger.Error("failed to delete binding", err, lager.Data{"bindingId": bindingId, "appId": appId})
		if errors.Is(err, db.ErrDoesNotExist) {
			return ErrBindingDoesNotExist
		}

		return ErrDeleteServiceBinding
	}

	err = h.credentials.Delete(context.Background(), appId)
	if err != nil {
		h.logger.Error("failed to delete custom metrics credential for unbinding", err, lager.Data{"appId": appId})
		return ErrCredentialNotDeleted
	}

	return nil
}
