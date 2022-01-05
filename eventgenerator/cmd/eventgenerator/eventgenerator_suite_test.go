package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"code.cloudfoundry.org/cfhttp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	yaml "gopkg.in/yaml.v2"
)

var (
	egPath     string
	testAppId  = "an-app-id"
	metricType = "a-metric-type"
	metricUnit = "a-metric-unit"

	regPath            = regexp.MustCompile(`^/v1/apps/.*/scale$`)
	configFile         *os.File
	conf               config.Config
	egPort             int
	healthport         int
	httpClient         *http.Client
	healthHttpClient   *http.Client
	metricCollector    *ghttp.Server
	scalingEngine      *ghttp.Server
	breachDurationSecs = 10
	metrics            = []*models.AppInstanceMetric{
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

	scalingResult = &models.AppScalingResult{
		AppId:             testAppId,
		Adjustment:        1,
		Status:            models.ScalingStatusSucceeded,
		CooldownExpiredAt: time.Now().Add(time.Duration(300) * time.Second).UnixNano(),
	}
)

func TestEventgenerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Eventgenerator Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	eg, err := gexec.Build("code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/cmd/eventgenerator", "-race")
	Expect(err).NotTo(HaveOccurred())
	initDB()
	return []byte(eg)
}, func(pathByte []byte) {
	egPath = string(pathByte)
	initHttpEndPoints()
	initConfig()
})

var _ = SynchronizedAfterSuite(func() {
	os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func initDB() {
	database, err := db.GetConnection(os.Getenv("DBURL"))
	Expect(err).NotTo(HaveOccurred())

	egDB, err := sqlx.Open(database.DriverName, database.DSN)
	Expect(err).NotTo(HaveOccurred())

	_, err = egDB.Exec("DELETE FROM app_metric")
	Expect(err).NotTo(HaveOccurred())

	_, err = egDB.Exec("DELETE from policy_json")
	Expect(err).NotTo(HaveOccurred())

	policy := fmt.Sprintf(`
		{
		   "instance_min_count":1,
		   "instance_max_count":5,
		   "scaling_rules":[
		      {
		         "metric_type":"a-metric-type",
		         "breach_duration_secs":%d,
		         "threshold":300,
		         "operator":">",
		         "cool_down_secs":300,
		         "adjustment":"+1"
		      }
		   ]
		}`, breachDurationSecs)
	query := egDB.Rebind("INSERT INTO policy_json(app_id, policy_json, guid) values(?, ?, ?)")
	_, err = egDB.Exec(query, testAppId, policy, "1234")
	Expect(err).NotTo(HaveOccurred())

	err = egDB.Close()
	Expect(err).NotTo(HaveOccurred())
	healthHttpClient = &http.Client{}
}

func initHttpEndPoints() {
	testCertDir := "../../../../../test-certs"

	_, err := ioutil.ReadFile(filepath.Join(testCertDir, "eventgenerator.key"))
	Expect(err).NotTo(HaveOccurred())
	_, err = ioutil.ReadFile(filepath.Join(testCertDir, "eventgenerator.crt"))
	Expect(err).NotTo(HaveOccurred())
	_, err = ioutil.ReadFile(filepath.Join(testCertDir, "autoscaler-ca.crt"))
	Expect(err).NotTo(HaveOccurred())

	//nolint:staticcheck  // SA1019 TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/548
	mcTLSConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, "metricscollector.crt"),
		filepath.Join(testCertDir, "metricscollector.key"),
		filepath.Join(testCertDir, "autoscaler-ca.crt"))
	Expect(err).NotTo(HaveOccurred())

	metricCollector = ghttp.NewUnstartedServer()
	metricCollector.HTTPTestServer.TLS = mcTLSConfig
	metricCollector.HTTPTestServer.StartTLS()

	//nolint:staticcheck  // SA1019 TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/548
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
	scalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult))
}

func initConfig() {
	testCertDir := "../../../../../test-certs"

	egPort = 7000 + GinkgoParallelProcess()
	healthport = 8000 + GinkgoParallelProcess()
	conf = config.Config{
		Logging: helpers.LoggingConfig{
			Level: "debug",
		},
		Server: config.ServerConfig{
			Port: egPort,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
			NodeAddrs: []string{"localhost"},
			NodeIndex: 0,
		},
		Aggregator: config.AggregatorConfig{
			AggregatorExecuteInterval: 1 * time.Second,
			PolicyPollerInterval:      1 * time.Second,
			SaveInterval:              1 * time.Second,
			MetricPollerCount:         1,
			AppMonitorChannelSize:     1,
			AppMetricChannelSize:      1,
			MetricCacheSizePerApp:     500,
		},
		Evaluator: config.EvaluatorConfig{
			EvaluationManagerInterval: 1 * time.Second,
			EvaluatorCount:            1,
			TriggerArrayChannelSize:   1,
		},
		DB: config.DBConfig{
			PolicyDB: db.DatabaseConfig{
				URL:                   os.Getenv("DBURL"),
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 10 * time.Second,
			},
			AppMetricDB: db.DatabaseConfig{
				URL:                   os.Getenv("DBURL"),
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 10 * time.Second,
			},
		},
		ScalingEngine: config.ScalingEngineConfig{
			ScalingEngineURL: scalingEngine.URL(),
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		MetricCollector: config.MetricCollectorConfig{
			MetricCollectorURL: metricCollector.URL(),
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			BackOffInitialInterval:  5 * time.Minute,
			BackOffMaxInterval:      2 * time.Hour,
			ConsecutiveFailureCount: 3,
		},
		DefaultBreachDurationSecs: 600,
		DefaultStatWindowSecs:     300,
		HttpClientTimeout:         10 * time.Second,
		Health: models.HealthConfig{
			Port:                healthport,
			HealthCheckUsername: "healthcheckuser",
			HealthCheckPassword: "healthcheckpassword",
		},
	}
	configFile = writeConfig(&conf)

	//nolint:staticcheck  // SA1019 TODO: https://github.com/cloudfoundry/app-autoscaler-release/issues/548
	tlsConfig, err := cfhttp.NewTLSConfig(
		filepath.Join(testCertDir, "api.crt"),
		filepath.Join(testCertDir, "api.key"),
		filepath.Join(testCertDir, "autoscaler-ca.crt"))
	Expect(err).NotTo(HaveOccurred())
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
}

func writeConfig(c *config.Config) *os.File {
	cfg, err := ioutil.TempFile("", "eg")
	Expect(err).NotTo(HaveOccurred())
	defer cfg.Close()
	configBytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())
	err = ioutil.WriteFile(cfg.Name(), configBytes, 0600)
	Expect(err).NotTo(HaveOccurred())
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
	// #nosec G204
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
