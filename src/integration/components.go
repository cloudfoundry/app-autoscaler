package integration

import (
	"autoscaler/cf"
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"

	apiConfig "autoscaler/api/config"
	egConfig "autoscaler/eventgenerator/config"
	mcConfig "autoscaler/metricscollector/config"
	mgConfig "autoscaler/metricsgateway/config"
	msConfig "autoscaler/metricsserver/config"
	opConfig "autoscaler/operator/config"
	seConfig "autoscaler/scalingengine/config"

	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit/ginkgomon"
	yaml "gopkg.in/yaml.v2"
	"github.com/go-sql-driver/mysql"
)

const (
	GolangAPIServer     = "golangApiServer"
	ServiceBroker       = "serviceBroker"
	GolangServiceBroker = "golangServiceBroker"
	Scheduler           = "scheduler"
	MetricsCollector    = "metricsCollector"
	EventGenerator      = "eventGenerator"
	ScalingEngine       = "scalingEngine"
	Operator            = "operator"
	MetricsGateway      = "metricsGateway"
	MetricsServerHTTP   = "metricsServerHTTP"
	MetricsServerWS     = "metricsServerWS"
)

var serviceCatalogPath string = "../../servicebroker/config/catalog.json"
var schemaValidationPath string = "../../servicebroker/config/catalog.schema.json"
var apiServerInfoFilePath string = "../../api/config/info.json"

var golangAPIInfoFilePath string = "../autoscaler/api/exampleconfig/catalog-example.json"
var golangSchemaValidationPath string = "../autoscaler/api/schemas/catalog.schema.json"
var golangApiServerPolicySchemaPath string = "../autoscaler/api/policyvalidator/policy_json.schema.json"
var golangServiceCatalogPath string = "../../servicebroker/config/catalog.json"

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

