package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsgateway/server"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const envelopesEndpoint = "/v1/envelopes"

type fakeEmitter struct {
	emittedMetrics []*models.CustomMetric
	emitErr        error
}

func (f *fakeEmitter) EmitMetric(metric *models.CustomMetric) error {
	if f.emitErr != nil {
		return f.emitErr
	}
	f.emittedMetrics = append(f.emittedMetrics, metric)
	return nil
}

var _ = Describe("Server", func() {
	var (
		srv      *server.Server
		conf     *config.Config
		logger   *lagertest.TestLogger
		fake     *fakeEmitter
		recorder *httptest.ResponseRecorder
		reqBody  []byte
		handler  http.Handler
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("metricsgateway-server-test")
		fake = &fakeEmitter{}
		conf = &config.Config{}
		srv = server.NewServer(logger, conf, fake)
		handler = srv.CreateTestRouter()
		recorder = httptest.NewRecorder()
	})

	Describe("handleEnvelopes", func() {
		When("valid metrics are sent", func() {
			BeforeEach(func() {
				metrics := []*models.CustomMetric{
					{Name: "cpu_usage", Value: 80.5, Unit: "percent", AppGUID: "app-1", InstanceIndex: 0},
					{Name: "memory", Value: 1024, Unit: "MB", AppGUID: "app-1", InstanceIndex: 1},
				}
				var err error
				reqBody, err = json.Marshal(metrics)
				Expect(err).ToNot(HaveOccurred())
			})

			It("emits all metrics and returns 200", func() {
				req := httptest.NewRequest(http.MethodPost, envelopesEndpoint, bytes.NewReader(reqBody))
				handler.ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(fake.emittedMetrics).To(HaveLen(2))
				Expect(fake.emittedMetrics[0].Name).To(Equal("cpu_usage"))
				Expect(fake.emittedMetrics[1].Name).To(Equal("memory"))
			})
		})

		When("invalid JSON is sent", func() {
			BeforeEach(func() {
				reqBody = []byte("not-valid-json")
			})

			It("returns 400", func() {
				req := httptest.NewRequest(http.MethodPost, envelopesEndpoint, bytes.NewReader(reqBody))
				handler.ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		When("empty metrics array is sent", func() {
			BeforeEach(func() {
				reqBody = []byte("[]")
			})

			It("returns 400", func() {
				req := httptest.NewRequest(http.MethodPost, envelopesEndpoint, bytes.NewReader(reqBody))
				handler.ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		When("emitter returns errors", func() {
			BeforeEach(func() {
				fake.emitErr = fmt.Errorf("syslog write failed")
				metrics := []*models.CustomMetric{
					{Name: "cpu_usage", Value: 80.5, Unit: "percent", AppGUID: "app-1", InstanceIndex: 0},
				}
				var err error
				reqBody, err = json.Marshal(metrics)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns 500", func() {
				req := httptest.NewRequest(http.MethodPost, "/v1/envelopes", bytes.NewReader(reqBody))
				handler.ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})
})
