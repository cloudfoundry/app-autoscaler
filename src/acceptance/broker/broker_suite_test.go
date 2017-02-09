package broker

import (
	"fmt"
	"testing"

	"acceptance/config"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var (
	cfg   *config.Config
	setup *workflowhelpers.ReproducibleTestSuiteSetup
)

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	cfg = config.LoadConfig(t)
	componentName := "Broker Suite"
	rs := []Reporter{}

	if cfg.GetArtifactsDirectory() != "" {
		helpers.EnableCFTrace(cfg, componentName)
		rs = append(rs, helpers.NewJUnitReporter(cfg, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

var _ = BeforeSuite(func() {

	setup = workflowhelpers.NewTestSuiteSetup(cfg)
	setup.Setup()

	serviceExists := cf.Cf("marketplace", "-s", cfg.ServiceName).Wait(cfg.DefaultTimeoutDuration())
	Expect(serviceExists).To(Exit(0), fmt.Sprintf("Service offering, %s, does not exist", cfg.ServiceName))
})

var _ = AfterSuite(func() {
	setup.Teardown()
})
