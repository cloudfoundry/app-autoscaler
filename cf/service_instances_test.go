package cf_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"

	"net/http"
)

var _ = Describe("Cf client Service Instances", func() {

	var (
		conf            *cf.Config
		cfc             *cf.Client
		fakeCC          *MockServer
		fakeLoginServer *Server
		err             error
		logger          lager.Logger
	)

	var setCfcClient = func(maxRetries int) {
		conf = &cf.Config{}
		conf.API = fakeCC.URL()
		conf.MaxRetries = maxRetries
		conf.MaxRetryWaitMs = 1
		cfc = cf.NewCFClient(conf, logger, clock.NewClock())
		err = cfc.Login()
		Expect(err).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		fakeCC = NewMockServer()
		fakeLoginServer = NewServer()
		fakeCC.Add().Info(fakeLoginServer.URL())
		fakeLoginServer.RouteToHandler("POST", cf.PathCFAuth, RespondWithJSONEncoded(http.StatusOK, cf.Tokens{
			AccessToken: "test-access-token",
			ExpiresIn:   12000,
		}))
		logger = lager.NewLogger("cf")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		setCfcClient(0)
	})

	AfterEach(func() {
		if fakeCC != nil {
			fakeCC.Close()
		}
		if fakeLoginServer != nil {
			fakeLoginServer.Close()
		}
	})

	Describe("Get Service Instances", func() {

		When("get service instances succeeds", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/service_instances/test_guid", "fields[service_plan]=name,guid"),
						RespondWith(http.StatusOK, LoadFile("service_instance.json"), http.Header{"Content-Type": []string{"application/json"}}),
					),
				)
			})

			It("returns correct struct", func() {
				serviceInstance, err := cfc.GetServiceInstance("test_guid")
				Expect(err).NotTo(HaveOccurred())
				Expect(serviceInstance).To(Equal(&cf.ServiceInstance{
					Guid: "c89b3280-fe8d-4aa0-a42e-44465bb1c61c",
					Type: "managed",
					Relationships: cf.ServiceInstanceRelationships{
						ServicePlan: cf.ServicePlan{
							Data: cf.ServicePlanData{
								Guid: "5358d122-638e-11ea-afca-bf6e756684ac",
							},
						},
					}}))
			})
		})

		When("get service instance returns a 500 code", func() {
			BeforeEach(func() {
				fakeCC.AppendHandlers(
					CombineHandlers(
						VerifyRequest("GET", "/v3/service_instances/test_guid", "fields[service_plan]=name,guid"),
						RespondWithJSONEncoded(http.StatusInternalServerError, models.CfInternalServerError),
					),
				)
			})

			It("should return correct error", func() {
				_, err := cfc.GetServiceInstance("test_guid")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`failed GetServiceInstance guid\(test_guid\):.*cf.ServiceInstance.*GET.*'UnknownError'.*`)))
			})
		})

	})

})
