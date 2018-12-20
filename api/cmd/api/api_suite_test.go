package main_test

import (
	"autoscaler/api/config"
	"autoscaler/db"
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

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
	apPath       string
	cfg          config.Config
	apPort       int
	configFile   *os.File
	httpClient   *http.Client
	catalogBytes []byte
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

	return []byte(ap)
}, func(pathsByte []byte) {
	apPath = string(pathsByte)

	apPort = 8000 + GinkgoParallelNode()
	cfg.Server.Port = apPort
	cfg.Logging.Level = "info"
	cfg.DB.BindingDB = db.DatabaseConfig{
		URL:                   os.Getenv("DBURL"),
		MaxOpenConnections:    10,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: 10 * time.Second,
	}
	cfg.BrokerUsername = username
	cfg.BrokerPassword = password
	cfg.CatalogPath = "../../exampleconfig/catalog-example.json"
	cfg.CatalogSchemaPath = "../../schemas/catalog.schema.json"

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
