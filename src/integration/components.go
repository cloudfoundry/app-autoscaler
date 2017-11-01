package integration

import (
	"autoscaler/cf"
	egConfig "autoscaler/eventgenerator/config"
	mcConfig "autoscaler/metricscollector/config"
	"autoscaler/models"
	seConfig "autoscaler/scalingengine/config"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit/ginkgomon"
	"gopkg.in/yaml.v2"
)

const (
	APIServer        = "apiServer"
	APIPublicServer  = "APIPublicServer"
	ServiceBroker    = "serviceBroker"
	Scheduler        = "scheduler"
	MetricsCollector = "metricsCollector"
	EventGenerator   = "eventGenerator"
	ScalingEngine    = "scalingEngine"
	ConsulCluster    = "consulCluster"
)

var testCertDir string = "../../test-certs"

var serviceCatalogPath string = "../../servicebroker/config/catalog.json"

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
type APIServerClient struct {
	Uri string          `json:"uri"`
	TLS models.TLSCerts `json:"tls"`
}

type ServiceBrokerConfig struct {
	Port int `json:"port"`

	Username string `json:"username"`
	Password string `json:"password"`

	DB DBConfig `json:"db"`

	APIServerClient    APIServerClient `json:"apiserver"`
	HttpRequestTimeout int             `json:"httpRequestTimeout"`
	TLS                models.TLSCerts `json:"tls"`
	ServiceCatalogPath string          `json:"serviceCatalogPath"`
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
type APIServerConfig struct {
	Port       int      `json:"port"`
	PublicPort int      `json:"publicPort"`
	CFAPI      string   `json:"cfApi"`
	DB         DBConfig `json:"db"`

	SchedulerClient        SchedulerClient        `json:"scheduler"`
	ScalingEngineClient    ScalingEngineClient    `json:"scalingEngine"`
	MetricsCollectorClient MetricsCollectorClient `json:"metricsCollector"`

	TLS       models.TLSCerts `json:"tls"`
	PublicTLS models.TLSCerts `json:"publicTls"`
}

func (components *Components) ServiceBroker(confPath string, argv ...string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:              ServiceBroker,
		AnsiColorCode:     "32m",
		StartCheck:        "Service broker app is running",
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			"node", append([]string{components.Executables[ServiceBroker], "-c", confPath}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}

func (components *Components) ApiServer(confPath string, argv ...string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:              APIServer,
		AnsiColorCode:     "33m",
		StartCheck:        "Autoscaler API server started",
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			"node", append([]string{components.Executables[APIServer], "-c", confPath}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}

func (components *Components) Scheduler(confPath string, argv ...string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:              Scheduler,
		AnsiColorCode:     "34m",
		StartCheck:        "Scheduler is ready to start",
		StartCheckTimeout: 120 * time.Second,
		Command: exec.Command(
			"java", append([]string{"-jar", "-Dspring.config.location=" + confPath, "-Dserver.port=" + strconv.FormatInt(int64(components.Ports[Scheduler]), 10), components.Executables[Scheduler]}, argv...)...,
		),
		Cleanup: func() {
		},
	})
}

func (components *Components) MetricsCollector(confPath string, argv ...string) *ginkgomon.Runner {

	return ginkgomon.New(ginkgomon.Config{
		Name:              MetricsCollector,
		AnsiColorCode:     "35m",
		StartCheck:        `"metricscollector.started"`,
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			components.Executables[MetricsCollector],
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
		StartCheckTimeout: 10 * time.Second,
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
		AnsiColorCode:     "37m",
		StartCheck:        `"scalingengine.started"`,
		StartCheckTimeout: 10 * time.Second,
		Command: exec.Command(
			components.Executables[ScalingEngine],
			append([]string{
				"-c", confPath,
			}, argv...)...,
		),
	})
}

func (components *Components) PrepareServiceBrokerConfig(port int, username string, password string, dbUri string, apiServerUri string, brokerApiHttpRequestTimeout time.Duration, tmpDir string) string {
	brokerConfig := ServiceBrokerConfig{
		Port:     port,
		Username: username,
		Password: password,
		DB: DBConfig{
			URI:            dbUri,
			MinConnections: 1,
			MaxConnections: 10,
			IdleTimeout:    1000,
		},
		APIServerClient: APIServerClient{
			Uri: apiServerUri,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "api.key"),
				CertFile:   filepath.Join(testCertDir, "api.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		HttpRequestTimeout: int(brokerApiHttpRequestTimeout / time.Millisecond),
		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "servicebroker.key"),
			CertFile:   filepath.Join(testCertDir, "servicebroker.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
		ServiceCatalogPath: serviceCatalogPath,
	}

	cfgFile, err := ioutil.TempFile(tmpDir, ServiceBroker)
	w := json.NewEncoder(cfgFile)
	err = w.Encode(brokerConfig)
	Expect(err).NotTo(HaveOccurred())
	cfgFile.Close()
	return cfgFile.Name()
}

func (components *Components) PrepareApiServerConfig(port int, publicPort int, cfApi string, dbUri string, schedulerUri string, scalingEngineUri string, metricsCollectorUri string, tmpDir string) string {
	apiConfig := APIServerConfig{
		Port:       port,
		PublicPort: publicPort,
		CFAPI:      cfApi,
		DB: DBConfig{
			URI:            dbUri,
			MinConnections: 1,
			MaxConnections: 10,
			IdleTimeout:    1000,
		},

		SchedulerClient: SchedulerClient{
			Uri: schedulerUri,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scheduler.key"),
				CertFile:   filepath.Join(testCertDir, "scheduler.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		ScalingEngineClient: ScalingEngineClient{
			Uri: scalingEngineUri,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
				CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		MetricsCollectorClient: MetricsCollectorClient{
			Uri: metricsCollectorUri,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricscollector.key"),
				CertFile:   filepath.Join(testCertDir, "metricscollector.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},

		TLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "api.key"),
			CertFile:   filepath.Join(testCertDir, "api.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},

		PublicTLS: models.TLSCerts{
			KeyFile:    filepath.Join(testCertDir, "api_public.key"),
			CertFile:   filepath.Join(testCertDir, "api_public.crt"),
			CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
		},
	}

	cfgFile, err := ioutil.TempFile(tmpDir, APIServer)
	w := json.NewEncoder(cfgFile)
	err = w.Encode(apiConfig)
	Expect(err).NotTo(HaveOccurred())
	cfgFile.Close()
	return cfgFile.Name()
}

func (components *Components) PrepareSchedulerConfig(dbUri string, scalingEngineUri string, tmpDir string, consulPort string) string {
	dbUrl, _ := url.Parse(dbUri)
	scheme := dbUrl.Scheme
	host := dbUrl.Host
	path := dbUrl.Path
	userInfo := dbUrl.User
	userName := userInfo.Username()
	password, _ := userInfo.Password()
	if scheme == "postgres" {
		scheme = "postgresql"
	}
	jdbcDBUri := fmt.Sprintf("jdbc:%s://%s%s", scheme, host, path)
	settingStrTemplate := `
#datasource for application and quartz
spring.datasource.driverClassName=org.postgresql.Driver
spring.datasource.url=%s
spring.datasource.username=%s
spring.datasource.password=%s
#policy db
spring.policyDbDataSource.driverClassName=org.postgresql.Driver
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
#Quartz
org.quartz.scheduler.instanceName=app-autoscaler-%d
org.quartz.scheduler.instanceId=app-autoscaler-%d
#consul
spring.cloud.consul.port=%s
spring.cloud.consul.discovery.serviceName=scheduler
spring.cloud.consul.discovery.instanceId=scheduler
spring.cloud.consul.discovery.heartbeat.enabled=true
spring.cloud.consul.discovery.heartbeat.ttlValue=20
spring.cloud.consul.discovery.hostname=

spring.application.name=scheduler
spring.mvc.servlet.load-on-startup=1
spring.aop.auto=false
endpoints.enabled=false
spring.data.jpa.repositories.enabled=false
`
	settingJsonStr := fmt.Sprintf(settingStrTemplate, jdbcDBUri, userName, password, jdbcDBUri, userName, password, scalingEngineUri, testCertDir, testCertDir, testCertDir, testCertDir, components.Ports[Scheduler], components.Ports[Scheduler], consulPort)
	cfgFile, err := os.Create(filepath.Join(tmpDir, "application.properties"))
	Expect(err).NotTo(HaveOccurred())
	ioutil.WriteFile(cfgFile.Name(), []byte(settingJsonStr), 0777)
	cfgFile.Close()
	return cfgFile.Name()
}

func (components *Components) PrepareMetricsCollectorConfig(dbUri string, port int, enableDBLock bool, ccNOAAUAAUrl string, cfGrantTypePassword string, collectInterval time.Duration,
	refreshInterval time.Duration, saveInterval time.Duration, collectMethod string, tmpDir string, lockTTL time.Duration, lockRetryInterval time.Duration, ConsulClusterConfig string) string {
	cfg := mcConfig.Config{
		Cf: cf.CfConfig{
			Api:       ccNOAAUAAUrl,
			GrantType: cfGrantTypePassword,
			Username:  "admin",
			Password:  "admin",
		},
		Server: mcConfig.ServerConfig{
			Port: port,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "metricscollector.key"),
				CertFile:   filepath.Join(testCertDir, "metricscollector.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Logging: mcConfig.LoggingConfig{
			Level: "debug",
		},
		Db: mcConfig.DbConfig{
			InstanceMetricsDbUrl: dbUri,
			PolicyDbUrl:          dbUri,
		},
		Collector: mcConfig.CollectorConfig{
			CollectInterval: collectInterval,
			RefreshInterval: refreshInterval,
			CollectMethod:   collectMethod,
			SaveInterval:    saveInterval,
		},
		Lock: mcConfig.LockConfig{
			LockTTL:             lockTTL,
			LockRetryInterval:   lockRetryInterval,
			ConsulClusterConfig: ConsulClusterConfig,
		},
		EnableDBLock: enableDBLock,
		DBLock: mcConfig.DBLockConfig{
			LockTTL:           time.Duration(10 * time.Second),
			LockDBURL:         dbUri,
			LockRetryInterval: time.Duration(2 * time.Second),
		},
	}

	return writeYmlConfig(tmpDir, MetricsCollector, &cfg)
}

func (components *Components) PrepareEventGeneratorConfig(dbUri string, metricsCollectorUrl string, scalingEngineUrl string, aggregatorExecuteInterval time.Duration, policyPollerInterval time.Duration,
	evaluationManagerInterval time.Duration, tmpDir string, lockTTL time.Duration, lockRetryInterval time.Duration, ConsulClusterConfig string) string {
	conf := &egConfig.Config{
		Logging: egConfig.LoggingConfig{
			Level: "debug",
		},
		Aggregator: egConfig.AggregatorConfig{
			AggregatorExecuteInterval: aggregatorExecuteInterval,
			PolicyPollerInterval:      policyPollerInterval,
			MetricPollerCount:         1,
			AppMonitorChannelSize:     1,
		},
		Evaluator: egConfig.EvaluatorConfig{
			EvaluationManagerInterval: evaluationManagerInterval,
			EvaluatorCount:            1,
			TriggerArrayChannelSize:   1,
		},
		DB: egConfig.DBConfig{
			PolicyDBUrl:    dbUri,
			AppMetricDBUrl: dbUri,
		},
		ScalingEngine: egConfig.ScalingEngineConfig{
			ScalingEngineUrl: scalingEngineUrl,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		MetricCollector: egConfig.MetricCollectorConfig{
			MetricCollectorUrl: metricsCollectorUrl,
			TLSClientCerts: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "eventgenerator.key"),
				CertFile:   filepath.Join(testCertDir, "eventgenerator.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Lock: egConfig.LockConfig{
			LockTTL:             lockTTL,
			LockRetryInterval:   lockRetryInterval,
			ConsulClusterConfig: ConsulClusterConfig,
		},
		DefaultBreachDurationSecs: 600,
		DefaultStatWindowSecs:     300,
	}
	return writeYmlConfig(tmpDir, EventGenerator, &conf)
}

func (components *Components) PrepareScalingEngineConfig(dbUri string, port int, ccUAAUrl string, cfGrantTypePassword string, tmpDir string, consulClusterConfig string) string {
	conf := seConfig.Config{
		Cf: cf.CfConfig{
			Api:       ccUAAUrl,
			GrantType: cfGrantTypePassword,
			Username:  "admin",
			Password:  "admin",
		},
		Server: seConfig.ServerConfig{
			Port: port,
			TLS: models.TLSCerts{
				KeyFile:    filepath.Join(testCertDir, "scalingengine.key"),
				CertFile:   filepath.Join(testCertDir, "scalingengine.crt"),
				CACertFile: filepath.Join(testCertDir, "autoscaler-ca.crt"),
			},
		},
		Logging: seConfig.LoggingConfig{
			Level: "debug",
		},
		Db: seConfig.DbConfig{
			PolicyDbUrl:        dbUri,
			ScalingEngineDbUrl: dbUri,
			SchedulerDbUrl:     dbUri,
		},
		Synchronizer: seConfig.SynchronizerConfig{
			ActiveScheduleSyncInterval: 10 * time.Second,
		},
		Consul: seConfig.ConsulConfig{
			Cluster: consulClusterConfig,
		},
		DefaultCoolDownSecs: 300,
	}

	return writeYmlConfig(tmpDir, ScalingEngine, &conf)
}

func writeYmlConfig(dir string, componentName string, c interface{}) string {
	cfgFile, err := ioutil.TempFile(dir, componentName)
	Expect(err).NotTo(HaveOccurred())
	defer cfgFile.Close()
	configBytes, err := yaml.Marshal(c)
	ioutil.WriteFile(cfgFile.Name(), configBytes, 0777)
	return cfgFile.Name()

}
