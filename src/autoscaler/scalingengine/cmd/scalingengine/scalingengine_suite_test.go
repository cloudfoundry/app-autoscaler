package main_test

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/models"
	"autoscaler/scalingengine/config"
	"path/filepath"

	"code.cloudfoundry.org/cfhttp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"gopkg.in/yaml.v2"

	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/consuladapter/consulrunner"
)

func TestScalingengine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scalingengine Suite")
}

var (
	enginePath   string
	conf         config.Config
	port         int
	configFile   *os.File
	ccUAA        *ghttp.Server
	appId        string
	httpClient   *http.Client
	consulRunner *consulrunner.ClusterRunner
)

var _ = SynchronizedBeforeSuite(
	func() []byte {
		compiledPath, err := gexec.Build("autoscaler/scalingengine/cmd/scalingengine", "-race")
		Expect(err).NotTo(HaveOccurred())
		return []byte(compiledPath)
	},
	func(pathBytes []byte) {
		consulRunner = consulrunner.NewClusterRunner(
			consulrunner.ClusterRunnerConfig{
				StartingPort: 9001 + GinkgoParallelNode()*consulrunner.PortOffsetLength,
				NumNodes:     1,
				Scheme:       "http",
			},
		)
		consulRunner.Start()
		consulRunner.WaitUntilReady()

		enginePath = string(pathBytes)

		ccUAA = ghttp.NewServer()
		ccUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
			cf.Endpoints{
				AuthEndpoint:    ccUAA.URL(),
				DopplerEndpoint: strings.Replace(ccUAA.URL(), "http", "ws", 1),
			}))

		ccUAA.RouteToHandler("POST", "/oauth/token", ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{}))

		appId = fmt.Sprintf("%s-%d", "app-id", GinkgoParallelNode())
		appState := models.AppStatusStarted
		ccUAA.RouteToHandler("GET", "/v2/apps/"+appId+"/summary", ghttp.RespondWithJSONEncoded(http.StatusOK,
			models.AppEntity{Instances: 2, State: &appState}))
		ccUAA.RouteToHandler("PUT", "/v2/apps/"+appId, ghttp.RespondWith(http.StatusCreated, ""))

		conf.Cf = cf.CfConfig{
			Api:       ccUAA.URL(),
			GrantType: cf.GrantTypePassword,
			Username:  "admin",
			Password:  "admin",
		}

		port = 7000 + GinkgoParallelNode()
		testCertDir := "../../../../../test-certs"
		conf.Server.Port = port
		conf.Server.TLS.KeyFile = filepath.Join(testCertDir, "scalingengine.key")
		conf.Server.TLS.CertFile = filepath.Join(testCertDir, "scalingengine.crt")
		conf.Server.TLS.CACertFile = filepath.Join(testCertDir, "autoscaler-ca.crt")

		conf.Logging.Level = "info"

		conf.Db.PolicyDbUrl = os.Getenv("DBURL")
		conf.Db.ScalingEngineDbUrl = os.Getenv("DBURL")
		conf.Db.SchedulerDbUrl = os.Getenv("DBURL")
		conf.Synchronizer.ActiveScheduleSyncInterval = 10 * time.Minute

		conf.Consul.Cluster = consulRunner.ConsulCluster()
		conf.DefaultCoolDownSecs = 300
		conf.LockSize = 32

		configFile = writeConfig(&conf)

		testDB, err := sql.Open(db.PostgresDriverName, os.Getenv("DBURL"))
		Expect(err).NotTo(HaveOccurred())

		_, err = testDB.Exec("DELETE FROM scalinghistory WHERE appid = $1", appId)
		Expect(err).NotTo(HaveOccurred())

		_, err = testDB.Exec("DELETE from policy_json WHERE app_id = $1", appId)
		Expect(err).NotTo(HaveOccurred())

		_, err = testDB.Exec("DELETE from activeschedule WHERE appid = $1", appId)
		Expect(err).NotTo(HaveOccurred())

		_, err = testDB.Exec("DELETE from app_scaling_active_schedule WHERE app_id = $1", appId)
		Expect(err).NotTo(HaveOccurred())

		policy := `
		{
 			"instance_min_count": 1,
  			"instance_max_count": 5
		}`
		_, err = testDB.Exec("INSERT INTO policy_json(app_id, policy_json, guid) values($1, $2, $3)", appId, policy, "1234")
		Expect(err).NotTo(HaveOccurred())

		err = testDB.Close()
		Expect(err).NotTo(HaveOccurred())

		tlsConfig, err := cfhttp.NewTLSConfig(
			filepath.Join(testCertDir, "eventgenerator.crt"),
			filepath.Join(testCertDir, "eventgenerator.key"),
			filepath.Join(testCertDir, "autoscaler-ca.crt"))
		Expect(err).NotTo(HaveOccurred())
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}

	})

var _ = SynchronizedAfterSuite(
	func() {
		if consulRunner != nil {
			consulRunner.Stop()
		}
		ccUAA.Close()
		os.Remove(configFile.Name())
	},
	func() {
		gexec.CleanupBuildArtifacts()
	})

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "engine")
	Expect(err).NotTo(HaveOccurred())

	defer cfg.Close()

	bytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())

	_, err = cfg.Write(bytes)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}

type ScalingEngineRunner struct {
	configPath string
	startCheck string
	Session    *gexec.Session
}

func NewScalingEngineRunner() *ScalingEngineRunner {
	return &ScalingEngineRunner{
		configPath: configFile.Name(),
		startCheck: "scalingengine.started",
	}
}

func (engine *ScalingEngineRunner) Start() {
	engineSession, err := gexec.Start(
		exec.Command(
			enginePath,
			"-c",
			engine.configPath,
		),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[engine]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[engine]\x1b[0m ", GinkgoWriter),
	)
	Expect(err).NotTo(HaveOccurred())

	if engine.startCheck != "" {
		Eventually(engineSession.Buffer, 2).Should(gbytes.Say(engine.startCheck))
	}

	engine.Session = engineSession
}

func (engine *ScalingEngineRunner) Interrupt() {
	if engine.Session != nil {
		engine.Session.Interrupt().Wait(5 * time.Second)
	}
}

func (engine *ScalingEngineRunner) KillWithFire() {
	if engine.Session != nil {
		engine.Session.Kill().Wait(5 * time.Second)
	}
}
