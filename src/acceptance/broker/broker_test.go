package broker

import (
	"acceptance/config"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("AutoScaler Service Broker", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("autoscaler-APP")
		createApp := cf.Cf("push", appName, "--no-start", "-b", cfg.JavaBuildpackName, "-m", config.DEFAULT_MEMORY_LIMIT, "-p", config.JAVA_APP, "-d", cfg.AppsDomain).Wait(config.DEFAULT_TIMEOUT)
		Expect(createApp).To(Exit(0), "failed creating app")
	})

	AfterEach(func() {
		appReport(appName, config.DEFAULT_TIMEOUT)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(config.CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	It("performs lifecycle operations", func() {
		instanceName := generator.PrefixedRandomName("scaling-")
		createService := cf.Cf("create-service", cfg.ServiceName, "free", instanceName).Wait(config.DEFAULT_TIMEOUT)
		Expect(createService).To(Exit(0), "failed creating service")

		bindService := cf.Cf("bind-service", appName, instanceName).Wait(config.DEFAULT_TIMEOUT)
		Expect(bindService).To(Exit(0), "failed binding app to service")

		unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(config.DEFAULT_TIMEOUT)
		Expect(unbindService).To(Exit(0), "failed unbinding app to service")

		deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(config.DEFAULT_TIMEOUT)
		Expect(deleteService).To(Exit(0))
	})
})

func appReport(appName string, timeout time.Duration) {
	Eventually(cf.Cf("app", appName, "--guid"), timeout).Should(Exit())
	Eventually(cf.Cf("logs", appName, "--recent"), timeout).Should(Exit())
}
