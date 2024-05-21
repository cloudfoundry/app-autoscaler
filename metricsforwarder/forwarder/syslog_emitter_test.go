package forwarder_test

import (
	"bufio"
	"fmt"
	"net"
	"net/url"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SyslogEmitter", func() {
	var (
		listener net.Listener
		err      error
		conf     *config.Config
		emitter  forwarder.Emitter
	)

	BeforeEach(func() {
		listener, err = net.Listen("tcp", "127.0.0.1:")
		Expect(err).ToNot(HaveOccurred())

		url, err := url.Parse(fmt.Sprintf("syslog://%s", listener.Addr()))
		Expect(err).ToNot(HaveOccurred())

		conf = &config.Config{
			SyslogConfig: config.SyslogConfig{
				ServerAddress: url.Host,
			},
		}

		logger := lager.NewLogger("metricsforwarder-test")
		emitter, err = forwarder.NewSyslogEmitter(logger, conf)
	})

	AfterEach(func() {
		listener.Close()
	})

	It("should send message to syslog server", func() {
		metric := &models.CustomMetric{Name: "queuelength", Value: 12.5, Unit: "unit", InstanceIndex: 123, AppGUID: "dummy-guid"}
		emitter.EmitMetric(metric)

		conn, err := listener.Accept()
		Expect(err).ToNot(HaveOccurred())
		buf := bufio.NewReader(conn)

		actual, err := buf.ReadString('\n')
		Expect(err).ToNot(HaveOccurred())

		expected := fmt.Sprintf("Some syslog %s metric", metric.Name)
		Expect(actual).To(Equal(expected))
	})
})
