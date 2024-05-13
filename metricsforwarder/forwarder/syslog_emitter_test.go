package forwarder_test

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"path/filepath"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SyslogEmitter", func() {
	var (
		listener     net.Listener
		err          error
		conf         *config.Config
		syslogConfig *config.SyslogConfig
		emitter      forwarder.Emitter
	)

	BeforeEach(func() {
		port := 10000 + GinkgoParallelProcess()
		listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		Expect(err).ToNot(HaveOccurred())

		url, err := url.Parse(fmt.Sprintf("syslog://%s", listener.Addr()))
		Expect(err).ToNot(HaveOccurred())

		syslogConfig = &config.SyslogConfig{
			ServerAddress: url.Host,
			Port:          port,
		}
	})

	JustBeforeEach(func() {
		conf = &config.Config{SyslogConfig: *syslogConfig}

		logger := lager.NewLogger("metricsforwarder-test")
		emitter, err = forwarder.NewSyslogEmitter(logger, conf)
	})

	AfterEach(func() {
		listener.Close()
	})

	Describe("EmitMetric", func() {
		Context("When tls config is provided", func() {
			BeforeEach(func() {
				testCertDir := "../../../../test-certs"
				//on server
				//filepath.Join(testCertDir, "metron.crt"),
				//filepath.Join(testCertDir, "metron.key"),
				//filepath.Join(testCertDir, "loggregator-ca.crt"),
				//
				conf.SyslogConfig.TLS = models.TLSCerts{
					KeyFile:    filepath.Join(testCertDir, "cf-app.key"),
					CertFile:   filepath.Join(testCertDir, "cf-app.crt"),
					CACertFile: filepath.Join(testCertDir, "log-cache-syslog-server-ca.crt"),
				}
			})

			XIt("should send message to syslog server", func() {
				emitter, err = forwarder.NewSyslogEmitter(lager.NewLogger("metricsforwarder-test"), conf)
				Expect(err).ToNot(HaveOccurred())

				// cast emitter to syslogEmitter to access writer
				Expect(emitter.(*forwarder.SyslogEmitter).Writer).To(BeAssignableToTypeOf(syslog.TLSWriter{}))

				metric := &models.CustomMetric{Name: "queuelength", Value: 12, Unit: "bytes", InstanceIndex: 123, AppGUID: "dummy-guid"}
				emitter.EmitMetric(metric)

				conn, err := listener.Accept()
				Expect(err).ToNot(HaveOccurred())
				buf := bufio.NewReader(conn)

				actual, err := buf.ReadString('\n')
				Expect(err).ToNot(HaveOccurred())

				expected := fmt.Sprintf(`130 <14>1 \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}\+\d{2}:\d{2} test-hostname %s \[%d\] - \[gauge@47450 name="%s" value="%.0f" unit="%s"\]`, metric.AppGUID, metric.InstanceIndex, metric.Name, metric.Value, metric.Unit)
				Expect(actual).To(MatchRegexp(expected))
			})

		})

		Context("When tls config is not provided", func() {
			It("should send message to syslog server", func() {
				metric := &models.CustomMetric{Name: "queuelength", Value: 12, Unit: "bytes", InstanceIndex: 123, AppGUID: "dummy-guid"}

				Expect(emitter.(*forwarder.SyslogEmitter).Writer).To(BeAssignableToTypeOf(&syslog.TCPWriter{}))
				emitter.EmitMetric(metric)

				conn, err := listener.Accept()
				Expect(err).ToNot(HaveOccurred())
				buf := bufio.NewReader(conn)

				actual, err := buf.ReadString('\n')
				Expect(err).ToNot(HaveOccurred())

				expected := fmt.Sprintf(`130 <14>1 \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}\+\d{2}:\d{2} test-hostname %s \[%d\] - \[gauge@47450 name="%s" value="%.0f" unit="%s"\]`, metric.AppGUID, metric.InstanceIndex, metric.Name, metric.Value, metric.Unit)
				Expect(actual).To(MatchRegexp(expected))
			})
		})
	})
})
