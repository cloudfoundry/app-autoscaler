package forwarder_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GatewayEmitter TLS", func() {
	var (
		emitter        forwarder.MetricForwarder
		logger         *lagertest.TestLogger
		metric         *models.CustomMetric
		testServer     *httptest.Server
		tlsCerts       models.TLSCerts
		tempDir        string
		receivedMetric bool
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "gateway-tls-test")
		Expect(err).ToNot(HaveOccurred())

		metric = &models.CustomMetric{
			Name:          "cpu",
			Value:         75.5,
			Unit:          "percent",
			InstanceIndex: 0,
			AppGUID:       "test-app",
		}
		logger = lagertest.NewTestLogger("gateway-emitter-tls-test")
		receivedMetric = false
	})

	AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	})

	Describe("TLS certificate validation", func() {
		When("server uses valid cert", func() {
			BeforeEach(func() {
				// httptest.NewTLSServer creates self-signed cert
				// Use its client for validation
				testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var metrics []*models.CustomMetric
					body, _ := io.ReadAll(r.Body)
					json.Unmarshal(body, &metrics)
					receivedMetric = len(metrics) > 0
					w.WriteHeader(http.StatusOK)
				}))

				// Write server's CA cert for client validation
				caCertPath := filepath.Join(tempDir, "ca.crt")
				certFile, err := os.Create(caCertPath)
				Expect(err).ToNot(HaveOccurred())
				// httptest server certificate is self-signed, encode as PEM
				pem.Encode(certFile, &pem.Block{
					Type:  "CERTIFICATE",
					Bytes: testServer.Certificate().Raw,
				})
				certFile.Close()

				// Client validates using server cert as CA
				tlsCerts = models.TLSCerts{
					CACertFile: caCertPath,
				}

				emitter, err = forwarder.NewGatewayEmitter(logger, testServer.URL, tlsCerts)
				Expect(err).ToNot(HaveOccurred())
			})

			It("validates server cert and sends metric successfully", func() {
				emitter.EmitMetric(metric)
				Eventually(func() bool { return receivedMetric }).Should(BeTrue())
			})
		})

		When("server cert not validated with InsecureSkipVerify", func() {
			BeforeEach(func() {
				testServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var metrics []*models.CustomMetric
					body, _ := io.ReadAll(r.Body)
					json.Unmarshal(body, &metrics)
					receivedMetric = len(metrics) > 0
					w.WriteHeader(http.StatusOK)
				}))

				// Client with NO CA cert and NO InsecureSkipVerify should fail
				tlsCerts = models.TLSCerts{}
				emitter, _ = forwarder.NewGatewayEmitter(logger, testServer.URL, tlsCerts)
			})

			It("fails cert validation without proper CA", func() {
				emitter.EmitMetric(metric)
				// Metric send should fail due to cert validation error
				Consistently(func() bool { return receivedMetric }).Should(BeFalse())
			})
		})

		When("using CF instance identity certs for mutual TLS", func() {
			BeforeEach(func() {
				// Skip if no actual CF instance certs available
				if os.Getenv("CF_INSTANCE_CERT") == "" {
					Skip("CF instance identity certs not available")
				}

				testServer = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify client cert was provided
					if len(r.TLS.PeerCertificates) > 0 {
						receivedMetric = true
					}
					w.WriteHeader(http.StatusOK)
				}))

				// Load CF instance CA for client auth
				caCertPEM, err := os.ReadFile(os.Getenv("CF_INSTANCE_CA_CERT"))
				Expect(err).ToNot(HaveOccurred())
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCertPEM)

				testServer.TLS = &tls.Config{
					ClientAuth: tls.RequireAndVerifyClientCert,
					ClientCAs:  caCertPool,
				}
				testServer.StartTLS()

				tlsCerts = models.TLSCerts{
					CertFile:   os.Getenv("CF_INSTANCE_CERT"),
					KeyFile:    os.Getenv("CF_INSTANCE_KEY"),
					CACertFile: os.Getenv("CF_INSTANCE_CA_CERT"),
				}

				emitter, err = forwarder.NewGatewayEmitter(logger, testServer.URL, tlsCerts)
				Expect(err).ToNot(HaveOccurred())
			})

			It("performs mutual TLS auth without InsecureSkipVerify", func() {
				emitter.EmitMetric(metric)
				Eventually(func() bool { return receivedMetric }).Should(BeTrue())
			})
		})
	})
})
