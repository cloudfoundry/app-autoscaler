package main_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"

	"autoscaler/pruner/config"
)

var (
	prPath         string
	cfg            config.Config
	prPort         int
	configFile     *os.File
	ccNOAAUAA      *ghttp.Server
	isTokenExpired bool
	eLock          *sync.Mutex
)

func TestPruner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pruner Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	pr, err := gexec.Build("autoscaler/pruner/cmd/pruner", "-race")
	Expect(err).NotTo(HaveOccurred())

	return []byte(pr)
}, func(pathsByte []byte) {
	prPath = string(pathsByte)

	initConfig()

	configFile = writeConfig(&cfg)
})

var _ = SynchronizedAfterSuite(func() {
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func initConfig() {
	cfg.Logging.Level = "debug"

	cfg.Db.MetricsDbUrl = os.Getenv("DBURL")

	cfg.Pruner.IntervalInHours = 12
	cfg.Pruner.CutoffDays = 20

}

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "pr")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()
	e := candiedyaml.NewEncoder(cfg)
	err = e.Encode(c)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}

type PrunerRunner struct {
	configPath string
	startCheck string
	Session    *gexec.Session
}

func NewPrunerRunner() *PrunerRunner {
	return &PrunerRunner{
		configPath: configFile.Name(),
		startCheck: "pruner.started",
	}
}

func (pr *PrunerRunner) Start() {
	prSession, err := gexec.Start(exec.Command(
		prPath,
		"-c",
		pr.configPath,
	),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[pr]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[pr]\x1b[0m ", GinkgoWriter),
	)
	Expect(err).NotTo(HaveOccurred())

	if pr.startCheck != "" {

		//Metric Pruner
		Eventually(prSession.Buffer()).Should(gbytes.Say("metrics-db-pruner-started"))

		//All pruners started
		Eventually(prSession.Buffer(), 2).Should(gbytes.Say(pr.startCheck))
	}

	pr.Session = prSession
}

func (pr *PrunerRunner) Interrupt() {
	if pr.Session != nil {
		pr.Session.Interrupt().Wait(5 * time.Second)
	}
}

func (pr *PrunerRunner) KillWithFire() {
	if pr.Session != nil {
		pr.Session.Kill().Wait(5 * time.Second)
	}
}
