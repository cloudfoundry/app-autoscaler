package forwarder_test

import (
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
		metrics := &models.CustomMetric{Name: "queuelength", Value: 12.5, Unit: "unit", InstanceIndex: 123, AppGUID: "dummy-guid"}
		emitter.EmitMetric(metrics)
		// TODO: Receive message in syslog server
	})
})
