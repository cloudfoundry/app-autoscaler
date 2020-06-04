package main_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"code.cloudfoundry.org/cfhttp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	yaml "gopkg.in/yaml.v2"

	"autoscaler/db"
	"autoscaler/metricsserver/config"
)

var (
	msPath           string
	cfg              config.Config
	msPort           int
	healthport       int
	configFile       *os.File
	messagesToSend   chan []byte
	isTokenExpired   bool
	eLock            *sync.Mutex
	httpClient       *http.Client
	healthHttpClient *http.Client
)

func TestMetricsServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MetricsServer Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	ms, err := gexec.Build("autoscaler/metricsserver/cmd/metricsserver", "-race")
	Expect(err).NotTo(HaveOccurred())

	database, err := db.GetConnection(os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	msDB, err := sqlx.Open(database.DriverName, database.DSN)
	Expect(err).NotTo(HaveOccurred())

	_, err = msDB.Exec("DELETE FROM appinstancemetrics")
	Expect(err).NotTo(HaveOccurred())

	_, err = msDB.Exec("DELETE from policy_json")
	Expect(err).NotTo(HaveOccurred())

	policy := `
		{
 			"instance_min_count": 1,
  			"instance_max_count": 5
		}`
	query := msDB.Rebind("INSERT INTO policy_json(app_id, policy_json, guid) values(?, ?, ?)")
	_, err = msDB.Exec(query, "an-app-id", policy, "1234")
	Expect(err).NotTo(HaveOccurred())

	err = msDB.Close()
	Expect(err).NotTo(HaveOccurred())

	return []byte(ms)
}, func(pathsByte []byte) {
	msPath = string(pathsByte)

	testCertDir := "../../../../../test-certs"
	msPort = 7000 + GinkgoParallelNode()
	healthport = 8000 + GinkgoParallelNode()
	cfg.Server.Port = msPort

	cfg.Health.Port = healthport

	cfg.Logging.Level = "info"

	cfg.DB.InstanceMetricsDB = db.DatabaseConfig{
		URL:                   os.Getenv("DBURL"),
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.DB.PolicyDB = db.DatabaseConfig{
		URL:                   os.Getenv("DBURL"),
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}

	cfg.Collector.CollectInterval = 10 * time.Second
	cfg.Collector.RefreshInterval = 30 * time.Second
	cfg.Collector.SaveInterval = 5 * time.Second
	cfg.Collector.MetricCacheSizePerApp = 100
	cfg.Collector.TLS.KeyFile = filepath.Join(testCertDir, "metricserver.key")
	cfg.Collector.TLS.CertFile = filepath.Join(testCertDir, "metricserver.crt")
	cfg.Collector.TLS.CACertFile = filepath.Join(testCertDir, "autoscaler-ca.crt")
	cfg.HttpClientTimeout = 10 * time.Second
	cfg.NodeAddrs = []string{"localhost"}
	cfg.NodeIndex = 0
	cfg.Collector.WSKeepAliveTime = 1 * time.Minute

	cfg.Collector.PersistMetrics = true
	cfg.Collector.EnvelopeProcessorCount = 5
	cfg.Collector.EnvelopeChannelSize = 1000
	cfg.Collector.MetricChannelSize = 1000

	cfg.Health.HealthCheckUsername = "metricsserverhealthcheckuser"
	cfg.Health.HealthCheckPassword = "metricsserverhealthcheckpassword"

	configFile = writeConfig(&cfg)

	tlsConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, "metricserver.crt"),
		filepath.Join(testCertDir, "metricserver.key"),
		filepath.Join(testCertDir, "autoscaler-ca.crt"))
	Expect(err).NotTo(HaveOccurred())
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	healthHttpClient = &http.Client{}
})

var _ = SynchronizedAfterSuite(func() {
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "ms")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()

	bytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = cfg.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}

type MetricsServerRunner struct {
	configPath string
	startCheck string
	Session    *gexec.Session
}

func NewMetricsServerRunner() *MetricsServerRunner {
	return &MetricsServerRunner{
		configPath: configFile.Name(),
		startCheck: "metricsserver.started",
	}
}

func (ms *MetricsServerRunner) Start() {
	msSession, err := gexec.Start(exec.Command(
		msPath,
		"-c",
		ms.configPath,
	),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[mc]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[mc]\x1b[0m ", GinkgoWriter),
	)
	Expect(err).NotTo(HaveOccurred())

	if ms.startCheck != "" {
		Eventually(msSession.Buffer, 2).Should(gbytes.Say(ms.startCheck))
	}

	ms.Session = msSession
}

func (ms *MetricsServerRunner) Interrupt() {
	if ms.Session != nil {
		ms.Session.Interrupt().Wait(5 * time.Second)
	}
}

func (ms *MetricsServerRunner) KillWithFire() {
	if ms.Session != nil {
		ms.Session.Kill().Wait(5 * time.Second)
	}
}
