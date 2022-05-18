package healthendpoint_test

import (
	"net/http"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/healthendpoint"
	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/steinfletcher/apitest"
)

var _ = Describe("Health Readiness", func() {

	var (
		t           GinkgoTInterface
		healthRoute *mux.Router
		logger      lager.Logger
		checkers    []healthendpoint.Checker
		username    string
		password    string
	)

	BeforeEach(func() {
		t = GinkgoT()
		logger = lager.NewLogger("healthendpoint-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		username = "test-user-name"
		password = "test-user-password"
		checkers = []healthendpoint.Checker{}
	})

	JustBeforeEach(func() {
		var err error
		healthRoute, err = healthendpoint.NewHealthRouter(checkers, logger, prometheus.NewRegistry(), username, password, "", "")
		Expect(err).ShouldNot(HaveOccurred())

	})

	Context("Readiness endpoint is called without basic auth", func() {
		It("should have json response", func() {
			apitest.New().
				Handler(healthRoute).
				Get("/health/readiness").
				Expect(t).
				Status(http.StatusOK).
				Header("Content-Type", "application/json").
				Body(`{"status" : "OK", "checks" : [] }`).
				End()
		})
	})

	Context("Readiness endpoint is called without basic auth", func() {
		BeforeEach(func() {
			checkers = []healthendpoint.Checker{func() healthendpoint.ReadinessCheck {
				return healthendpoint.ReadinessCheck{Name: "policy", Type: "database", Status: "OK"}
			}}
		})
		It("should have database check passing", func() {
			apitest.New().
				Handler(healthRoute).
				Get("/health/readiness").
				Expect(t).
				Status(http.StatusOK).
				Header("Content-Type", "application/json").
				Body(`{ 
	"status" : "OK",
	"checks" : [ {"name": "policy", "type": "database", "status": "OK" } ]
}`).
				End()
		})
	})

	Context("Prometheus Health endpoint is called without basic auth", func() {
		BeforeEach(func() {
			username = ""
			password = ""
		})
		It("/anything should respond OK", func() {
			apitest.New().
				Handler(healthRoute).
				Get("/anything").
				Expect(t).
				Status(http.StatusOK).
				Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8").
				End()
		})
		It("/health/readiness should response OK", func() {
			apitest.New().
				Handler(healthRoute).
				Get("/health/readiness").
				Expect(t).
				Status(http.StatusOK).
				Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8").
				End()
		})
	})

	Context("Prometheus Health endpoint is called", func() {
		It("should require basic auth", func() {
			apitest.New().
				Handler(healthRoute).
				Get("/health").
				Expect(t).
				Status(http.StatusUnauthorized).
				End()
		})
	})

	Context("Health endpoint default response", func() {
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
