package api

import (
	"fmt"
	"testing"

	"acceptance/config"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	cfhelpers "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var (
	cfg         config.Config
	defaultCfg  cfhelpers.Config
	context     cfhelpers.SuiteContext
	environment *cfhelpers.Environment
)

func TestAcceptance(t *testing.T) {
	RegisterFailHandler(Fail)

	cfg, defaultCfg = config.LoadConfig(t)

	context = cfhelpers.NewContext(defaultCfg)
	environment = cfhelpers.NewEnvironment(context)

	componentName := "API Suite"

	rs := []Reporter{}

	if defaultCfg.ArtifactsDirectory != "" {
		cfhelpers.EnableCFTrace(defaultCfg, componentName)
		rs = append(rs, cfhelpers.NewJUnitReporter(defaultCfg, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

var _ = BeforeSuite(func() {
	environment.Setup()

	serviceExists := cf.Cf("marketplace", "-s", cfg.ServiceName).Wait(config.DEFAULT_TIMEOUT)
	Expect(serviceExists).To(Exit(0), fmt.Sprintf("Service offering, %s, does not exist", cfg.ServiceName))
})

var _ = AfterSuite(func() {
	environment.Teardown()
})
