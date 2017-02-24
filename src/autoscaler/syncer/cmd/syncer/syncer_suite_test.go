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

	"autoscaler/syncer/config"
)

var (
	srPath     string
	cfg        config.Config
	configFile *os.File
)

func TestSyncer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Syncer Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	sr, err := gexec.Build("autoscaler/syncer/cmd/syncer", "-race")
	Expect(err).NotTo(HaveOccurred())

	return []byte(sr)
}, func(pathsByte []byte) {
	srPath = string(pathsByte)

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

	cfg.Db.PolicyDbUrl = os.Getenv("DBURL")
	cfg.Db.SchedulerDbUrl = os.Getenv("DBURL")

	cfg.SynchronizeInterval = 12 * time.Hour
}

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "sr")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()

	var bytes []byte
	bytes, err = yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = cfg.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}

type SyncerRunner struct {
	configPath string
	Session    *gexec.Session
}

func NewSyncerRunner() *SyncerRunner {
	return &SyncerRunner{
		configPath: configFile.Name(),
	}
}

func (sr *SyncerRunner) Start() {
	srSession, err := gexec.Start(exec.Command(
		srPath,
		"-c",
		sr.configPath,
	),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[sr]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[sr]\x1b[0m ", GinkgoWriter),
	)
	Expect(err).NotTo(HaveOccurred())

	sr.Session = srSession
}

func (sr *SyncerRunner) Interrupt() {
	if sr.Session != nil {
		sr.Session.Interrupt().Wait(5 * time.Second)
	}
}

func (sr *SyncerRunner) KillWithFire() {
	if sr.Session != nil {
		sr.Session.Kill().Wait(5 * time.Second)
	}
}
