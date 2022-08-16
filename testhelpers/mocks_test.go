package testhelpers_test

import (
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/ghttp"

	"net/http"
)

var _ = Describe("Cf cloud controller", func() {

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

	Describe("GetApp", func() {

		When("the mocks are used", func() {
			var mocks = NewMockServer()
			BeforeEach(func() {
				conf.API = mocks.URL()
				mocks.Add().GetApp("STARTED", http.StatusOK, "test_space_guid").Info(fakeLoginServer.URL())

				DeferCleanup(mocks.Close)
			})
			It("will return success", func() {
				app, err := cfc.GetApp("test-app-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(app).To(Equal(&cf.App{
					Guid:      "testing-guid-get-app",
					Name:      "mock-get-app",
					State:     "STARTED",
					CreatedAt: ParseDate("2022-07-21T13:42:30Z"),
					UpdatedAt: ParseDate("2022-07-21T14:30:17Z"),
					Relationships: cf.Relationships{
						Space: &cf.Space{
							Data: cf.SpaceData{
								Guid: "test_space_guid",
							},
						},
					},
				}))
			})
		})

	})

	Describe("GetAppProcesses", func() {

		When("the mocks are used", func() {
			var mocks = NewMockServer()
			BeforeEach(func() {
				conf.API = mocks.URL()
				mocks.Add().GetAppProcesses(27).Info(fakeLoginServer.URL())
				DeferCleanup(mocks.Close)
			})
			It("will return success", func() {
				app, err := cfc.GetAppProcesses("test-app-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(app).To(Equal(cf.Processes{{Instances: 27}}))
			})
		})

	})

	Describe("GetAppAndProcesses", func() {
		When("the mocks are used", func() {
			var mocks = NewMockServer()
			BeforeEach(func() {
				conf.API = mocks.URL()
				mocks.Add().GetAppProcesses(27).Info(fakeLoginServer.URL())
				mocks.Add().GetApp("STARTED", http.StatusOK, "test_space_guid")
				DeferCleanup(mocks.Close)
			})
			It("will return success", func() {
				app, err := cfc.GetAppAndProcesses("test-app-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(app).To(Equal(&cf.AppAndProcesses{
					App: &cf.App{
						Guid:      "testing-guid-get-app",
						Name:      "mock-get-app",
						State:     "STARTED",
						CreatedAt: ParseDate("2022-07-21T13:42:30Z"),
						UpdatedAt: ParseDate("2022-07-21T14:30:17Z"),
						Relationships: cf.Relationships{
							Space: &cf.Space{
								Data: cf.SpaceData{
									Guid: "test_space_guid",
								},
							},
						},
					},
					Processes: cf.Processes{{Instances: 27}},
				}))
			})
		})

	})

	Describe("ScaleAppWebProcess", func() {
		JustBeforeEach(func() {
			err = cfc.ScaleAppWebProcess("test-app-id", 6)
		})

		When("the mocks are used", func() {
			var mocks = NewMockServer()
			BeforeEach(func() {
				conf.API = mocks.URL()
				mocks.Add().ScaleAppWebProcess().Info(fakeLoginServer.URL())
				DeferCleanup(mocks.Close)
			})
			It("will return success", func() {
				err := cfc.ScaleAppWebProcess("r_scalingengine:503,testAppId,1:c8ec66ba", 3)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("GetRoles", func() {
		When("the mocks are used", func() {
			var mocks = NewMockServer()
			BeforeEach(func() {
				conf.API = mocks.URL()
				mocks.Add().Info(fakeLoginServer.URL()).Roles(http.StatusOK, cf.Role{Guid: "mock_guid", Type: cf.RoleSpaceDeveloper})

				DeferCleanup(mocks.Close)
			})
			It("will return success", func() {
				roles, err := cfc.GetSpaceDeveloperRoles("some_space", "some_user")
				Expect(err).NotTo(HaveOccurred())
				Expect(roles).To(Equal(cf.Roles{
					{
						Guid: "mock_guid",
						Type: cf.RoleSpaceDeveloper,
					},
				}))
				Expect(roles.HasRole(cf.RoleSpaceDeveloper)).To(BeTrue())
			})
		})
	})

	Describe("GetServiceInstance", func() {
		When("the mocks are used", func() {
			var mocks = NewMockServer()
			BeforeEach(func() {
				conf.API = mocks.URL()
				mocks.Add().Info(fakeLoginServer.URL()).ServiceInstance("A-service-plan-guid")
				DeferCleanup(mocks.Close)
			})
			It("will return success", func() {
				roles, err := cfc.GetServiceInstance("some-guid")
				Expect(err).NotTo(HaveOccurred())
				Expect(roles).To(Equal(&cf.ServiceInstance{
					Guid:          "service-instance-mock-guid",
					Type:          "managed",
					Relationships: cf.ServiceInstanceRelationships{ServicePlan: cf.ServicePlanRelation{Data: cf.ServicePlanData{Guid: "A-service-plan-guid"}}}}))
			})
		})
	})

	Describe("ServicePlan", func() {
		When("the mocks are used", func() {
			var mocks = NewMockServer()
			BeforeEach(func() {
				conf.API = mocks.URL()
				mocks.Add().Info(fakeLoginServer.URL()).ServicePlan("a-broker-plan-id")
				DeferCleanup(mocks.Close)
			})
			It("will return success", func() {
				roles, err := cfc.GetServicePlanResource("a-broker-plan-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(roles).To(Equal(&cf.ServicePlan{BrokerCatalog: cf.BrokerCatalog{Id: "a-broker-plan-id"}}))
			})
		})
	})
})
