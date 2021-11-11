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

	"code.cloudfoundry.org/cfhttp"

	"github.com/onsi/gomega/ghttp"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	username = "broker_username"
	password = "broker_password"
)

var (
	apPath           string
	cfg              config.Config
	configFile       *os.File
	apiHttpClient    *http.Client
	brokerHttpClient *http.Client
	healthHttpClient *http.Client
	catalogBytes     []byte
	schedulerServer  *ghttp.Server
	brokerPort       int
	publicApiPort    int
	healthport       int
	infoBytes        []byte
	ccServer         *ghttp.Server
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	dbUrl := os.Getenv("DBURL")
	if dbUrl == "" {
		Fail("environment variable $DBURL is not set")
	}

	database, e := db.GetConnection(dbUrl)
	if e != nil {
		Fail("failed to get database URL and drivername: " + e.Error())
	}

	ap, err := gexec.Build("autoscaler/api/cmd/api", "-race")
	Expect(err).NotTo(HaveOccurred())

	apDB, err := sql.Open(database.DriverName, database.DSN)
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

	ccServer = ghttp.NewServer()
	ccServer.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
		cf.Endpoints{
			AuthEndpoint:  ccServer.URL(),
			TokenEndpoint: ccServer.URL(),
		}))

	ccServer.RouteToHandler("POST", "/oauth/token", ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{}))

	apPath = string(pathsByte)

	brokerPort = 8000 + GinkgoParallelProcess()
	publicApiPort = 9000 + GinkgoParallelProcess()
	healthport = 7000 + GinkgoParallelProcess()
	testCertDir := "../../../../../test-certs"

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
	cfg.DB["binding_db"] = db.DatabaseConfig{
		URL:                   os.Getenv("DBURL"),
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.DB["policy_db"] = db.DatabaseConfig{
		URL:                   os.Getenv("DBURL"),
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

	configFile = writeConfig(&cfg)
	apiClientTLSConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, "api.crt"),
		filepath.Join(testCertDir, "api.key"),
		filepath.Join(testCertDir, "autoscaler-ca.crt"))
	Expect(err).NotTo(HaveOccurred())
	apiHttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: apiClientTLSConfig,
		},
	}
	brokerClientTLSConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, "servicebroker.crt"),
		filepath.Join(testCertDir, "servicebroker.key"),
		filepath.Join(testCertDir, "autoscaler-ca.crt"))
	Expect(err).NotTo(HaveOccurred())
	brokerHttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: brokerClientTLSConfig,
		},
	}
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
	cfg, err := ioutil.TempFile("", "ap")
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

func (ap *ApiRunner) KillWithFire() {
	if ap.Session != nil {
		ap.Session.Kill().Wait(5 * time.Second)
	}
}
