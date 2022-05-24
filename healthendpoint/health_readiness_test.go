package healthendpoint_test

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

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

var _ healthendpoint.Pinger = &testPinger{}

type testPinger struct {
	error   error
	counter int
}

func (pinger *testPinger) Ping() error {
	pinger.counter += 1
	return pinger.error
}

var _ = Describe("Health Readiness", func() {

	var (
		t           GinkgoTInterface
		healthRoute *mux.Router
		logger      lager.Logger
		checkers    []healthendpoint.Checker
		config      models.HealthConfig
		timesetter  *time.Time
	)

	BeforeEach(func() {
		t = GinkgoT()
		logger = lager.NewLogger("healthendpoint-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		config.HealthCheckUsername = "test-user-name"
		config.HealthCheckPassword = "test-user-password"
		config.HealthCheckPasswordHash = ""
		config.HealthCheckUsernameHash = ""
		config.ReadinessCheckEnabled = true
		checkers = []healthendpoint.Checker{}
		tmsttr := time.Now()
		timesetter = &(tmsttr)
	})

	JustBeforeEach(func() {
		var err error
		healthRoute, err = healthendpoint.NewHealthRouter(config, checkers, logger, prometheus.NewRegistry(), func() time.Time { return *timesetter })
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
				var pinger *testPinger

				BeforeEach(func() {
					pinger = &testPinger{error: nil}
					checkers = []healthendpoint.Checker{healthendpoint.DbChecker("policy", pinger)}
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
				It("should cache health result", func() {
					apitest.New().
						Handler(healthRoute).
						Get("/health/readiness").
						Expect(t).
						Status(http.StatusOK).
						End()
					Expect(pinger.counter).To(Equal(1))
					tmsttr := timesetter.Add(29999 * time.Millisecond)
					timesetter = &(tmsttr)
					apitest.New().
						Handler(healthRoute).
						Get("/health/readiness").
						Expect(t).
						Status(http.StatusOK).
						End()
					Expect(pinger.counter).To(Equal(1))
				})
				It("should expire the cache entry after 30 seconds", func() {
					apitest.New().
						Handler(healthRoute).
						Get("/health/readiness").
						Expect(t).
						Status(http.StatusOK).
						End()
					Expect(pinger.counter).To(Equal(1))
					tmsttr := timesetter.Add(30 * time.Second)
					timesetter = &(tmsttr)
					apitest.New().
						Handler(healthRoute).
						Get("/health/readiness").
						Expect(t).
						Status(http.StatusOK).
						End()
					Expect(pinger.counter).To(Equal(2))
				})
			})
			Context("and a checker is supplied but readiness is disabled", func() {

				BeforeEach(func() {
					checkers = []healthendpoint.Checker{healthendpoint.DbChecker("policy", &testPinger{error: nil})}
					config.ReadinessCheckEnabled = false
				})

				It("should respond with 401 due fallthough to Prometheus health", func() {
					apitest.New().Debug().
						Handler(healthRoute).
						Get("/health/readiness").
						Expect(t).
						Status(http.StatusUnauthorized).
						End()
				})
			})
			Context("and two checkers and one is failing", func() {

				BeforeEach(func() {

					dbUpFunc := healthendpoint.DbChecker("policy", &testPinger{error: nil})
					dbDownFunc := healthendpoint.DbChecker("instance-db", &testPinger{error: errors.Errorf("DB is DOWN")})

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
			When("There are many requests at the same time", func() {
				counter := int32(0)
				numThreads := 100
				BeforeEach(func() {
					checkers = []healthendpoint.Checker{func() healthendpoint.ReadinessCheck {
						time.Sleep(10 * time.Millisecond)
						atomic.AddInt32(&counter, 1)
						return healthendpoint.ReadinessCheck{}
					}}
				})
				It("will still only call the checkers once", func() {
					wg := sync.WaitGroup{}
					wg.Add(numThreads)
					mu := sync.RWMutex{}
					mu.Lock()
					for i := numThreads; i > 0; i-- {
						go func() {
							mu.RLock()
							apitest.New().
								Handler(healthRoute).
								Get("/health/readiness").
								Expect(t).
								Status(http.StatusOK).
								End()
							wg.Done()
						}()
					}
					mu.Unlock()
					wg.Wait()
					Expect(counter).To(Equal(int32(1)))
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
