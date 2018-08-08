package main_test

import (
	"database/sql"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"code.cloudfoundry.org/consuladapter/consulrunner"
	"code.cloudfoundry.org/locket"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"gopkg.in/yaml.v2"

	"autoscaler/db"
	"autoscaler/pruner/config"
)

var (
	prPath       string
	cfg          config.Config
	configFile   *os.File
	consulRunner *consulrunner.ClusterRunner
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
	initConsul()
	initConfig()

	configFile = writeConfig(&cfg)
})

var _ = SynchronizedAfterSuite(func() {
	os.Remove(configFile.Name())
	if consulRunner != nil {
		consulRunner.Stop()
	}
}, func() {
	gexec.CleanupBuildArtifacts()
})

func initConsul() {
	consulRunner = consulrunner.NewClusterRunner(
		consulrunner.ClusterRunnerConfig{
			StartingPort: 9001 + GinkgoParallelNode()*consulrunner.PortOffsetLength,
			NumNodes:     1,
			Scheme:       "http",
		},
	)
	consulRunner.Start()
	consulRunner.WaitUntilReady()
}

func initConfig() {
	cfg.Logging.Level = "debug"
	dbUrl := os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}

	cfg.InstanceMetricsDb.Db = db.DatabaseConfig{
		Url:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.InstanceMetricsDb.RefreshInterval = 12 * time.Hour
	cfg.InstanceMetricsDb.CutoffDays = 20

	cfg.AppMetricsDb.Db = db.DatabaseConfig{
		Url:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.AppMetricsDb.RefreshInterval = 12 * time.Hour
	cfg.AppMetricsDb.CutoffDays = 20

	cfg.ScalingEngineDb.Db = db.DatabaseConfig{
		Url:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.ScalingEngineDb.RefreshInterval = 12 * time.Hour
	cfg.ScalingEngineDb.CutoffDays = 20

	cfg.Lock.ConsulClusterConfig = consulRunner.ConsulCluster()
	cfg.Lock.LockRetryInterval = locket.RetryInterval
	cfg.Lock.LockTTL = locket.DefaultSessionTTL

	cfg.DBLock.LockDB = db.DatabaseConfig{
		Url:                   os.Getenv("DBURL"),
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}

	cfg.DBLock.LockTTL = 15 * time.Second
	cfg.DBLock.LockRetryInterval = 5 * time.Second
	cfg.EnableDBLock = false
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
	configPath        string
	startCheck        string
	acquiredLockCheck string
	Session           *gexec.Session
}

func NewPrunerRunner() *PrunerRunner {
	return &PrunerRunner{
		configPath:        configFile.Name(),
		startCheck:        "pruner.started",
		acquiredLockCheck: "pruner.lock.acquire-lock-succeeded",
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

func (pr *PrunerRunner) ClearLockDatabase() {
	lockDB, err := sql.Open(db.PostgresDriverName, os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	_, err = lockDB.Exec("DELETE FROM pruner_lock")
	Expect(err).NotTo(HaveOccurred())
}
