package integration_test

import (
	_ "embed"
	"text/template"

	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	apiConfig "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	egConfig "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	opConfig "code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/config"
	seConfig "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"

	"github.com/go-sql-driver/mysql"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit/ginkgomon_v2"
	"gopkg.in/yaml.v3"
)

const (
	GolangAPIServer     = "golangApiServer"
	GolangServiceBroker = "golangServiceBroker"
	Scheduler           = "scheduler"
	MetricsCollector    = "metricsCollector"
	EventGenerator      = "eventGenerator"
	CfEventGenerator    = "cfEventGenerator"
	ScalingEngine       = "scalingEngine"
	Operator            = "operator"
)

var golangAPIInfoFilePath = "../api/exampleconfig/catalog-example.json"
var golangSchemaValidationPath = "../api/schemas/catalog.schema.json"
var golangApiServerPolicySchemaPath = "../api/policyvalidator/policy_json.schema.json"
var golangServiceCatalogPath = "../servicebroker/config/catalog.json"

//go:embed scheduler_application.template.yml
var schedulerApplicationConfigTemplate string

type Executables map[string]string
type Ports map[string]int

type Components struct {
	Executables Executables
	Ports       Ports
}

type DBConfig struct {
	URI            string `json:"uri"`
	MinConnections int    `json:"minConnections"`
	MaxConnections int    `json:"maxConnections"`
	IdleTimeout    int    `json:"idleTimeout"`
}
type SchedulerClient struct {
	Uri string          `json:"uri"`
	TLS models.TLSCerts `json:"tls"`
}
type ScalingEngineClient struct {
	Uri string          `json:"uri"`
	TLS models.TLSCerts `json:"tls"`
}
type MetricsCollectorClient struct {
	Uri string          `json:"uri"`
	TLS models.TLSCerts `json:"tls"`
}
type EventGeneratorClient struct {
	Uri string          `json:"uri"`
	TLS models.TLSCerts `json:"tls"`
}
type ServiceOffering struct {
	Enabled             bool                `json:"enabled"`
	ServiceBrokerClient ServiceBrokerClient `json:"serviceBroker"`
}
type ServiceBrokerClient struct {
	Uri string          `json:"uri"`
	TLS models.TLSCerts `json:"tls"`
}

