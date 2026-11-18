package app_test

import (
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/app-autoscaler/acceptance/assets/app/go_app/internal/app"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Vertical rollout readiness fixture", func() {
	var next http.Handler
	BeforeEach(func() {
		GinkgoT().Setenv("VERTICAL_TEST_ENABLED", "true")
		GinkgoT().Setenv("VERTICAL_TEST_READINESS_DELAY_SECONDS", "0")
		GinkgoT().Setenv("VERTICAL_TEST_MIN_MEMORY_MB", "384")
		GinkgoT().Setenv("VERTICAL_TEST_READINESS_BLOCK", "false")
		GinkgoT().Setenv("MEMORY_LIMIT", "512m")
		next = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	})

	It("does not change ordinary test applications", func() {
		GinkgoT().Setenv("VERTICAL_TEST_ENABLED", "false")
		Expect(app.VerticalReadinessHandler(next)).To(BeIdenticalTo(next))
	})

	It("checks readiness and serves successful application requests", func() {
		handler := app.VerticalReadinessHandler(next)
		Expect(readinessResponse(handler, "/vertical/ready").Body.String()).To(MatchJSON(`{"status":"ready"}`))
		Expect(readinessResponse(handler, "/").Code).To(Equal(http.StatusOK))
	})

	DescribeTable("does not serve unready replacement containers", func(key, value string) {
		GinkgoT().Setenv(key, value)
		handler := app.VerticalReadinessHandler(next)
		Expect(readinessResponse(handler, "/vertical/ready").Code).To(Equal(http.StatusServiceUnavailable))
		Expect(readinessResponse(handler, "/").Code).To(Equal(http.StatusServiceUnavailable))
	},
		Entry("startup delay", "VERTICAL_TEST_READINESS_DELAY_SECONDS", "45"),
		Entry("undersized memory limit", "MEMORY_LIMIT", "256m"),
		Entry("unknown memory limit", "MEMORY_LIMIT", "unknown"),
		Entry("blocked startup", "VERTICAL_TEST_READINESS_BLOCK", "true"),
		Entry("invalid delay", "VERTICAL_TEST_READINESS_DELAY_SECONDS", "bad"),
	)

	It("preserves readiness of old containers after a configuration change", func() {
		old := app.VerticalReadinessHandler(next)
		GinkgoT().Setenv("VERTICAL_TEST_READINESS_BLOCK", "true")
		replacement := app.VerticalReadinessHandler(next)
		Expect(readinessResponse(old, "/").Code).To(Equal(http.StatusOK))
		Expect(readinessResponse(replacement, "/").Code).To(Equal(http.StatusServiceUnavailable))
	})
})

func readinessResponse(handler http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}
