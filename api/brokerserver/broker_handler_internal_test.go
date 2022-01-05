package brokerserver

import (
	"database/sql"
	"net/http"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/routes"

	"code.cloudfoundry.org/lager/lagertest"

	"github.com/pivotal-cf/brokerapi/domain"

	"github.com/onsi/gomega/ghttp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	bindingdb             *fakes.FakeBindingDB
	policydb              *fakes.FakePolicyDB
	fakecfClient          *fakes.FakeCFClient
	fakeCredentials       *fakes.FakeCredentials
	handler               *BrokerHandler
	conf                  *config.Config
	schedulerServer       = ghttp.NewServer()
	quotaServer           = ghttp.NewServer()
	port                  = 10000 + GinkgoParallelProcess()
	testBindingId         = "a-binding-id"
	testServiceInstanceId = "a-service-instance-id"
	testAppId             = "an-app-id"
)

var _ = Describe("BrokerHandler", func() {

	BeforeEach(func() {
		urlPath, err := routes.SchedulerRoutes().Get(routes.UpdateScheduleRouteName).URLPath("appId", testAppId)
		Expect(err).ToNot(HaveOccurred())
		schedulerServer.RouteToHandler("DELETE", urlPath.String(), ghttp.RespondWith(http.StatusOK, nil))
		schedulerServer = ghttp.NewServer()
		quotaServer = ghttp.NewServer()

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
			QuotaManagement: &config.QuotaManagementConfig{
				API: quotaServer.URL(),
			},
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
	})

	JustBeforeEach(func() {
		handler = NewBrokerHandler(lagertest.NewTestLogger("test"), conf, bindingdb, policydb, []domain.Service{{
			ID:   "a-service-id",
			Name: "autoscaler",
			Plans: []domain.ServicePlan{{
				ID:   "a-plan-id",
				Name: "standard",
			}},
		}}, fakecfClient, fakeCredentials,
		)
	})
	Describe("test delete binding", func() {
		var err error
		Context("when there is no binding to delete", func() {
			BeforeEach(func() {
				bindingdb.GetAppIdByBindingIdReturns("", sql.ErrNoRows)
			})
			JustBeforeEach(func() {
				err = deleteBinding(handler, testBindingId, testServiceInstanceId)
				Expect(err).To(HaveOccurred())
			})

			It("sql.ErrNoRows error occurs", func() {
				Expect(err.Error()).To(Equal("Service binding does not exist"))
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
				err = deleteBinding(handler, testBindingId, testServiceInstanceId)
			})

			It("service binding should be deleted", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(schedulerServer.ReceivedRequests()).To(HaveLen(1))
				Expect(policydb.DeletePolicyCallCount()).To(Equal(1))
				Expect(policydb.DeletePolicyArgsForCall(0)).To(Equal(testAppId))
				Expect(bindingdb.DeleteServiceBindingCallCount()).To(Equal(1))
				Expect(bindingdb.DeleteServiceBindingArgsForCall(0)).To(Equal(testBindingId))
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