func (components *Components) GolangAPIServer(confPath string, argv ...string) *ginkgomon_v2.Runner {
	return ginkgomon_v2.New(ginkgomon_v2.Config{
		Name:              GolangAPIServer,
		AnsiColorCode:     "33m",
		StartCheck:        `"api.started"`,
		StartCheckTimeout: 20 * time.Second,
		// #nosec G204
		Command: exec.Command(
			components.Executables[GolangAPIServer],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}
func (components *Components) Scheduler(confPath string, argv ...string) *ginkgomon_v2.Runner {
	return ginkgomon_v2.New(ginkgomon_v2.Config{
		Name:              Scheduler,
		AnsiColorCode:     "34m",
		StartCheck:        "Scheduler is ready to start",
		StartCheckTimeout: 120 * time.Second,
		// #nosec G204
		Command: exec.Command(
			"java", append([]string{"-jar", "-Dspring.config.location=" + confPath, components.Executables[Scheduler]}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}

func (components *Components) EventGenerator(confPath string, argv ...string) *ginkgomon_v2.Runner {
	return ginkgomon_v2.New(ginkgomon_v2.Config{
		Name:              EventGenerator,
		AnsiColorCode:     "36m",
		StartCheck:        `"eventgenerator.started"`,
		StartCheckTimeout: 20 * time.Second,
		// #nosec G204
		Command: exec.Command(
			components.Executables[EventGenerator],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) ScalingEngine(confPath string, argv ...string) *ginkgomon_v2.Runner {
	return ginkgomon_v2.New(ginkgomon_v2.Config{
		Name:              ScalingEngine,
		AnsiColorCode:     "31m",
		StartCheck:        `"scalingengine.started"`,
		StartCheckTimeout: 20 * time.Second,
		// #nosec G204
		Command: exec.Command(
			components.Executables[ScalingEngine],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) Operator(confPath string, argv ...string) *ginkgomon_v2.Runner {
	return ginkgomon_v2.New(ginkgomon_v2.Config{
		Name:              Operator,
		AnsiColorCode:     "38m",
		StartCheck:        `"operator.started"`,
		StartCheckTimeout: 40 * time.Second,
		// #nosec G204
		Command: exec.Command(
			components.Executables[Operator],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) PrepareGolangApiServerConfig(dbURI string, publicApiPort int, brokerPort int, cfApi string, schedulerUri string, scalingEngineUri string, eventGeneratorUri string, metricsForwarderUri string, tmpDir string) string {
	brokerCred1 := apiConfig.BrokerCredentialsConfig{
		BrokerUsername: "broker_username",
		//BrokerUsernameHash: []byte("$2a$10$WNO1cPko4iDAT6MkhaDojeJMU8ZdNH6gt.SapsFOsC0OF4cQ9qQwu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_username")'
		BrokerPassword: "broker_password",
		//BrokerPasswordHash: []byte("$2a$10$evLviRLcIPKnWQqlBl3DJOvBZir9vJ4gdEeyoGgvnK/CGBnxIAFRu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_password")'
	}
	brokerCred2 := apiConfig.BrokerCredentialsConfig{
		BrokerUsername: "broker_username2",
		//	BrokerUsernameHash: []byte("$2a$10$NK76ms9n/oeD1.IumovhIu2fiiQ/4FIVc81o4rdNS8beJMxYvhTqG"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_username2")'
		BrokerPassword: "broker_password2",
		//	BrokerPasswordHash: []byte("$2a$10$HZOfLweDfjNfe2h3KItdg.26BxNU6TVKMDwhJMNPPIWpj7T2HCVbW"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_password2")'
	}
	var brokerCreds []apiConfig.BrokerCredentialsConfig
	brokerCreds = append(brokerCreds, brokerCred1, brokerCred2)

	cfg := apiConfig.Config{
		Logging: helpers.LoggingConfig{
			Level: LOGLEVEL,
		},
		Server: helpers.ServerConfig{
			Port: publicApiPort,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "api.key"),
				CertFile:   filepath.Join(testCertDir, "api.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		BrokerServer: helpers.ServerConfig{
			Port: brokerPort,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "servicebroker.key"),
				CertFile:   filepath.Join(testCertDir, "servicebroker.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Db: map[string]db.DatabaseConfig{
			"policy_db": {
				URL: dbURI,
			},
			"binding_db": {
				URL: dbURI,
			},
		},
		BrokerCredentials:    brokerCreds,
		CatalogPath:          golangServiceCatalogPath,
		CatalogSchemaPath:    golangSchemaValidationPath,
		DashboardRedirectURI: "",
		PolicySchemaPath:     golangApiServerPolicySchemaPath,
		Scheduler: apiConfig.SchedulerConfig{
			SchedulerURL: schedulerUri,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scheduler.key"),
				CertFile:   filepath.Join(testCertDir, "scheduler.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		ScalingEngine: apiConfig.ScalingEngineConfig{
			ScalingEngineUrl: scalingEngineUri,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
				CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		EventGenerator: apiConfig.EventGeneratorConfig{
			EventGeneratorUrl: eventGeneratorUri,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		CF: cf.Config{
			API:      cfApi,
			ClientID: "admin",
			Secret:   "admin",
		},
		InfoFilePath: golangAPIInfoFilePath,
		MetricsForwarder: apiConfig.MetricsForwarderConfig{
			MetricsForwarderUrl: metricsForwarderUri,
		},
		RateLimit: models.RateLimitConfig{
			MaxAmount:     10,
			ValidDuration: 1 * time.Second,
		},
		CredHelperImpl:                     "default",
		DefaultCustomMetricsCredentialType: "binding-secret",
	}

	return writeYmlConfig(tmpDir, GolangAPIServer, &cfg)
}

func (components *Components) PrepareSchedulerConfig(dbUri string, scalingEngineUri string, tmpDir string, httpClientTimeout time.Duration) string {
	var (
		driverClassName string
		userName        string
		password        string
		jdbcDBUri       string
	)
	if strings.Contains(dbUri, "postgres") {
		dbUrl, _ := url.Parse(dbUri)
		scheme := dbUrl.Scheme
		host := dbUrl.Host
		path := dbUrl.Path
		userInfo := dbUrl.User
		userName = userInfo.Username()
		password, _ = userInfo.Password()
		if scheme == "postgres" {
			scheme = "postgresql"
		}
		jdbcDBUri = fmt.Sprintf("jdbc:%s://%s%s", scheme, host, path)
		driverClassName = "org.postgresql.Driver"
	} else {
		cfg, _ := mysql.ParseDSN(dbUri)
		scheme := "mysql"
		host := cfg.Addr
		path := cfg.DBName
		userName = cfg.User
		password = cfg.Passwd
		jdbcDBUri = fmt.Sprintf("jdbc:%s://%s/%s", scheme, host, path)
		driverClassName = "com.mysql.cj.jdbc.Driver"
	}

	type TemplateParameters struct {
		ScalingEngineUri  string
		HttpClientTimeout int
		TestCertDir       string
		Port              int
		DriverClassName   string
		DBUser            string
		DBPassword        string
		JDBCURI           string
	}

	templateParameters := TemplateParameters{
		ScalingEngineUri:  scalingEngineUri,
		HttpClientTimeout: int(httpClientTimeout / time.Second),
		TestCertDir:       testCertDir,
		Port:              components.Ports[Scheduler],
		DriverClassName:   driverClassName,
		DBUser:            userName,
		DBPassword:        password,
		JDBCURI:           jdbcDBUri,
	}

	ut, err := template.New("application.yaml").Parse(schedulerApplicationConfigTemplate)
	Expect(err).NotTo(HaveOccurred())

	cfgFile, err := os.Create(filepath.Join(tmpDir, "application.yaml"))
	Expect(err).NotTo(HaveOccurred())

	err = ut.Execute(cfgFile, templateParameters)
	Expect(err).NotTo(HaveOccurred())
	cfgFile.Close()
	return cfgFile.Name()
}

func (components *Components) PrepareEventGeneratorConfig(dbUri string, port int, metricsCollectorURL string, scalingEngineURL string, aggregatorExecuteInterval time.Duration,
	policyPollerInterval time.Duration, saveInterval time.Duration, evaluationManagerInterval time.Duration, httpClientTimeout time.Duration, tmpDir string) string {
	conf := &egConfig.Config{
		Logging: helpers.LoggingConfig{
			Level: LOGLEVEL,
		},
		CFServer: helpers.ServerConfig{
			Port: components.Ports[CfEventGenerator],
		},
		Server: egConfig.ServerConfig{
			ServerConfig: helpers.ServerConfig{
				Port: port,
				TLS: models.TLSCerts{
					KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
					CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
					CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
				},
			},
			NodeAddrs: []string{"localhost"},
			NodeIndex: 0,
		},
		Aggregator: egConfig.AggregatorConfig{
			AggregatorExecuteInterval: aggregatorExecuteInterval,
			PolicyPollerInterval:      policyPollerInterval,
			SaveInterval:              saveInterval,
			MetricPollerCount:         1,
			AppMonitorChannelSize:     1,
			AppMetricChannelSize:      1,
			MetricCacheSizePerApp:     50,
		},
		Evaluator: egConfig.EvaluatorConfig{
			EvaluationManagerInterval: evaluationManagerInterval,
			EvaluatorCount:            1,
			TriggerArrayChannelSize:   1,
		},
		DB: egConfig.DBConfig{
			PolicyDB: db.DatabaseConfig{
				URL: dbUri,
			},
			AppMetricDB: db.DatabaseConfig{
				URL: dbUri,
			},
		},
		ScalingEngine: egConfig.ScalingEngineConfig{
			ScalingEngineURL: scalingEngineURL,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		MetricCollector: egConfig.MetricCollectorConfig{
			MetricCollectorURL: metricsCollectorURL,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		DefaultBreachDurationSecs: 600,
		DefaultStatWindowSecs:     60,
		HttpClientTimeout:         httpClientTimeout,
	}
	return writeYmlConfig(tmpDir, EventGenerator, &conf)
}

func (components *Components) PrepareScalingEngineConfig(dbURI string, port int, ccUAAURL string, httpClientTimeout time.Duration, tmpDir string) string {
	conf := seConfig.Config{
		CF: cf.Config{
			API:      ccUAAURL,
			ClientID: "admin",
			Secret:   "admin",
		},
		Server: helpers.ServerConfig{
			Port: port,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
				CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Logging: helpers.LoggingConfig{
			Level: LOGLEVEL,
		},
		DB: seConfig.DBConfig{
			PolicyDB: db.DatabaseConfig{
				URL: dbURI,
			},
			ScalingEngineDB: db.DatabaseConfig{
				URL: dbURI,
			},
			SchedulerDB: db.DatabaseConfig{
				URL: dbURI,
			},
		},
		DefaultCoolDownSecs: 300,
		LockSize:            32,
		HttpClientTimeout:   httpClientTimeout,
	}

	return writeYmlConfig(tmpDir, ScalingEngine, &conf)
}

func (components *Components) PrepareOperatorConfig(dbURI string, ccUAAURL string, scalingEngineURL string, schedulerURL string, syncInterval time.Duration, cutoffDuration time.Duration, httpClientTimeout time.Duration, tmpDir string) string {
	conf := &opConfig.Config{
		Logging: helpers.LoggingConfig{
			Level: LOGLEVEL,
		},
		CF: cf.Config{
			API:      ccUAAURL,
			ClientID: "admin",
			Secret:   "admin",
		},
		AppMetricsDB: opConfig.DbPrunerConfig{
			RefreshInterval: 2 * time.Minute,
			CutoffDuration:  cutoffDuration,
			DB: db.DatabaseConfig{
				URL: dbURI,
			},
		},
		ScalingEngineDB: opConfig.DbPrunerConfig{
			RefreshInterval: 2 * time.Minute,
			CutoffDuration:  cutoffDuration,
			DB: db.DatabaseConfig{
				URL: dbURI,
			},
		},
		ScalingEngine: opConfig.ScalingEngineConfig{
			URL:          scalingEngineURL,
			SyncInterval: syncInterval,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
				CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Scheduler: opConfig.SchedulerConfig{
			URL:          schedulerURL,
			SyncInterval: syncInterval,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scheduler.key"),
				CertFile:   filepath.Join(testCertDir, "scheduler.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		DBLock: opConfig.DBLockConfig{
			LockTTL: 30 * time.Second,
			DB: db.DatabaseConfig{
				URL: dbURI,
			},
			LockRetryInterval: 15 * time.Second,
		},
		AppSyncer: opConfig.AppSyncerConfig{
			SyncInterval: 60 * time.Second,
			DB: db.DatabaseConfig{
				URL: dbURI,
			},
		},
		HttpClientTimeout: httpClientTimeout,
	}
	return writeYmlConfig(tmpDir, Operator, &conf)
}

func writeYmlConfig(dir string, componentName string, c interface{}) string {
	cfgFile, err := os.CreateTemp(dir, componentName)
	Expect(err).NotTo(HaveOccurred())
	defer cfgFile.Close()
	configBytes, err := yaml.Marshal(c)
	Expect(err).NotTo(HaveOccurred())
	err = os.WriteFile(cfgFile.Name(), configBytes, 0600)
	Expect(err).NotTo(HaveOccurred())
	return cfgFile.Name()
}
