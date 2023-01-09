package main_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf/mocks"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"github.com/onsi/gomega/ghttp"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"gopkg.in/yaml.v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	username    = "broker_username"
	password    = "broker_password"
	testCertDir = "../../../../../test-certs"
)

var (
	apPath           string
	cfg              config.Config
	configFile       *os.File
	apiHttpClient    *http.Client
	healthHttpClient *http.Client
	catalogBytes     string
	schedulerServer  *ghttp.Server
	brokerPort       int
	publicApiPort    int
	healthport       int
	infoBytes        string
	ccServer         *mocks.Server
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api Suite")
}

type testdata struct {
	ApPath       string
	InfoBytes    string
	CatalogBytes string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	info := testdata{}
	dbUrl := GetDbUrl()

	database, e := db.GetConnection(dbUrl)
	if e != nil {
		Fail("failed to get database URL and drivername: " + e.Error())
	}
	var err error
	info.ApPath, err = gexec.Build("code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/cmd/api", "-race")
	if err != nil {
		AbortSuite(err.Error())
	}

	apDB, err := sql.Open(database.DriverName, database.DSN)
	if err != nil {
		AbortSuite(err.Error())
	}

	_, err = apDB.Exec("DELETE FROM binding")
	if err != nil {
		AbortSuite(err.Error())
	}

	_, err = apDB.Exec("DELETE FROM service_instance")
	if err != nil {
		AbortSuite(err.Error())
	}

	err = apDB.Close()
	if err != nil {
		AbortSuite(err.Error())
	}

	info.CatalogBytes = readFile("../../exampleconfig/catalog-example.json")
	info.InfoBytes = readFile("../../exampleconfig/info-file.json")
	bytes, err := json.Marshal(info)
	if err != nil {
		AbortSuite("Failed to serialise:" + err.Error())
	}
	return bytes
}, func(testParams []byte) {
	info := &testdata{}
	err := json.Unmarshal(testParams, info)
	Expect(err).NotTo(HaveOccurred())
	ccServer = mocks.NewServer()
	ccServer.Add().Info(ccServer.URL()).OauthToken("test-token")

	apPath = info.ApPath
	catalogBytes = info.CatalogBytes
	infoBytes = info.InfoBytes
	brokerPort = 8000 + GinkgoParallelProcess()
	publicApiPort = 9000 + GinkgoParallelProcess()
	healthport = 7000 + GinkgoParallelProcess()

	cfg.BrokerServer = config.ServerConfig{
		Port: brokerPort,
		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "servicebroker.key"),
			CertFile:   filepath.Join(testCertDir, "servicebroker.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}
	cfg.PublicApiServer = config.ServerConfig{
		Port: publicApiPort,
		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "api.key"),
			CertFile:   filepath.Join(testCertDir, "api.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}
	cfg.Logging.Level = "info"
	cfg.DB = make(map[string]db.DatabaseConfig)
	dbUrl := GetDbUrl()
	cfg.DB[db.BindingDb] = db.DatabaseConfig{
		URL:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.DB[db.PolicyDb] = db.DatabaseConfig{
		URL:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}

	brokerCred1 := config.BrokerCredentialsConfig{
		BrokerUsername:     "broker_username",
		BrokerUsernameHash: nil,
		BrokerPassword:     "broker_password",
		BrokerPasswordHash: nil,
	}
	brokerCred2 := config.BrokerCredentialsConfig{
		BrokerUsername:     "broker_username2",
		BrokerUsernameHash: nil,
		BrokerPassword:     "broker_password2",
		BrokerPasswordHash: nil,
	}
	var brokerCreds []config.BrokerCredentialsConfig
	brokerCreds = append(brokerCreds, brokerCred1, brokerCred2)
	cfg.BrokerCredentials = brokerCreds

	cfg.CatalogPath = "../../exampleconfig/catalog-example.json"
	cfg.CatalogSchemaPath = "../../schemas/catalog.schema.json"
	cfg.PolicySchemaPath = "../../policyvalidator/policy_json.schema.json"

	schedulerServer = ghttp.NewServer()
	cfg.Scheduler.SchedulerURL = schedulerServer.URL()
	cfg.InfoFilePath = "../../exampleconfig/info-file.json"

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
	cfg.MetricsForwarder = config.MetricsForwarderConfig{
		MetricsForwarderUrl: "http://localhost:8088",
	}

	cfg.UseBuildInMode = false

	cfg.CF.API = ccServer.URL()
	cfg.CF.ClientID = "client-id"
	cfg.CF.Secret = "client-secret"
	cfg.CF.SkipSSLValidation = true
	cfg.Health = models.HealthConfig{
		Port:                healthport,
		HealthCheckUsername: "healthcheckuser",
		HealthCheckPassword: "healthcheckpassword",
	}
	cfg.RateLimit.MaxAmount = 10
	cfg.RateLimit.ValidDuration = 1 * time.Second

	cfg.CredHelperImpl = "default"

	configFile = writeConfig(&cfg)

	apiHttpClient = NewApiClient()

	healthHttpClient = &http.Client{}
})

var _ = SynchronizedAfterSuite(func() {
	if configFile != nil {
		_ = os.Remove(configFile.Name())
	}
	if ccServer != nil {
		ccServer.Close()
	}
}, func() {
	gexec.CleanupBuildArtifacts()
})

func writeConfig(c *config.Config) *os.File {
	cfg, err := os.CreateTemp("", "ap")
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = cfg.Close() }()

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
	// #nosec G204
	apSession, err := gexec.Start(exec.Command(
		apPath,
		"-c",
		ap.configPath,
	),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[api]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[api]\x1b[0m ", GinkgoWriter),
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

func readFile(filename string) string {
	contents, err := os.ReadFile(filename)
	FailOnError("Failed to read file:"+filename+" ", err)
	return string(contents)
}
