package main_test

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/models"
	"autoscaler/scalingengine/config"

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
)

func TestScalingengine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scalingengine Suite")
}

var (
	enginePath string
	conf       config.Config
	port       int
	configFile *os.File
	ccUAA      *ghttp.Server
	appId      string
)

var _ = SynchronizedBeforeSuite(
	func() []byte {
		compiledPath, err := gexec.Build("autoscaler/scalingengine/cmd/scalingengine", "-race")
		Expect(err).NotTo(HaveOccurred())
		return []byte(compiledPath)
	},
	func(pathBytes []byte) {
		enginePath = string(pathBytes)

		ccUAA = ghttp.NewServer()
		ccUAA.RouteToHandler("GET", "/v2/info", ghttp.RespondWithJSONEncoded(http.StatusOK,
			cf.Endpoints{
				AuthEndpoint:    ccUAA.URL(),
				DopplerEndpoint: strings.Replace(ccUAA.URL(), "http", "ws", 1),
			}))

		ccUAA.RouteToHandler("POST", "/oauth/token", ghttp.RespondWithJSONEncoded(http.StatusOK, cf.Tokens{}))

		appId = fmt.Sprintf("%s-%d", "app-id", GinkgoParallelNode())
		ccUAA.RouteToHandler("GET", "/v2/apps/"+appId, ghttp.RespondWithJSONEncoded(http.StatusOK,
			models.AppInfo{Entity: models.AppEntity{Instances: 2}}))
		ccUAA.RouteToHandler("PUT", "/v2/apps/"+appId, ghttp.RespondWith(http.StatusCreated, ""))

		conf.Cf = cf.CfConfig{
			Api:       ccUAA.URL(),
			GrantType: cf.GrantTypePassword,
			Username:  "admin",
			Password:  "admin",
		}

		port = 7000 + GinkgoParallelNode()
		conf.Server.Port = port

		conf.Logging.Level = "debug"

		conf.Db.PolicyDbUrl = os.Getenv("DBURL")
		conf.Db.HistoryDbUrl = os.Getenv("DBURL")

		configFile = writeConfig(&conf)

		engineDB, err := sql.Open(db.PostgresDriverName, os.Getenv("DBURL"))
		Expect(err).NotTo(HaveOccurred())

		_, err = engineDB.Exec("DELETE FROM scalinghistory WHERE appid = $1", appId)
		Expect(err).NotTo(HaveOccurred())

		_, err = engineDB.Exec("DELETE from policy_json WHERE app_id = $1", appId)
		Expect(err).NotTo(HaveOccurred())

		policy := `
		{
 			"instance_min_count": 1,
  			"instance_max_count": 5
		}`
		_, err = engineDB.Exec("INSERT INTO policy_json(app_id, policy_json) values($1, $2)", appId, policy)
		Expect(err).NotTo(HaveOccurred())

		err = engineDB.Close()
		Expect(err).NotTo(HaveOccurred())

	})

var _ = SynchronizedAfterSuite(
	func() {
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
		Eventually(engineSession.Buffer(), 2).Should(gbytes.Say(engine.startCheck))
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
