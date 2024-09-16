package forwarder_test

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

type fakeWriter struct {
	returnWriteError  bool
	receivedEnvelopes []*loggregator_v2.Envelope
}

func (f *fakeWriter) ReceivedEnvelope() []*loggregator_v2.Envelope {
	return f.receivedEnvelopes
}

func (f *fakeWriter) SetReturnError(returnError bool) {
	f.returnWriteError = returnError
}

func (f *fakeWriter) Close() error {
	return nil
}

func (f *fakeWriter) Write(envelope *loggregator_v2.Envelope) error {
	if f.returnWriteError {
		return fmt.Errorf("error when writing")
	}
	f.receivedEnvelopes = append(f.receivedEnvelopes, envelope)
	return nil
}

var _ = Describe("SyslogEmitter", func() {
	var (
		listener net.Listener
		err      error
		port     int
		conf     *config.Config
		tlsCerts models.TLSCerts
		logger   *lagertest.TestLogger
		emitter  forwarder.MetricForwarder
		buffer   *gbytes.Buffer
	)

	BeforeEach(func() {
		port = 10000 + GinkgoParallelProcess()
		listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		Expect(err).ToNot(HaveOccurred())
		tlsCerts = models.TLSCerts{}
		Expect(err).ToNot(HaveOccurred())

	})

	JustBeforeEach(func() {
		host, port, err := net.SplitHostPort(listener.Addr().String())
		Expect(err).ToNot(HaveOccurred())

		portNumber, err := strconv.Atoi(port)
		Expect(err).ToNot(HaveOccurred())

		conf = &config.Config{
			SyslogConfig: config.SyslogConfig{
				ServerAddress: host,
				Port:          portNumber,
				TLS:           tlsCerts,
			},
		}

	})

	AfterEach(func() {
		emitter = nil
		err = listener.Close()
		Expect(err).ToNot(HaveOccurred())
		// Wait for the listener to be closed
		// Otherwise, the next test may fail with "address already in use"
		// because the listener may not be closed yet
		Eventually(func() error {
			_, err := listener.Accept()
			return err
		}).Should(HaveOccurred())
	})

	JustBeforeEach(func() {
		logger = lagertest.NewTestLogger("metricsforwarder-test")
		buffer = logger.Buffer()
		emitter, err = forwarder.NewSyslogEmitter(logger, conf)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewSyslogEmitter", func() {
		When("tls config is provided", func() {
			BeforeEach(func() {
				testCertDir := "../../../../test-certs"
				tlsCerts = models.TLSCerts{
					KeyFile:    filepath.Join(testCertDir, "cf-app.key"),
					CertFile:   filepath.Join(testCertDir, "cf-app.crt"),
					CACertFile: filepath.Join(testCertDir, "log-cache-syslog-server-ca.crt"),
				}
			})

			It("Writer should be TLS", func() {
				Expect(emitter.(*forwarder.SyslogEmitter).GetWriter()).To(BeAssignableToTypeOf(&syslog.RetryWriter{}))
			})
		})

		When("tls config is not provided", func() {
			JustBeforeEach(func() {
				conf.SyslogConfig.TLS = models.TLSCerts{}
			})

			It("Writer should be TCP", func() {
				Expect(emitter.(*forwarder.SyslogEmitter).GetWriter()).To(BeAssignableToTypeOf(&syslog.RetryWriter{}))
			})
		})
	})

	Describe("EmitMetric", func() {
		var (
			returnWriteError bool
			writer           *fakeWriter
			metric           *models.CustomMetric
		)

		BeforeEach(func() {
			metric = &models.CustomMetric{Name: "queuelength", Value: 12, Unit: "bytes", InstanceIndex: 123, AppGUID: "dummy-guid"}
		})

		JustBeforeEach(func() {
			writer = &fakeWriter{}
			writer.SetReturnError(returnWriteError)

			emitter.(*forwarder.SyslogEmitter).SetWriter(writer)
		})

		When("writer syslog returns error", func() {
			BeforeEach(func() {
				returnWriteError = true
			})

			It("should log it out", func() {
				emitter.EmitMetric(metric)
				Eventually(buffer).Should(gbytes.Say("failed-to-write-metric-to-syslog"))
			})
		})

		When("writer syslog does not return error", func() {
			BeforeEach(func() {
				returnWriteError = false
			})

			It("should send message to syslog server", func() {
				emitter.EmitMetric(metric)
				Eventually(writer.ReceivedEnvelope()).Should(HaveLen(1))
				receivedMetric := writer.ReceivedEnvelope()[0]
				expectedEnvelope := forwarder.EnvelopeForMetric(metric)
				Expect(receivedMetric.Message).To(Equal(expectedEnvelope.Message))
				Expect(receivedMetric.SourceId).To(Equal(expectedEnvelope.SourceId))
				Expect(receivedMetric.InstanceId).To(Equal(expectedEnvelope.InstanceId))

			})
		})
	})
})
