package main_test

import (
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf/mocks"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"

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
	testCertDir = "../../../test-certs"
)

var (
	apPath          string
	conf            YamlValue
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

	schedulerServer = ghttp.NewServer()

	// Load base config from YAML file and override dynamic values
	conf = loadBaseConfig()
	conf["broker_server"].(map[string]any)["port"] = brokerPort
	conf["public_api_server"].(map[string]any)["port"] = publicApiPort
	conf["health"].(map[string]any)["server_config"].(map[string]any)["port"] = healthport
	conf["db"].(map[string]any)["binding_db"].(map[string]any)["url"] = GetDbUrl()
	conf["db"].(map[string]any)["policy_db"].(map[string]any)["url"] = GetDbUrl()
	conf["scheduler"].(map[string]any)["scheduler_url"] = schedulerServer.URL()
	conf["cf"].(map[string]any)["api"] = ccServer.URL()

	configFile = writeConfigValue(conf)
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

type YamlValue = map[string]any

func loadBaseConfig() YamlValue {
	baseBytes, err := os.ReadFile("testdata/base_config.yml")
	Expect(err).NotTo(HaveOccurred())
	var base YamlValue
	err = yaml.Unmarshal(baseBytes, &base)
	Expect(err).NotTo(HaveOccurred())
	return base
}

func copyConfig(src YamlValue) YamlValue {
	bytes, err := yaml.Marshal(src)
	Expect(err).NotTo(HaveOccurred())
	var dst YamlValue
	err = yaml.Unmarshal(bytes, &dst)
	Expect(err).NotTo(HaveOccurred())
	return dst
}

func writeConfigValue(c YamlValue) *os.File {
	f, err := os.CreateTemp("", "ap")
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = f.Close() }()

	bytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = f.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return f
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
