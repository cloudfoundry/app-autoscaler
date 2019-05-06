package main_test

import (
	"autoscaler/api/config"
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/models"
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/onsi/gomega/ghttp"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	yaml "gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	username = "brokeruser"
	password = "supersecretpassword"
)

var (
	apPath          string
	cfg             config.Config
	apPort          int
	configFile      *os.File
	httpClient      *http.Client
	catalogBytes    []byte
	schedulerServer *ghttp.Server
	brokerPort      int
	publicApiPort   int
	infoBytes       []byte
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	ap, err := gexec.Build("autoscaler/api/cmd/api", "-race")
	Expect(err).NotTo(HaveOccurred())

	apDB, err := sql.Open(db.PostgresDriverName, os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	_, err = apDB.Exec("DELETE FROM binding")
	Expect(err).NotTo(HaveOccurred())

	_, err = apDB.Exec("DELETE FROM service_instance")
	Expect(err).NotTo(HaveOccurred())

	err = apDB.Close()
	Expect(err).NotTo(HaveOccurred())

	catalogBytes, err = ioutil.ReadFile("../../exampleconfig/catalog-example.json")
	Expect(err).NotTo(HaveOccurred())

	infoBytes, err = ioutil.ReadFile("../../exampleconfig/info-file.json")
	Expect(err).NotTo(HaveOccurred())

	return []byte(ap)
}, func(pathsByte []byte) {
	apPath = string(pathsByte)

	brokerPort = 8000 + GinkgoParallelNode()
	publicApiPort = 9000 + GinkgoParallelNode()

	testCertDir := "../../../../../test-certs"

	cfg.BrokerServer.Port = brokerPort
	cfg.PublicApiServer.Port = publicApiPort
	cfg.Logging.Level = "info"
	cfg.DB.BindingDB = db.DatabaseConfig{
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
	cfg.BrokerUsername = username
	cfg.BrokerPassword = password
	cfg.CatalogPath = "../../exampleconfig/catalog-example.json"
	cfg.CatalogSchemaPath = "../../schemas/catalog.schema.json"
	cfg.PolicySchemaPath = "../../policyvalidator/policy_json.schema.json"

	schedulerServer = ghttp.NewServer()
	cfg.Scheduler.SchedulerURL = schedulerServer.URL()
	cfg.InfoFilePath = "../../exampleconfig/info-file.json"

	cfg.MetricsCollector = config.MetricsCollectorConfig{
		MetricsCollectorUrl: "http://localhost:8083",
		TLSClientCerts: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "metricscollector.key"),
			CertFile:   filepath.Join(testCertDir, "metricscollector.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}
	cfg.EventGenerator = config.EventGeneratorConfig{
		EventGeneratorUrl: "http://localhost:8084",
		TLSClientCerts: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
			CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}
	cfg.ScalingEngine = config.ScalingEngineConfig{
		ScalingEngineUrl: "http://localhost:8085",
		TLSClientCerts: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
			CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}

	cfg.UseBuildInMode = false

	cfg.CF.API = "http://api.bosh-lite.com"
	cfg.CF.GrantType = cf.GrantTypeClientCredentials
	cfg.CF.ClientID = "client-id"
	cfg.CF.Secret = "client-secret"
	cfg.CF.SkipSSLValidation = true

	configFile = writeConfig(&cfg)

	httpClient = &http.Client{}

})

var _ = SynchronizedAfterSuite(func() {
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "ap")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()

	bytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = cfg.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}

type ApiRunner struct {
	configPath string
	startCheck string
	Session    *gexec.Session
}

func NewApiRunner() *ApiRunner {
	return &ApiRunner{
		configPath: configFile.Name(),
		startCheck: "api.started",
	}
}

func (ap *ApiRunner) Start() {
	apSession, err := gexec.Start(exec.Command(
		apPath,
		"-c",
		ap.configPath,
	),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[mc]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[mc]\x1b[0m ", GinkgoWriter),
	)
	Expect(err).NotTo(HaveOccurred())

	if ap.startCheck != "" {
		Eventually(apSession.Buffer, 2).Should(gbytes.Say(ap.startCheck))
	}

	ap.Session = apSession
}

func (ap *ApiRunner) Interrupt() {
	if ap.Session != nil {
		ap.Session.Interrupt().Wait(5 * time.Second)
	}
}

func (ap *ApiRunner) KillWithFire() {
	if ap.Session != nil {
		ap.Session.Kill().Wait(5 * time.Second)
	}
}
