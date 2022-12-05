package handlers_test

import (
	"net/http/httptest"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/handlers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var w *httptest.ResponseRecorder

type Response struct {
	Key string `json:"key"`
}

var _ = Describe("handlers", func() {
	BeforeEach(func() {
		w = httptest.NewRecorder()
	})

	Context("with valid http response writer", func() {
		Context("with valid json structure", func() {
			It("should succeed", func() {
				var structure Response
				structure.Key = "val"
				WriteJSONResponse(w, 200, structure)
				Expect(w.Result().Header.Values("Content-Length")).To(Equal([]string{"13"}))
				Expect(w.Body.String()).To(Equal("{\"key\":\"val\"}"))
				Expect(w.Code).To(Equal(200))
			})
		})
		Context("with invalid json structure", func() {
			It("should return an internal server error", func() {
				var garbage map[float64]string
				WriteJSONResponse(w, 200, garbage)
				Expect(w.Code).To(Equal(500))
			})
		})
	})

})
