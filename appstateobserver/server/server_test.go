package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstate"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstateobserver/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/appstateobserver/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "Observer Server Suite") }

type source struct {
	after       int64
	invalidated string
	err         error
}

func (s *source) Changes(_ context.Context, after int64) (appstate.Changes, error) {
	s.after = after
	return appstate.Changes{Cursor: after + 1}, s.err
}
func (s *source) Invalidate(_ context.Context, id string) error { s.invalidated = id; return s.err }

var _ = Describe("Authenticated observer API", func() {
	var handler http.Handler
	var backing *source
	var logger = lagertest.NewTestLogger("observer-api")
	BeforeEach(func() {
		backing = &source{}
		am := &fakes.FakeXFCCAuthMiddleware{}
		am.XFCCAuthenticationMiddlewareStub = func(next http.Handler) http.Handler { return next }
		handler = server.NewServer(logger, &config.Config{}, backing).Router(am)
	})
	It("rejects unauthenticated access to both surfaces", func() {
		handler = server.NewServer(logger, &config.Config{}, backing).Router(auth.NewXfccAuthMiddleware(logger, models.XFCCAuth{ValidOrgGuid: "autoscaler-org"}))
		for _, request := range []struct{ method, path string }{{"GET", "/v1/changes?after=0"}, {"POST", "/v1/apps/app/invalidate"}} {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(request.method, request.path, nil))
			Expect(response.Code).To(Equal(http.StatusUnauthorized))
		}
	})
	It("returns changes after a validated cursor", func() {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", "/v1/changes?after=7", nil))
		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(backing.after).To(Equal(int64(7)))
		var result appstate.Changes
		Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())
		Expect(result.Cursor).To(Equal(int64(8)))
	})
	It("rejects malformed and negative cursors", func() {
		for _, cursor := range []string{"-1", "bad", ""} {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest("GET", "/v1/changes?after="+cursor, nil))
			Expect(response.Code).To(Equal(http.StatusBadRequest))
		}
	})
	It("invalidates only the requested app", func() {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("POST", "/v1/apps/app/invalidate", nil))
		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(backing.invalidated).To(Equal("app"))
	})
	It("returns a recoverable error on store failure", func() {
		backing.err = errors.New("database offline")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", "/v1/changes?after=0", nil))
		Expect(response.Code).To(Equal(http.StatusServiceUnavailable))
	})
})