func (components *Components) GolangAPIServer(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              GolangAPIServer,
		AnsiColorCode:     "33m",
		StartCheck:        `"api.started"`,
		StartCheckTimeout: 20 * time.Second,
		Command: exec.Command(
			components.Executables[GolangAPIServer],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}
func (components *Components) Scheduler(confPath string, argv ...string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:              Scheduler,
		AnsiColorCode:     "34m",
		StartCheck:        "Scheduler is ready to start",
		StartCheckTimeout: 120 * time.Second,
		Command: exec.Command(
			"java", append([]string{"-jar", "-Dspring.config.location=" + confPath, components.Executables[Scheduler]}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}

func (components *Components) MetricsServer(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              MetricsServerHTTP,
		AnsiColorCode:     "33m",
		StartCheck:        `"metricsserver.started"`,
		StartCheckTimeout: 20 * time.Second,
		Command: exec.Command(
			components.Executables[MetricsServerHTTP],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) EventGenerator(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              EventGenerator,
		AnsiColorCode:     "36m",
		StartCheck:        `"eventgenerator.started"`,
		StartCheckTimeout: 20 * time.Second,
		Command: exec.Command(
			components.Executables[EventGenerator],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) ScalingEngine(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              ScalingEngine,
		AnsiColorCode:     "31m",
		StartCheck:        `"scalingengine.started"`,
		StartCheckTimeout: 20 * time.Second,
		Command: exec.Command(
			components.Executables[ScalingEngine],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) Operator(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              Operator,
		AnsiColorCode:     "38m",
		StartCheck:        `"operator.started"`,
		StartCheckTimeout: 40 * time.Second,
		Command: exec.Command(
			components.Executables[Operator],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) MetricsGateway(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              MetricsGateway,
		AnsiColorCode:     "32m",
		StartCheck:        `"metricsgateway.started"`,
		StartCheckTimeout: 20 * time.Second,
		Command: exec.Command(
			components.Executables[MetricsGateway],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) PrepareGolangApiServerConfig(dbURI string, publicApiPort int, brokerPort int, cfApi string, skipSSLValidation bool, cacheTTL int, schedulerUri string, scalingEngineUri string, metricsCollectorUri string, eventGeneratorUri string, metricsForwarderUri string, useBuildInMode bool, httpClientTimeout time.Duration, tmpDir string) string {

	cfg := apiConfig.Config{
		Logging: helpers.LoggingConfig{
			Level: LOGLEVEL,
		},
		PublicApiServer: apiConfig.ServerConfig{
			Port: publicApiPort,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "api.key"),
				CertFile:   filepath.Join(testCertDir, "api.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		BrokerServer: apiConfig.ServerConfig{
			Port: brokerPort,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "servicebroker.key"),
				CertFile:   filepath.Join(testCertDir, "servicebroker.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		DB: apiConfig.DBConfig{
			PolicyDB: db.DatabaseConfig{
				URL: dbURI,
			},
			BindingDB: db.DatabaseConfig{
				URL: dbURI,
			},
		},
		BrokerUsername:       brokerUserName,
		BrokerPassword:       brokerPassword,
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
		MetricsCollector: apiConfig.MetricsCollectorConfig{
			MetricsCollectorUrl: metricsCollectorUri,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricscollector.key"),
				CertFile:   filepath.Join(testCertDir, "metricscollector.crt"),
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
		CF: cf.CFConfig{
			API:      cfApi,
			ClientID: "admin",
			Secret:   "admin",
		},
		UseBuildInMode: useBuildInMode,
		InfoFilePath:   golangAPIInfoFilePath,
		MetricsForwarder: apiConfig.MetricsForwarderConfig{
			MetricsForwarderUrl: metricsForwarderUri,
		},
		RateLimit: models.RateLimitConfig{
			MaxAmount:     10,
			ValidDuration: 1 * time.Second,
		},
	}

	return writeYmlConfig(tmpDir, GolangAPIServer, &cfg)
}

func (components *Components) PrepareSchedulerConfig(dbUri string, scalingEngineUri string, tmpDir string, httpClientTimeout time.Duration) string {
	var (
		driverClassName string
		userName      string
		password      string
		jdbcDBUri     string
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
	}else {
		cfg, _ := mysql.ParseDSN(dbUri)
		scheme := "mysql"
		host := cfg.Addr
		path := cfg.DBName
		userName = cfg.User
		password = cfg.Passwd
		jdbcDBUri = fmt.Sprintf("jdbc:%s://%s/%s", scheme, host, path)
		driverClassName = "com.mysql.cj.jdbc.Driver"
	}
		settingStrTemplate := `
#datasource for application and quartz
spring.datasource.driverClassName=%s
spring.datasource.url=%s
spring.datasource.username=%s
spring.datasource.password=%s
#policy db
spring.policyDbDataSource.driverClassName=%s
spring.policyDbDataSource.url=%s
spring.policyDbDataSource.username=%s
spring.policyDbDataSource.password=%s
#quartz job
scalingenginejob.reschedule.interval.millisecond=10000
scalingenginejob.reschedule.maxcount=3
scalingengine.notification.reschedule.maxcount=3
# scaling engine url
autoscaler.scalingengine.url=%s
#ssl
server.ssl.key-store=%s/scheduler.p12
server.ssl.key-alias=scheduler
server.ssl.key-store-password=123456
server.ssl.key-store-type=PKCS12
server.ssl.trust-store=%s/autoscaler.truststore
server.ssl.trust-store-password=123456
client.ssl.key-store=%s/scheduler.p12
client.ssl.key-store-password=123456
client.ssl.key-store-type=PKCS12
client.ssl.trust-store=%s/autoscaler.truststore
client.ssl.trust-store-password=123456
client.ssl.protocol=TLSv1.2
server.ssl.enabled-protocols[3]=TLSv1,TLSv1.1,TLSv1.2
server.ssl.ciphers[23]=TLS_RSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_256_CBC_SHA256,TLS_RSA_WITH_AES_256_CBC_SHA,TLS_RSA_WITH_AES_128_GCM_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA,TLS_RSA_WITH_3DES_EDE_CBC_SHA,TLS_ECDHE_RSA_WITH_RC4_128_SHA,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384,TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,SSL_RSA_WITH_RC4_128_SHA

server.port=%d
scheduler.healthserver.port=0
client.httpClientTimeout=%d
#Quartz
org.quartz.scheduler.instanceName=app-autoscaler
org.quartz.scheduler.instanceId=0

spring.application.name=scheduler
spring.mvc.servlet.load-on-startup=1
spring.aop.auto=false
endpoints.enabled=false
spring.data.jpa.repositories.enabled=false
`
	settingJsonStr := fmt.Sprintf(settingStrTemplate, driverClassName, jdbcDBUri, userName, password, driverClassName, jdbcDBUri, userName, password, scalingEngineUri, testCertDir, testCertDir, testCertDir, testCertDir, components.Ports[Scheduler], components.Ports[Scheduler], int(httpClientTimeout/time.Second))
	cfgFile, err := os.Create(filepath.Join(tmpDir, "application.properties"))
	Expect(err).NotTo(HaveOccurred())
	ioutil.WriteFile(cfgFile.Name(), []byte(settingJsonStr), 0777)
	cfgFile.Close()
	return cfgFile.Name()
}

func (components *Components) PrepareMetricsCollectorConfig(dbURI string, port int, ccNOAAUAAURL string, collectInterval time.Duration,
	refreshInterval time.Duration, saveInterval time.Duration, collectMethod string, httpClientTimeout time.Duration, tmpDir string) string {
	cfg := mcConfig.Config{
		CF: cf.CFConfig{
			API:      ccNOAAUAAURL,
			ClientID: "admin",
			Secret:   "admin",
		},
		Server: mcConfig.ServerConfig{
			Port: port,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricscollector.key"),
				CertFile:   filepath.Join(testCertDir, "metricscollector.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
			NodeAddrs: []string{"localhost"},
			NodeIndex: 0,
		},
		Logging: helpers.LoggingConfig{
			Level: LOGLEVEL,
		},
		DB: mcConfig.DBConfig{
			InstanceMetricsDB: db.DatabaseConfig{
				URL: dbURI,
			},
			PolicyDB: db.DatabaseConfig{
				URL: dbURI,
			},
		},
		Collector: mcConfig.CollectorConfig{
			CollectInterval:       collectInterval,
			RefreshInterval:       refreshInterval,
			CollectMethod:         collectMethod,
			SaveInterval:          saveInterval,
			MetricCacheSizePerApp: 500,
			PersistMetrics:        true,
		},
		HttpClientTimeout: httpClientTimeout,
	}
	return writeYmlConfig(tmpDir, MetricsCollector, &cfg)
}

func (components *Components) PrepareEventGeneratorConfig(dbUri string, port int, metricsCollectorURL string, scalingEngineURL string, aggregatorExecuteInterval time.Duration,
	policyPollerInterval time.Duration, saveInterval time.Duration, evaluationManagerInterval time.Duration, httpClientTimeout time.Duration, tmpDir string) string {
	conf := &egConfig.Config{
		Logging: helpers.LoggingConfig{
			Level: LOGLEVEL,
		},
		Server: egConfig.ServerConfig{
			Port: port,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
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
		CF: cf.CFConfig{
			API:      ccUAAURL,
			ClientID: "admin",
			Secret:   "admin",
		},
		Server: seConfig.ServerConfig{
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
		CF: cf.CFConfig{
			API:      ccUAAURL,
			ClientID: "admin",
			Secret:   "admin",
		},
		InstanceMetricsDB: opConfig.InstanceMetricsDbPrunerConfig{
			RefreshInterval: 2 * time.Minute,
			CutoffDuration:  cutoffDuration,
			DB: db.DatabaseConfig{
				URL: dbURI,
			},
		},
		AppMetricsDB: opConfig.AppMetricsDBPrunerConfig{
			RefreshInterval: 2 * time.Minute,
			CutoffDuration:  cutoffDuration,
			DB: db.DatabaseConfig{
				URL: dbURI,
			},
		},
		ScalingEngineDB: opConfig.ScalingEngineDBPrunerConfig{
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

func (components *Components) PrepareMetricsGatewayConfig(dbURI string, metricServerAddresses []string, rlpAddr string, tmpDir string) string {
	cfg := mgConfig.Config{
		Logging: helpers.LoggingConfig{
			Level: LOGLEVEL,
		},
		EnvelopChanSize:   500,
		NozzleCount:       1,
		MetricServerAddrs: metricServerAddresses,
		AppManager: mgConfig.AppManagerConfig{
			AppRefreshInterval: 10 * time.Second,
			PolicyDB: db.DatabaseConfig{
				URL:                   dbURI,
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			},
		},
		Emitter: mgConfig.EmitterConfig{
			BufferSize:         500,
			KeepAliveInterval:  1 * time.Second,
			HandshakeTimeout:   1 * time.Second,
			MaxSetupRetryCount: 3,
			MaxCloseRetryCount: 3,
			RetryDelay:         500 * time.Millisecond,
			MetricsServerClientTLS: &models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricserver_client.key"),
				CertFile:   filepath.Join(testCertDir, "metricserver_client.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Nozzle: mgConfig.NozzleConfig{
			RLPAddr: rlpAddr,
			ShardID: "autoscaler",
			RLPClientTLS: &models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "reverselogproxy_client.key"),
				CertFile:   filepath.Join(testCertDir, "reverselogproxy_client.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
	}
	return writeYmlConfig(tmpDir, MetricsGateway, &cfg)
}

func (components *Components) PrepareMetricsServerConfig(dbURI string, httpClientTimeout time.Duration, httpServerPort int, wsServerPort int, tmpDir string) string {
	cfg := msConfig.Config{
		Logging: helpers.LoggingConfig{
			Level: LOGLEVEL,
		},
		HttpClientTimeout: httpClientTimeout,
		NodeAddrs:         []string{"localhost"},
		NodeIndex:         0,
		DB: msConfig.DBConfig{
			PolicyDB: db.DatabaseConfig{
				URL:                   dbURI,
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			},
			InstanceMetricsDB: db.DatabaseConfig{
				URL:                   dbURI,
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			},
		},
		Collector: msConfig.CollectorConfig{
			WSPort:          wsServerPort,
			WSKeepAliveTime: 5 * time.Second,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricserver.key"),
				CertFile:   filepath.Join(testCertDir, "metricserver.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
			RefreshInterval:        5 * time.Second,
			CollectInterval:        1 * time.Second,
			SaveInterval:           2 * time.Second,
			MetricCacheSizePerApp:  100,
			PersistMetrics:         true,
			EnvelopeProcessorCount: 2,
			EnvelopeChannelSize:    100,
			MetricChannelSize:      100,
		},
		Server: msConfig.ServerConfig{
			Port: httpServerPort,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricserver.key"),
				CertFile:   filepath.Join(testCertDir, "metricserver.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
	}
	return writeYmlConfig(tmpDir, MetricsServerHTTP, &cfg)
}

func writeYmlConfig(dir string, componentName string, c interface{}) string {
	cfgFile, err := ioutil.TempFile(dir, componentName)
	Expect(err).NotTo(HaveOccurred())
	defer cfgFile.Close()
	configBytes, err := yaml.Marshal(c)
	ioutil.WriteFile(cfgFile.Name(), configBytes, 0777)
	return cfgFile.Name()

}
