package main_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"gopkg.in/yaml.v2"

	"autoscaler/pruner/config"
)

var (
	prPath     string
	cfg        config.Config
	configFile *os.File
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

	cfg.InstanceMetricsDb.DbUrl = os.Getenv("DBURL")
	cfg.InstanceMetricsDb.RefreshInterval = 12 * time.Hour
	cfg.InstanceMetricsDb.CutoffDays = 20

	cfg.AppMetricsDb.DbUrl = os.Getenv("DBURL")
	cfg.AppMetricsDb.RefreshInterval = 12 * time.Hour
	cfg.AppMetricsDb.CutoffDays = 20

	cfg.ScalingEngineDb.DbUrl = os.Getenv("DBURL")
	cfg.ScalingEngineDb.RefreshInterval = 12 * time.Hour
	cfg.ScalingEngineDb.CutoffDays = 20
}

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "pr")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()

	var bytes []byte
	bytes, err = yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = cfg.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}

type PrunerRunner struct {
	configPath string
	Session    *gexec.Session
}

func NewPrunerRunner() *PrunerRunner {
	return &PrunerRunner{
		configPath: configFile.Name(),
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
