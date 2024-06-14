package main_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"
	rpc "code.cloudfoundry.org/go-log-cache/v2/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"fmt"
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

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	yaml "gopkg.in/yaml.v3"
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
	mockLogCache       *testhelpers.MockLogCache
	mockScalingEngine  *ghttp.Server
	breachDurationSecs = 10

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
	healthHttpClient = &http.Client{}
	egPath = string(pathByte)
	initHttpEndPoints()
	initConfig()
	httpClient = testhelpers.NewApiClient()
})

var _ = SynchronizedAfterSuite(func() {
	_ = os.Remove(configFile.Name())
}, func() {
	gexec.CleanupBuildArtifacts()
})

func initDB() {
	dbUrl := testhelpers.GetDbUrl()
	database, err := db.GetConnection(dbUrl)

	Expect(err).NotTo(HaveOccurred())

	egDB, err := sqlx.Open(database.DriverName, database.DSN)
	defer func() { _ = egDB.Close() }()
	Expect(err).NotTo(HaveOccurred())

	_, err = egDB.Exec("DELETE FROM app_metric")
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to clean app_metric %s", err.Error()))
	}

	_, err = egDB.Exec("DELETE from policy_json")
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to clean policy_json %s", err.Error()))
	}

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
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to insert policy %s", err.Error()))
	}
}

func initHttpEndPoints() {
	testCertDir := testhelpers.TestCertFolder()
	tlsConfig, err := testhelpers.NewTLSConfig(
		filepath.Join(testCertDir, "autoscaler-ca.crt"),
		filepath.Join(testCertDir, "log-cache.crt"),
		filepath.Join(testCertDir, "log-cache.key"),
		"log-cache",
	)
	Expect(err).ToNot(HaveOccurred())
	mockLogCache = testhelpers.NewMockLogCache(tlsConfig)
	mockLogCache.ReadReturns(testAppId, &rpc.ReadResponse{
		Envelopes: &loggregator_v2.EnvelopeBatch{
			Batch: []*loggregator_v2.Envelope{
				{
					SourceId:   testAppId,
					InstanceId: "0",
					Timestamp:  111100,
					DeprecatedTags: map[string]*loggregator_v2.Value{
						"origin": {
							Data: &loggregator_v2.Value_Text{
								Text: "autoscaler_metrics_forwarder",
							},
						},
					},
					Message: &loggregator_v2.Envelope_Gauge{
						Gauge: &loggregator_v2.Gauge{
							Metrics: map[string]*loggregator_v2.GaugeValue{
								metricType: {
									Unit:  metricUnit,
									Value: 500,
								},
							},
						},
					},
				},
				{
					SourceId:   testAppId,
					InstanceId: "1",
					Timestamp:  110000,
					DeprecatedTags: map[string]*loggregator_v2.Value{
						"origin": {
							Data: &loggregator_v2.Value_Text{
								Text: "autoscaler_metrics_forwarder",
							},
						},
					},
					Message: &loggregator_v2.Envelope_Gauge{
						Gauge: &loggregator_v2.Gauge{
							Metrics: map[string]*loggregator_v2.GaugeValue{
								metricType: {
									Unit:  metricUnit,
									Value: 600,
								},
							},
						},
					},
				},
				{
					SourceId:   testAppId,
					InstanceId: "0",
					Timestamp:  222200,
					DeprecatedTags: map[string]*loggregator_v2.Value{
						"origin": {
							Data: &loggregator_v2.Value_Text{
								Text: "autoscaler_metrics_forwarder",
							},
						},
					},
					Message: &loggregator_v2.Envelope_Gauge{
						Gauge: &loggregator_v2.Gauge{
							Metrics: map[string]*loggregator_v2.GaugeValue{
								metricType: {
									Unit:  metricUnit,
									Value: 700,
								},
							},
						},
					},
				},
				{
					SourceId:   testAppId,
					InstanceId: "1",
					Timestamp:  220000,
					DeprecatedTags: map[string]*loggregator_v2.Value{
						"origin": {
							Data: &loggregator_v2.Value_Text{
								Text: "autoscaler_metrics_forwarder",
							},
						},
					},
					Message: &loggregator_v2.Envelope_Gauge{
						Gauge: &loggregator_v2.Gauge{
							Metrics: map[string]*loggregator_v2.GaugeValue{
								metricType: {
									Unit:  metricUnit,
									Value: 800,
								},
							},
						},
					},
				},
			},
		},
	}, nil)
	err = mockLogCache.Start(10000 + GinkgoParallelProcess())
	Expect(err).ToNot(HaveOccurred())

	mockScalingEngine = ghttp.NewUnstartedServer()
	mockScalingEngine.HTTPTestServer.TLS = testhelpers.ServerTlsConfig("scalingengine")
	mockScalingEngine.HTTPTestServer.StartTLS()

	mockScalingEngine.RouteToHandler("POST", regPath, ghttp.RespondWithJSONEncoded(http.StatusOK, &scalingResult))
}

func initConfig() {
	testCertDir := testhelpers.TestCertFolder()

	egPort = 7000 + GinkgoParallelProcess()
	healthport = 8000 + GinkgoParallelProcess()
	dbUrl := testhelpers.GetDbUrl()
	conf = config.Config{
		Logging: helpers.LoggingConfig{
			Level: "debug",
		},
		Server: config.ServerConfig{
			ServerConfig: helpers.ServerConfig{
				Port: egPort,
				TLS: models.TLSCerts{
					KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
					CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
					CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
				},
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
				URL:                   dbUrl,
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 10 * time.Second,
			},
			AppMetricDB: db.DatabaseConfig{
				URL:                   dbUrl,
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 10 * time.Second,
			},
		},
		ScalingEngine: config.ScalingEngineConfig{
			ScalingEngineURL: mockScalingEngine.URL(),
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		MetricCollector: config.MetricCollectorConfig{
			MetricCollectorURL: mockLogCache.URL(),
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
		Health: helpers.HealthConfig{
			ServerConfig: helpers.ServerConfig{
				Port: healthport,
			},
			HealthCheckUsername: "healthcheckuser",
			HealthCheckPassword: "healthcheckpassword",
		},
	}
	configFile = writeConfig(&conf)
}

func writeConfig(c *config.Config) *os.File {
	cfg, err := os.CreateTemp("", "eg")
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = cfg.Close() }()
	configBytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())
	err = os.WriteFile(cfg.Name(), configBytes, 0600)
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
