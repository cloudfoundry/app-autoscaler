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
		fake     *fakeEmitter
		recorder *httptest.ResponseRecorder
		handler  http.Handler
	)

	postEnvelopes := func(body []byte) {
		req := httptest.NewRequest(http.MethodPost, envelopesEndpoint, bytes.NewReader(body))
		handler.ServeHTTP(recorder, req)
	}

	marshalMetrics := func(metrics []*models.CustomMetric) []byte {
		body, err := json.Marshal(metrics)
		Expect(err).ToNot(HaveOccurred())
		return body
	}

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("metricsgateway-server-test")
		fake = &fakeEmitter{}
		conf := &config.Config{}
		srv := server.NewServer(logger, conf, fake)
		handler = srv.CreateTestRouter()
		recorder = httptest.NewRecorder()
	})

	Describe("handleEnvelopes", func() {
		When("valid metrics are sent", func() {
			It("emits all metrics and returns 200", func() {
				body := marshalMetrics([]*models.CustomMetric{
					{Name: "cpu_usage", Value: 80.5, Unit: "percent", AppGUID: "app-1", InstanceIndex: 0},
					{Name: "memory", Value: 1024, Unit: "MB", AppGUID: "app-1", InstanceIndex: 1},
				})
				postEnvelopes(body)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(fake.emittedMetrics).To(HaveLen(2))
				Expect(fake.emittedMetrics[0].Name).To(Equal("cpu_usage"))
				Expect(fake.emittedMetrics[1].Name).To(Equal("memory"))
			})
		})

		When("invalid JSON is sent", func() {
			It("returns 400", func() {
				postEnvelopes([]byte("not-valid-json"))
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		When("empty metrics array is sent", func() {
			It("returns 400", func() {
				postEnvelopes([]byte("[]"))
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})

		When("emitter returns errors", func() {
			BeforeEach(func() {
				fake.emitErr = fmt.Errorf("syslog write failed")
			})

			It("returns 500", func() {
				body := marshalMetrics([]*models.CustomMetric{
					{Name: "cpu_usage", Value: 80.5, Unit: "percent", AppGUID: "app-1", InstanceIndex: 0},
				})
				postEnvelopes(body)
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})
})
