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

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/testhelpers"
)

var (
	mfPath                string
	cfg                   YamlValue
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
	//#nosec G101 -- test credentials, not real secrets
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

	// Load base config and override dynamic values
	cfg = loadBaseConfig()
	loggregator := cfg["loggregator"].(YamlValue)
	loggregatorTLS := loggregator["tls"].(YamlValue)
	loggregatorTLS["ca_file"] = filepath.Join(testCertDir, "loggregator-ca.crt")
	loggregatorTLS["cert_file"] = filepath.Join(testCertDir, "metron.crt")
	loggregatorTLS["key_file"] = filepath.Join(testCertDir, "metron.key")
	loggregator["metron_address"] = grpcIngressTestServer.GetAddr()

	cfg["server"].(YamlValue)["port"] = 10000 + GinkgoParallelProcess()
	healthport = 8000 + GinkgoParallelProcess()
	cfg["health"].(YamlValue)["server_config"].(YamlValue)["port"] = healthport

	dbUrl := testhelpers2.GetDbUrl()
	cfg["db"].(YamlValue)["policy_db"].(YamlValue)["url"] = dbUrl
	cfg["db"].(YamlValue)["binding_db"].(YamlValue)["url"] = dbUrl

	configFile = writeConfigValue(cfg)

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

	encryptedUsername, _ := bcrypt.GenerateFromPassword([]byte(username), bcrypt.DefaultCost)
	encryptedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

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
	f, err := os.CreateTemp("", "mf")
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()

	bytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = f.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return f
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
