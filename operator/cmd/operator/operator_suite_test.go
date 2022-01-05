package main_test

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v2"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/config"
)

var (
	prPath            string
	cfg               config.Config
	configFile        *os.File
	cfServer          *ghttp.Server
	healthHttpClient  *http.Client
	healthport        int
	appSummaryRegPath = regexp.MustCompile(`^/v2/apps/.*/summary$`)
)

func TestOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator Main Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	pr, err := gexec.Build("code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/cmd/operator", "-race")
	Expect(err).NotTo(HaveOccurred())

	return []byte(pr)
}, func(pathsByte []byte) {
	prPath = string(pathsByte)
	initConfig()
	healthHttpClient = &http.Client{}
	configFile = writeConfig(&cfg)
})

var _ = SynchronizedAfterSuite(func() {
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func initConfig() {
	cfServer = ghttp.NewServer()
	cfServer.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
		cf.Endpoints{
			TokenEndpoint:   cfServer.URL(),
			DopplerEndpoint: strings.Replace(cfServer.URL(), "http", "ws", 1),
		}))

	cfServer.RouteToHandler("POST", "/oauth/token", ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{}))

	appState := models.AppStatusStarted
	cfServer.RouteToHandler("GET", appSummaryRegPath, ghttp.RespondWithJSONEncoded(http.StatusOK,
		models.AppEntity{Instances: 2, State: &appState}))

	cfg.CF = cf.CFConfig{
		API:      cfServer.URL(),
		ClientID: "client-id",
		Secret:   "secret",
	}
	healthport = 8000 + GinkgoParallelProcess()
	cfg.Health.Port = healthport
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
	cfg.InstanceMetricsDB.CutoffDuration = 20 * 24 * time.Hour

	cfg.AppMetricsDB.DB = db.DatabaseConfig{
		URL:                   dbURL,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.AppMetricsDB.RefreshInterval = 12 * time.Hour
	cfg.AppMetricsDB.CutoffDuration = 20 * 24 * time.Hour

	cfg.ScalingEngineDB.DB = db.DatabaseConfig{
		URL:                   dbURL,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.ScalingEngineDB.RefreshInterval = 12 * time.Hour
	cfg.ScalingEngineDB.CutoffDuration = 20 * 24 * time.Hour

	cfg.ScalingEngine = config.ScalingEngineConfig{
		URL:          "http://localhost:8082",
		SyncInterval: 10 * time.Second,
	}

	cfg.Scheduler = config.SchedulerConfig{
		URL:          "http://localhost:8083",
		SyncInterval: 10 * time.Second,
	}

	cfg.DBLock.DB = db.DatabaseConfig{
		URL:                   os.Getenv("DBURL"),
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}

	cfg.DBLock.LockTTL = 15 * time.Second
	cfg.DBLock.LockRetryInterval = 5 * time.Second
	cfg.AppSyncer.DB = db.DatabaseConfig{
		URL:                   dbURL,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.AppSyncer.SyncInterval = 60 * time.Second
	cfg.HttpClientTimeout = 10 * time.Second

	cfg.Health.HealthCheckUsername = "operatorhealthcheckuser"
	cfg.Health.HealthCheckPassword = "operatorhealthcheckuser"
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
	database, err := db.GetConnection(os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	lockDB, err := sql.Open(database.DriverName, database.DSN)
	Expect(err).NotTo(HaveOccurred())

	_, err = lockDB.Exec("DELETE FROM operator_lock")
	Expect(err).NotTo(HaveOccurred())
}
