package main_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf/mocks"

	models2 "code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	_ "github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"go.yaml.in/yaml/v4"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/config"
)

var (
	prPath     string
	conf       config.Config
	configFile *os.File
	cfServer   *mocks.Server
	healthport int
)

func TestOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator Main Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	pr, err := gexec.Build("code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/cmd/operator", "-race")
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to build cmd/operator: %s", err.Error()))
	}

	return []byte(pr)
}, func(pathsByte []byte) {
	prPath = string(pathsByte)
	initConfig()
	configFile = writeConfig(&conf)
})

var _ = SynchronizedAfterSuite(func() {
	_ = os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func initConfig() {
	cfServer = mocks.NewServer()
	cfServer.Add().
		Info(cfServer.URL()).
		GetApp(models2.AppStatusStarted, http.StatusOK, "test_space_guid").
		GetAppProcesses(2).
		OauthToken("some-test-token")

	conf.CF = cf.Config{
		API:      cfServer.URL(),
		ClientID: "client-id",
		Secret:   "secret",
	}
	healthport = 8000 + GinkgoParallelProcess()
	conf.Health.ServerConfig.Port = healthport
	conf.Logging.Level = "debug"
	dbUrl := testhelpers.GetDbUrl()
	conf.Db = make(map[string]db.DatabaseConfig)

	conf.Db[db.AppMetricsDb] = db.DatabaseConfig{
		URL:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	conf.AppMetricsDb.RefreshInterval = 12 * time.Hour
	conf.AppMetricsDb.CutoffDuration = 20 * 24 * time.Hour

	conf.Db[db.ScalingEngineDb] = db.DatabaseConfig{
		URL:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	conf.ScalingEngineDb.RefreshInterval = 12 * time.Hour
	conf.ScalingEngineDb.CutoffDuration = 20 * 24 * time.Hour

	conf.ScalingEngine = config.ScalingEngineConfig{
		URL:          "http://localhost:8082",
		SyncInterval: 10 * time.Second,
	}

	conf.Scheduler = config.SchedulerConfig{
		URL:          "http://localhost:8083",
		SyncInterval: 10 * time.Second,
	}

	conf.Db[db.LockDb] = db.DatabaseConfig{
		URL:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}

	conf.DBLock.LockTTL = 15 * time.Second
	conf.DBLock.LockRetryInterval = 5 * time.Second
	conf.Db[db.PolicyDb] = db.DatabaseConfig{
		URL:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	conf.AppSyncer.SyncInterval = 60 * time.Second
	conf.HttpClientTimeout = 10 * time.Second

	conf.Health.BasicAuth.Username = "operatorhealthcheckuser"
	conf.Health.BasicAuth.Password = "operatorhealthcheckuser"
}

func writeConfig(c *config.Config) *os.File {
	conf, err := os.CreateTemp("", "pr")
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = conf.Close() }()

	var bytes []byte
	bytes, err = yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = conf.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return conf
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
	// #nosec G204
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
	dbUrl := testhelpers.GetDbUrl()
	database, err := db.GetConnection(dbUrl)
	Expect(err).NotTo(HaveOccurred())

	lockDB, err := sql.Open(database.DriverName, database.DataSourceName)
	Expect(err).NotTo(HaveOccurred())

	_, err = lockDB.Exec("DELETE FROM operator_lock")
	Expect(err).NotTo(HaveOccurred())
}
