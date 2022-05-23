package healthendpoint_test

import (
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"github.com/pkg/errors"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/steinfletcher/apitest"
)

type testPinger struct {
	error error
}

func (pinger testPinger) Ping() error {
	return pinger.error
}

var _ = Describe("Health Readiness", func() {

	var (
		t           GinkgoTInterface
		healthRoute *mux.Router
		logger      lager.Logger
		checkers    []healthendpoint.Checker
		config      models.HealthConfig
	)

	BeforeEach(func() {
		t = GinkgoT()
		logger = lager.NewLogger("healthendpoint-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		config.HealthCheckUsername = "test-user-name"
		config.HealthCheckPassword = "test-user-password"
		config.HealthCheckUsernameHash = ""
		config.HealthCheckPasswordHash = ""
		config.ReadinessCheckEnabled = true
		checkers = []healthendpoint.Checker{}
	})

	JustBeforeEach(func() {
		var err error
		healthRoute, err = healthendpoint.NewHealthRouter(config, checkers, logger, prometheus.NewRegistry())
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("Authentication parameter checks", func() {
		When("username and password are defined", func() {
			BeforeEach(func() {
				config.HealthCheckUsername = "username"
				config.HealthCheckPassword = "password"
				config.HealthCheckUsernameHash = ""
				config.HealthCheckPasswordHash = ""
			})
			When("Prometheus Health endpoint is called", func() {
				It("should require basic auth", func() {
					apitest.New().
						Handler(healthRoute).
						Get("/health").
						Expect(t).
						Status(http.StatusUnauthorized).
						End()
				})
			})
		})
		When("username_hash and password_hash are defined", func() {
			BeforeEach(func() {
				config.HealthCheckUsername = ""
				config.HealthCheckPassword = ""
				config.HealthCheckUsernameHash = "username_hash"
				config.HealthCheckPasswordHash = "username_hash"
			})
			When("Prometheus Health endpoint is called without basic auth", func() {
				It("should require basic auth", func() {
					apitest.New().
						Handler(healthRoute).
						Get("/health").
						Expect(t).
						Status(http.StatusUnauthorized).
						End()
				})
			})
		})
	})

	Context("without basic auth configured", func() {
		BeforeEach(func() {
			config.HealthCheckUsername = ""
			config.HealthCheckPassword = ""
		})
		When("Prometheus Health endpoint is called", func() {
			It("should respond OK", func() {
				apitest.New().
					Handler(healthRoute).
					Get("/anything").
					Expect(t).
					Status(http.StatusOK).
					Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8").
					End()
			})
		})
		When("/health/readiness endpoint is called", func() {
			It("should response OK", func() {
				apitest.New().
					Handler(healthRoute).
					Get("/health/readiness").
					Expect(t).
					Status(http.StatusOK).
					Header("Content-Type", "application/json").
					Body(`{"overall_status" : "UP", "checks" : [] }`).
					End()
			})
		})
		When("readiness is disabled", func() {
			BeforeEach(func() { config.ReadinessCheckEnabled = false })

			It("should respond Prometheus Health endpoint", func() {
				apitest.New().
					Handler(healthRoute).
					Get("/health/readiness").
					Expect(t).
					Status(http.StatusOK).
					Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8").
					End()
			})
		})
	})

	Context("with basic auth configured", func() {
		When("Readiness endpoint is called without basic auth", func() {
			Context("and without checkers", func() {
				It("should have json response", func() {
					apitest.New().
						Handler(healthRoute).
						Get("/health/readiness").
						Expect(t).
						Status(http.StatusOK).
						Header("Content-Type", "application/json").
						Body(`{"overall_status" : "UP", "checks" : [] }`).
						End()
				})
			})
			Context("and a checker is passing", func() {

				BeforeEach(func() {
					checkers = []healthendpoint.Checker{healthendpoint.DbChecker("policy", testPinger{nil})}
				})

				It("should have database check passing", func() {
					apitest.New().
						Handler(healthRoute).
						Get("/health/readiness").
						Expect(t).
						Status(http.StatusOK).
						Header("Content-Type", "application/json").
						Body(`{ 
	"overall_status" : "UP",
	"checks" : [ {"name": "policy", "type": "database", "status": "UP" } ]
}`).
						End()
				})
			})
			Context("and a checker is supplied but readiness is disabled", func() {

				BeforeEach(func() {
					checkers = []healthendpoint.Checker{healthendpoint.DbChecker("policy", testPinger{nil})}
					config.ReadinessCheckEnabled = false
				})

				It("should respond with 401 due fallthough to Prometheus health", func() {
					apitest.New().
						Handler(healthRoute).
						Get("/health/readiness").
						Expect(t).
						Status(http.StatusUnauthorized).
						End()
				})
			})
			Context("and two checkers and one is failing", func() {

				BeforeEach(func() {

					dbUpFunc := healthendpoint.DbChecker("policy", testPinger{nil})
					dbDownFunc := healthendpoint.DbChecker("instance-db", testPinger{errors.Errorf("DB is DOWN")})

					serverDownFunc := func() healthendpoint.ReadinessCheck {
						return healthendpoint.ReadinessCheck{Name: "instance", Type: "server", Status: "DOWN"}
					}
					checkers = []healthendpoint.Checker{dbUpFunc, dbDownFunc, serverDownFunc}
				})
				It("should have overall status down", func() {
					apitest.New().
						Handler(healthRoute).
						Get("/health/readiness").
						Expect(t).
						Status(http.StatusOK).
						Header("Content-Type", "application/json").
						Body(`{ 
							"overall_status" : "DOWN",
							"checks" : [ 
									{"name": "policy", "type": "database", "status": "UP" },
									{"name": "instance-db", "type": "database", "status": "DOWN" },
									{"name": "instance", "type": "server", "status": "DOWN" }
						]}`).
						End()
				})
			})
		})
		When("Prometheus Health endpoint is called", func() {
			It("should require basic auth", func() {
				apitest.New().
					Handler(healthRoute).
					Get("/health").
					Expect(t).
					Status(http.StatusUnauthorized).
					End()
			})
		})
		When("Default endpoint is called", func() {
			It("should require basic auth", func() {
				apitest.New().
					Handler(healthRoute).
					Get("/any").
					Expect(t).
					Status(http.StatusUnauthorized).
					End()
			})
		})

	})

})
