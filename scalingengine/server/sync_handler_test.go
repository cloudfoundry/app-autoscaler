package server_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/server"

	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("SyncHandler", func() {
	var (
		sychronizer *fakes.FakeActiveScheduleSychronizer
		handler     *SyncHandler
		resp        *httptest.ResponseRecorder
		req         *http.Request
		body        []byte
		err         error
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("sync-handler-test")
		sychronizer = &fakes.FakeActiveScheduleSychronizer{}
		handler = NewSyncHandler(logger, sychronizer)
		resp = httptest.NewRecorder()
	})

	Context("Sync", func() {

		BeforeEach(func() {
			req, err = http.NewRequest("PUT", "/v1/syncSchedules", bytes.NewReader(body))
			Expect(err).NotTo(HaveOccurred())
			handler.Sync(resp, req, map[string]string{})
		})

		It("returns 200 ", func() {
			Eventually(sychronizer.SyncCallCount).Should(Equal(1))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Code).To(Equal(http.StatusOK))
		})
	})

})
