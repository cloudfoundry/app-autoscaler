package forwarder_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("GatewayEmitter", func() {
	var (
		emitter    forwarder.MetricForwarder
		logger     *lagertest.TestLogger
		buffer     *gbytes.Buffer
		metric     *models.CustomMetric
		testServer *httptest.Server
	)

	BeforeEach(func() {
		metric = &models.CustomMetric{
			Name:          "queuelength",
			Value:         12,
			Unit:          "bytes",
			InstanceIndex: 123,
			AppGUID:       "dummy-guid",
		}
		logger = lagertest.NewTestLogger("gateway-emitter-test")
		buffer = logger.Buffer()
	})

	AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	When("gateway returns 200 OK", func() {
		var receivedMetrics []*models.CustomMetric

		BeforeEach(func() {
			receivedMetrics = nil
			testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal(http.MethodPost))
				Expect(r.URL.Path).To(Equal("/v1/envelopes"))
				Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

				body, err := io.ReadAll(r.Body)
				Expect(err).ToNot(HaveOccurred())

				err = json.Unmarshal(body, &receivedMetrics)
				Expect(err).ToNot(HaveOccurred())

				w.WriteHeader(http.StatusOK)
			}))

			var err error
			emitter, err = forwarder.NewGatewayEmitter(logger, testServer.URL, models.TLSCerts{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("sends the metric to the gateway", func() {
			emitter.EmitMetric(metric)
			Expect(receivedMetrics).To(HaveLen(1))
			Expect(receivedMetrics[0].Name).To(Equal("queuelength"))
			Expect(receivedMetrics[0].Value).To(Equal(float64(12)))
			Expect(receivedMetrics[0].Unit).To(Equal("bytes"))
			Expect(receivedMetrics[0].AppGUID).To(Equal("dummy-guid"))
			Expect(receivedMetrics[0].InstanceIndex).To(Equal(uint32(123)))
		})
	})

	When("gateway returns an error status", func() {
		BeforeEach(func() {
			testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))

			var err error
			emitter, err = forwarder.NewGatewayEmitter(logger, testServer.URL, models.TLSCerts{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("logs the error", func() {
			emitter.EmitMetric(metric)
			Eventually(buffer).Should(gbytes.Say("gateway-returned-error"))
		})
	})

	When("gateway is unreachable", func() {
		BeforeEach(func() {
			var err error
			emitter, err = forwarder.NewGatewayEmitter(logger, "http://127.0.0.1:1", models.TLSCerts{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("logs the error", func() {
			emitter.EmitMetric(metric)
			Eventually(buffer).Should(gbytes.Say("failed-to-send-metric-to-gateway"))
		})
	})
})
