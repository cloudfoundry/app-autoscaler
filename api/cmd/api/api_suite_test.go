package main_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf/mocks"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"github.com/onsi/gomega/ghttp"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"go.yaml.in/yaml/v4"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

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
	apPath          string
	conf            config.Config
	configFile      *os.File
	schedulerServer *ghttp.Server
	catalogBytes    string
	brokerPort      int
	publicApiPort   int
	healthport      int
	infoBytes       string
	ccServer        *mocks.Server
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

	apDB, err := sql.Open(database.DriverName, database.DataSourceName)
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

	conf.BrokerServer = helpers.ServerConfig{
		Port: brokerPort,
		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "servicebroker.key"),
			CertFile:   filepath.Join(testCertDir, "servicebroker.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}
	conf.Server = helpers.ServerConfig{
		Port: publicApiPort,
		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "api.key"),
			CertFile:   filepath.Join(testCertDir, "api.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}
	conf.Logging.Level = "info"
	conf.Db = make(map[string]db.DatabaseConfig)
	dbUrl := GetDbUrl()
	conf.Db[db.BindingDb] = db.DatabaseConfig{
		URL:                   dbUrl,
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	conf.Db[db.PolicyDb] = db.DatabaseConfig{
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
	conf.BrokerCredentials = brokerCreds

	conf.CatalogPath = "../../exampleconfig/catalog-example.json"
	conf.CatalogSchemaPath = "../../schemas/catalog.schema.json"
	conf.PolicySchemaPath = "../../policyvalidator/policy_json.schema.json"

	schedulerServer = ghttp.NewServer()
	conf.Scheduler.SchedulerURL = schedulerServer.URL()
	conf.InfoFilePath = "../../exampleconfig/info-file.json"

	conf.EventGenerator = config.EventGeneratorConfig{
		EventGeneratorUrl: "http://localhost:8084",
		TLSClientCerts: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
			CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}
	conf.ScalingEngine = config.ScalingEngineConfig{
		ScalingEngineUrl: "http://localhost:8085",
		TLSClientCerts: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
			CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}
	conf.MetricsForwarder = config.MetricsForwarderConfig{
		MetricsForwarderUrl: "http://localhost:8088",
	}

	conf.CF.API = ccServer.URL()
	conf.CF.ClientID = "client-id"
	conf.CF.Secret = "client-secret"
	conf.CF.SkipSSLValidation = true
	conf.Health = helpers.HealthConfig{
		ServerConfig: helpers.ServerConfig{
			Port: healthport,
		},
		BasicAuth: models.BasicAuth{
			Username: "healthcheckuser",
			Password: "healthcheckpassword",
		},
	}
	conf.RateLimit.MaxAmount = 10
	conf.RateLimit.ValidDuration = 1 * time.Second

	conf.CredHelperImpl = "default"

	configFile = writeConfig(&conf)

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
	conf, err := os.CreateTemp("", "ap")
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = conf.Close() }()

	bytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = conf.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return conf
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
	GinkgoHelper()
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
