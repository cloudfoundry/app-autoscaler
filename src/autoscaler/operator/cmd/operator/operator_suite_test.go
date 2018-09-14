package main_test

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/consuladapter/consulrunner"
	"code.cloudfoundry.org/locket"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"gopkg.in/yaml.v2"

	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/models"
	"autoscaler/operator/config"
)

var (
	prPath       string
	cfg          config.Config
	configFile   *os.File
	consulRunner *consulrunner.ClusterRunner
	cfServer     *ghttp.Server
	appId        string
)

func TestOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator Main Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	pr, err := gexec.Build("autoscaler/operator/cmd/operator", "-race")
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

	cfServer = ghttp.NewServer()
	cfServer.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
		cf.Endpoints{
			AuthEndpoint:    cfServer.URL(),
			DopplerEndpoint: strings.Replace(cfServer.URL(), "http", "ws", 1),
		}))

	cfServer.RouteToHandler("POST", "/oauth/token", ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{}))

	appState := models.AppStatusStarted
	cfServer.RouteToHandler("GET", "/v2/apps/an-app-id/summary", ghttp.RespondWithJSONEncoded(http.StatusOK,
		models.AppEntity{Instances: 2, State: &appState}))

	cfg.CF = cf.CFConfig{
		API:       cfServer.URL(),
		GrantType: cf.GrantTypePassword,
		Username:  "admin",
		Password:  "admin",
	}

	cfg.Logging.Level = "debug"
	dbURL := os.Getenv("DBURL")
	if dbURL == "" {
		Fail("environment variable $DBURL is not set")
	}

	cfg.InstanceMetricsDB.DB = db.DatabaseConfig{
		URL:                   dbURL,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.InstanceMetricsDB.RefreshInterval = 12 * time.Hour
	cfg.InstanceMetricsDB.CutoffDays = 20

	cfg.AppMetricsDB.DB = db.DatabaseConfig{
		URL:                   dbURL,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.AppMetricsDB.RefreshInterval = 12 * time.Hour
	cfg.AppMetricsDB.CutoffDays = 20

	cfg.ScalingEngineDB.DB = db.DatabaseConfig{
		URL:                   dbURL,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.ScalingEngineDB.RefreshInterval = 12 * time.Hour
	cfg.ScalingEngineDB.CutoffDays = 20

	cfg.ScalingEngine = config.ScalingEngineConfig{
		URL:          "http://localhost:8082",
		SyncInterval: 10 * time.Second,
	}

	cfg.Scheduler = config.SchedulerConfig{
		URL:          "http://localhost:8083",
		SyncInterval: 10 * time.Second,
	}

	cfg.Lock.ConsulClusterConfig = consulRunner.ConsulCluster()
	cfg.Lock.LockRetryInterval = locket.RetryInterval
	cfg.Lock.LockTTL = locket.DefaultSessionTTL

	cfg.DBLock.DB = db.DatabaseConfig{
		URL:                   os.Getenv("DBURL"),
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}

	cfg.DBLock.LockTTL = 15 * time.Second
	cfg.DBLock.LockRetryInterval = 5 * time.Second
	cfg.EnableDBLock = false
	cfg.AppSyncer.DB = db.DatabaseConfig{
		URL:                   dbURL,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.AppSyncer.SyncInterval = 60 * time.Second

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

type OperatorRunner struct {
	configPath        string
	startCheck        string
	acquiredLockCheck string
	Session           *gexec.Session
}

func NewOperatorRunner() *OperatorRunner {
	return &OperatorRunner{
		configPath:        configFile.Name(),
		startCheck:        "operator.started",
		acquiredLockCheck: "operator.lock.acquire-lock-succeeded",
	}
}

func (pr *OperatorRunner) Start() {
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

func (pr *OperatorRunner) Interrupt() {
	if pr.Session != nil {
		pr.Session.Interrupt().Wait(5 * time.Second)
	}
}

func (pr *OperatorRunner) KillWithFire() {
	if pr.Session != nil {
		pr.Session.Kill().Wait(5 * time.Second)
	}
}

func (pr *OperatorRunner) ClearLockDatabase() {
	lockDB, err := sql.Open(db.PostgresDriverName, os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	_, err = lockDB.Exec("DELETE FROM operator_lock")
	Expect(err).NotTo(HaveOccurred())
}
