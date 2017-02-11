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
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"gopkg.in/yaml.v2"
)

var (
	egPath          string
	testAppId       string = "testAppId"
	metricType      string = models.MetricNameMemory
	timestamp       int64  = time.Now().UnixNano()
	regPath                = regexp.MustCompile(`^/v1/apps/.*/scale$`)
	configFile      *os.File
	conf            *config.Config
	metricCollector *ghttp.Server
	scalingEngine   *ghttp.Server
	metrics         []*models.AppInstanceMetric = []*models.AppInstanceMetric{
		&models.AppInstanceMetric{
			AppId:         testAppId,
			InstanceIndex: 0,
			CollectedAt:   111111,
			Name:          metricType,
			Unit:          models.UnitMegaBytes,
			Value:         "500",
			Timestamp:     111100,
		},
		&models.AppInstanceMetric{
			AppId:         testAppId,
			InstanceIndex: 1,
			CollectedAt:   111111,
			Name:          metricType,
			Unit:          models.UnitMegaBytes,
			Value:         "600",
			Timestamp:     110000,
		},

		&models.AppInstanceMetric{
			AppId:         testAppId,
			InstanceIndex: 0,
			CollectedAt:   222222,
			Name:          metricType,
			Unit:          models.UnitMegaBytes,
			Value:         "700",
			Timestamp:     222200,
		},
		&models.AppInstanceMetric{
			AppId:         testAppId,
			InstanceIndex: 1,
			CollectedAt:   222222,
			Name:          metricType,
			Unit:          models.UnitMegaBytes,
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
	return []byte(eg)
}, func(pathByte []byte) {
	egPath = string(pathByte)
	initDB()
	initHttpEndPoints()
	initConfig()

})

var _ = SynchronizedAfterSuite(func() {
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

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
		         "metric_type":"memoryused",
		         "stat_window_secs":300,
		         "breach_duration_secs":300,
		         "threshold":300,
		         "operator":">",
		         "cool_down_secs":300,
		         "adjustment":"+1"
		      }
		   ]
		}`
	query := "INSERT INTO policy_json(app_id, policy_json) values($1, $2)"
	_, err = egDB.Exec(query, testAppId, policy)
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

	metricCollector.RouteToHandler("GET", "/v1/apps/"+testAppId+"/metric_histories/memoryused", ghttp.RespondWithJSONEncoded(http.StatusOK,
		&metrics))
	scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWith(http.StatusOK, "successful"))
}

func initConfig() {
	testCertDir := "../../../../../test-certs"
	conf := &config.Config{
		Server: config.ServerConfig{
			Port: config.DefaultServerPort + 1,
		},
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
	configPath string
	startCheck string
	Session    *gexec.Session
}

func NewEventGeneratorRunner() *EventGeneratorRunner {
	return &EventGeneratorRunner{
		configPath: configFile.Name(),
		startCheck: "eventgenerator.started",
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
		Eventually(egSession.Buffer(), 2).Should(gbytes.Say(eg.startCheck))
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
