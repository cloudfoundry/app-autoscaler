package main_test

import (
	"fmt"
	"net/http"
	"path/filepath"

	testhelpers2 "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"golang.org/x/crypto/bcrypt"

	"os"
	"os/exec"
	"time"

	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	yaml "go.yaml.in/yaml/v4"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/testhelpers"
)

var (
	mfPath                string
	cfg                   config.Config
	healthport            int
	healthHttpClient      *http.Client
	configFile            *os.File
	httpClient            *http.Client
	req                   *http.Request
	err                   error
	body                  []byte
	grpcIngressTestServer *testhelpers.TestIngressServer
)

const (
	username = "username"
	password = "password"
)

func TestMetricsforwarder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metricsforwarder Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	mf, err := gexec.Build("code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/cmd/metricsforwarder", "-race")
	if err != nil {
		AbortSuite(fmt.Sprintf("Could not build metricsforwarder: %s", err.Error()))
	}

	dbUrl := testhelpers2.GetDbUrl()
	database, err := db.GetConnection(dbUrl)
	if err != nil {
		AbortSuite(fmt.Sprintf("DBURL not found: %s", err.Error()))
	}
	preparePolicyDb(database)
	prepareBindingDb(database)

	return []byte(mf)
}, func(pathsByte []byte) {
	mfPath = string(pathsByte)

	testCertDir := "../../../test-certs"

	grpcIngressTestServer, err = testhelpers.NewTestIngressServer(
		filepath.Join(testCertDir, "metron.crt"),
		filepath.Join(testCertDir, "metron.key"),
		filepath.Join(testCertDir, "loggregator-ca.crt"),
	)
	Expect(err).NotTo(HaveOccurred())

	err = grpcIngressTestServer.Start()
	Expect(err).NotTo(HaveOccurred())

	cfg.LoggregatorConfig.TLS.CACertFile = filepath.Join(testCertDir, "loggregator-ca.crt")
	cfg.LoggregatorConfig.TLS.CertFile = filepath.Join(testCertDir, "metron.crt")
	cfg.LoggregatorConfig.TLS.KeyFile = filepath.Join(testCertDir, "metron.key")
	cfg.LoggregatorConfig.MetronAddress = grpcIngressTestServer.GetAddr()

	cfg.RateLimit.MaxAmount = 10
	cfg.RateLimit.ValidDuration = 1 * time.Second
	cfg.Logging.Level = "debug"

	cfg.Health.BasicAuth.Username = "metricsforwarderhealthcheckuser"
	cfg.Health.BasicAuth.Password = "metricsforwarderhealthcheckpassword"
	cfg.Health.ReadinessCheckEnabled = true

	cfg.Server.Port = 10000 + GinkgoParallelProcess()
	healthport = 8000 + GinkgoParallelProcess()
	cfg.Health.ServerConfig.Port = healthport
	cfg.CacheCleanupInterval = 10 * time.Minute
	cfg.PolicyPollerInterval = 40 * time.Second
	cfg.Db = make(map[string]db.DatabaseConfig)
	dbUrl := testhelpers2.GetDbUrl()
	cfg.Db[db.PolicyDb] = db.DatabaseConfig{
		URL:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.Db[db.BindingDb] = db.DatabaseConfig{
		URL:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}

	cfg.CredHelperImpl = "default"

	configFile = writeConfig(&cfg)

	httpClient = &http.Client{}
	healthHttpClient = &http.Client{}
})

func preparePolicyDb(database *db.Database) {
	policyDB, err := sqlx.Open(database.DriverName, database.DataSourceName)
	Expect(err).NotTo(HaveOccurred())

	_, err = policyDB.Exec("DELETE from policy_json")
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed clean policy_json %s", err.Error()))
	}
	_, err = policyDB.Exec("DELETE from credentials")
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed clean credentials %s", err.Error()))
	}

	policy := `
		{
			"instance_min_count": 1,
			"instance_max_count": 5,
			"scaling_rules":[
				{
					"metric_type":"custom",
					"breach_duration_secs":600,
					"threshold":30,
					"operator":"<",
					"cool_down_secs":300,
					"adjustment":"-1"
				}
			]
		}`
	query := policyDB.Rebind("INSERT INTO policy_json(app_id, policy_json, guid) values(?, ?, ?)")
	_, err = policyDB.Exec(query, "an-app-id", policy, "1234")
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed clean credentials %s", err.Error()))
	}

	encryptedUsername, _ := bcrypt.GenerateFromPassword([]byte(username), 8)
	encryptedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), 8)

	query = policyDB.Rebind("INSERT INTO credentials(id, username, password, updated_at) values(?, ?, ?, ?)")
	_, err = policyDB.Exec(query, "an-app-id", encryptedUsername, encryptedPassword, "2011-06-18 15:36:38")
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to add credentials: %s", err.Error()))
	}

	err = policyDB.Close()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to close connection: %s", err.Error()))
	}
}

func prepareBindingDb(database *db.Database) {
	bindingDB, err := sqlx.Open(database.DriverName, database.DataSourceName)
	Expect(err).NotTo(HaveOccurred())

	_, err = bindingDB.Exec("DELETE from binding")
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed clean policy_json %s", err.Error()))
	}
	err = bindingDB.Close()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to close connection: %s", err.Error()))
	}
}

var _ = SynchronizedAfterSuite(func() {
	grpcIngressTestServer.Stop()
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func writeConfig(c *config.Config) *os.File {
	cfg, err := os.CreateTemp("", "mf")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()

	bytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = cfg.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}

type MetricsForwarderRunner struct {
	configPath string
	Session    *gexec.Session
	startCheck string
}

func NewMetricsForwarderRunner() *MetricsForwarderRunner {
	return &MetricsForwarderRunner{
		configPath: configFile.Name(),
		startCheck: "metricsforwarder.started",
	}
}

func (mf *MetricsForwarderRunner) Start() {
	// #nosec G204
	mfSession, err := gexec.Start(exec.Command(
		mfPath,
		"-c",
		mf.configPath,
	),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[mc]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[mc]\x1b[0m ", GinkgoWriter),
	)
	Expect(err).NotTo(HaveOccurred())

	if mf.startCheck != "" {
		Eventually(mfSession.Buffer, 2).Should(gbytes.Say(mf.startCheck))
	}

	mf.Session = mfSession
}

func (mf *MetricsForwarderRunner) Interrupt() {
	if mf.Session != nil {
		mf.Session.Interrupt().Wait(5 * time.Second)
	}
}
