package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"autoscaler/db"
	"autoscaler/eventgenerator/config"
	"autoscaler/models"
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"code.cloudfoundry.org/cfhttp"
	"code.cloudfoundry.org/consuladapter/consulrunner"
	"code.cloudfoundry.org/locket"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"gopkg.in/yaml.v2"
)

var (
	egPath     string
	testAppId  string = "an-app-id"
	metricType string = "a-metric-type"
	metricUnit string = "a-metric-unit"

	regPath         = regexp.MustCompile(`^/v1/apps/.*/scale$`)
	configFile      *os.File
	conf            *config.Config
	metricCollector *ghttp.Server
	scalingEngine   *ghttp.Server
	consulRunner    *consulrunner.ClusterRunner
	metrics         []*models.AppInstanceMetric = []*models.AppInstanceMetric{
		{
			AppId:         testAppId,
			InstanceIndex: 0,
			CollectedAt:   111111,
			Name:          metricType,
			Unit:          metricUnit,
			Value:         "500",
			Timestamp:     111100,
		},
		{
			AppId:         testAppId,
			InstanceIndex: 1,
			CollectedAt:   111111,
			Name:          metricType,
			Unit:          metricUnit,
			Value:         "600",
			Timestamp:     110000,
		},

		{
			AppId:         testAppId,
			InstanceIndex: 0,
			CollectedAt:   222222,
			Name:          metricType,
			Unit:          metricUnit,
			Value:         "700",
			Timestamp:     222200,
		},
		{
			AppId:         testAppId,
			InstanceIndex: 1,
			CollectedAt:   222222,
			Name:          metricType,
			Unit:          metricUnit,
			Value:         "800",
			Timestamp:     220000,
		},
	}
)

func TestEventgenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Eventgenerator Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	eg, err := gexec.Build("autoscaler/eventgenerator/cmd/eventgenerator", "-race")
	Expect(err).NotTo(HaveOccurred())
	initDB()
	return []byte(eg)
}, func(pathByte []byte) {
	egPath = string(pathByte)
	initHttpEndPoints()
	initConsul()
	initConfig()
})

var _ = SynchronizedAfterSuite(func() {
	os.Remove(configFile.Name())
	if consulRunner != nil {
		consulRunner.Stop()
	}
}, func() {
	gexec.CleanupBuildArtifacts()
})

func initConsul() {
	consulRunner = consulrunner.NewClusterRunner(
		consulrunner.ClusterRunnerConfig{
			StartingPort: 9001 + GinkgoParallelNode()*consulrunner.PortOffsetLength,
			NumNodes:     1,
			Scheme:       "http",
		},
	)
	consulRunner.Start()
	consulRunner.WaitUntilReady()
}

func initDB() {
	egDB, err := sql.Open(db.PostgresDriverName, os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	_, err = egDB.Exec("DELETE FROM app_metric")
	Expect(err).NotTo(HaveOccurred())

	_, err = egDB.Exec("DELETE from policy_json")
	Expect(err).NotTo(HaveOccurred())

	policy := `
		{
		   "instance_min_count":1,
		   "instance_max_count":5,
		   "scaling_rules":[
		      {
		         "metric_type":"a-metric-type",
		         "stat_window_secs":300,
		         "breach_duration_secs":300,
		         "threshold":300,
		         "operator":">",
		         "cool_down_secs":300,
		         "adjustment":"+1"
		      }
		   ]
		}`
	query := "INSERT INTO policy_json(app_id, policy_json, guid) values($1, $2, $3)"
	_, err = egDB.Exec(query, testAppId, policy, "1234")
	Expect(err).NotTo(HaveOccurred())

	err = egDB.Close()
	Expect(err).NotTo(HaveOccurred())
}

func initHttpEndPoints() {
	testCertDir := "../../../../../test-certs"

	mcTLSConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, "metricscollector.crt"),
		filepath.Join(testCertDir, "metricscollector.key"),
		filepath.Join(testCertDir, "autoscaler-ca.crt"))
	Expect(err).NotTo(HaveOccurred())

	metricCollector = ghttp.NewUnstartedServer()
	metricCollector.HTTPTestServer.TLS = mcTLSConfig
	metricCollector.HTTPTestServer.StartTLS()

	seTLSConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, "scalingengine.crt"),
		filepath.Join(testCertDir, "scalingengine.key"),
		filepath.Join(testCertDir, "autoscaler-ca.crt"))
	Expect(err).NotTo(HaveOccurred())

	scalingEngine = ghttp.NewUnstartedServer()
	scalingEngine.HTTPTestServer.TLS = seTLSConfig
	scalingEngine.HTTPTestServer.StartTLS()

	metricCollector.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metric_histories/"+metricType, ghttp.RespondWithJSONEncoded(http.StatusOK,
		&metrics))
	scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
}

func initConfig() {
	testCertDir := "../../../../../test-certs"
	conf = &config.Config{
		Logging: config.LoggingConfig{
			Level: "debug",
		},
		Aggregator: config.AggregatorConfig{
			AggregatorExecuteInterval: 1 * time.Second,
			PolicyPollerInterval:      1 * time.Second,
			MetricPollerCount:         1,
			AppMonitorChannelSize:     1,
		},
		Evaluator: config.EvaluatorConfig{
			EvaluationManagerInterval: 1 * time.Second,
			EvaluatorCount:            1,
			TriggerArrayChannelSize:   1,
		},
		DB: config.DBConfig{
			PolicyDBUrl:    os.Getenv("DBURL"),
			AppMetricDBUrl: os.Getenv("DBURL"),
		},
		ScalingEngine: config.ScalingEngineConfig{
			ScalingEngineUrl: scalingEngine.URL(),
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		MetricCollector: config.MetricCollectorConfig{
			MetricCollectorUrl: metricCollector.URL(),
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Lock: config.LockConfig{
			LockRetryInterval:   locket.RetryInterval,
			LockTTL:             locket.DefaultSessionTTL,
			ConsulClusterConfig: consulRunner.ConsulCluster(),
		},
	}
	configFile = writeConfig(conf)
}

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "eg")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()
	configBytes, err1 := yaml.Marshal(c)
	ioutil.WriteFile(cfg.Name(), configBytes, 0777)
	Expect(err1).NotTo(HaveOccurred())
	return cfg

}

type EventGeneratorRunner struct {
	configPath        string
	startCheck        string
	acquiredLockCheck string
	Session           *gexec.Session
}

func NewEventGeneratorRunner() *EventGeneratorRunner {
	return &EventGeneratorRunner{
		configPath:        configFile.Name(),
		startCheck:        "eventgenerator.started",
		acquiredLockCheck: "eventgenerator.lock.acquire-lock-succeeded",
	}
}

func (eg *EventGeneratorRunner) Start() {
	egSession, err := gexec.Start(exec.Command(
		egPath,
		"-c",
		eg.configPath,
	),
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[32m[eg]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[32m[eg]\x1b[0m ", GinkgoWriter),
	)
	Expect(err).NotTo(HaveOccurred())

	if eg.startCheck != "" {
		Eventually(egSession.Buffer, 2).Should(gbytes.Say(eg.startCheck))
	}

	eg.Session = egSession
}

func (eg *EventGeneratorRunner) Interrupt() {
	if eg.Session != nil {
		eg.Session.Interrupt().Wait(5 * time.Second)
	}
}

func (eg *EventGeneratorRunner) KillWithFire() {
	if eg.Session != nil {
		eg.Session.Kill().Wait(5 * time.Second)
	}
}
