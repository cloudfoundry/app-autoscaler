package main_test

import (
	"net/http"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"

	"io/ioutil"

	"os"
	"os/exec"
	"time"

	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	yaml "gopkg.in/yaml.v2"

	"autoscaler/db"
	"autoscaler/metricsforwarder/config"

	"autoscaler/metricsforwarder/testhelpers"
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
	username              string
	password              string
	grpcIngressTestServer *testhelpers.TestIngressServer
)

func TestMetricsforwarder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metricsforwarder Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	mf, err := gexec.Build("autoscaler/metricsforwarder/cmd/metricsforwarder", "-race")
	Expect(err).NotTo(HaveOccurred())

	database, err := db.GetConnection(os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	policyDB, err := sqlx.Open(database.DriverName, database.DSN)
	Expect(err).NotTo(HaveOccurred())

	_, err = policyDB.Exec("DELETE from policy_json")
	Expect(err).NotTo(HaveOccurred())

	_, err = policyDB.Exec("DELETE from credentials")
	Expect(err).NotTo(HaveOccurred())

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

	username = "username"
	password = "password"
	encryptedUsername, _ := bcrypt.GenerateFromPassword([]byte(username), 8)
	encryptedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), 8)

	query = policyDB.Rebind("INSERT INTO credentials(id, username, password, updated_at) values(?, ?, ?, ?)")
	_, err = policyDB.Exec(query, "an-app-id", encryptedUsername, encryptedPassword, "2011-06-18 15:36:38")
	Expect(err).NotTo(HaveOccurred())

	err = policyDB.Close()
	Expect(err).NotTo(HaveOccurred())

	return []byte(mf)
}, func(pathsByte []byte) {
	mfPath = string(pathsByte)

	testCertDir := "../../../../../test-certs"

	grpcIngressTestServer, _ = testhelpers.NewTestIngressServer(
		filepath.Join(testCertDir, "metron.crt"),
		filepath.Join(testCertDir, "metron.key"),
		filepath.Join(testCertDir, "loggregator-ca.crt"),
	)

	grpcIngressTestServer.Start()

	cfg.LoggregatorConfig.TLS.CACertFile = filepath.Join(testCertDir, "loggregator-ca.crt")
	cfg.LoggregatorConfig.TLS.CertFile = filepath.Join(testCertDir, "metron.crt")
	cfg.LoggregatorConfig.TLS.KeyFile = filepath.Join(testCertDir, "metron.key")
	cfg.LoggregatorConfig.MetronAddress = grpcIngressTestServer.GetAddr()

	cfg.RateLimit.MaxAmount = 10
	cfg.RateLimit.ValidDuration = 1 * time.Second
	cfg.Logging.Level = "debug"

	cfg.Health.HealthCheckUsername = "metricsforwarderhealthcheckuser"
	cfg.Health.HealthCheckPassword = "metricsforwarderhealthcheckpassword"

	cfg.Server.Port = 10000 + GinkgoParallelNode()
	healthport = 8000 + GinkgoParallelNode()
	cfg.Health.Port = healthport
	cfg.CacheCleanupInterval = 10 * time.Minute
	cfg.PolicyPollerInterval = 40 * time.Second
	cfg.Db.PolicyDb = db.DatabaseConfig{
		URL:                   os.Getenv("DBURL"),
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}

	configFile = writeConfig(&cfg)

	httpClient = &http.Client{}
	healthHttpClient = &http.Client{}
})

var _ = SynchronizedAfterSuite(func() {
	grpcIngressTestServer.Stop()
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "mf")
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

func (mf *MetricsForwarderRunner) KillWithFire() {
	if mf.Session != nil {
		mf.Session.Kill().Wait(5 * time.Second)
	}
}
