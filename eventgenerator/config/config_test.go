package config_test

import (
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var (
		conf                        *Config
		err                         error
		configBytes                 []byte
		configFile                  string
		mockVCAPConfigurationReader *fakes.FakeVCAPConfigurationReader
		expectedDbConfig            map[string]db.DatabaseConfig
	)

	BeforeEach(func() {
		mockVCAPConfigurationReader = &fakes.FakeVCAPConfigurationReader{}
		expectedDbConfig = map[string]db.DatabaseConfig{
			"policy_db": {
				URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			},
			"appmetrics_db": {
				URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			},
		}
	})

	Describe("LoadConfig", func() {
		When("config is read from env", func() {
			var expectedTLSConfig = models.TLSCerts{
				KeyFile:    "some/path/in/container/cfcert.key",
				CertFile:   "some/path/in/container/cfcert.crt",
				CACertFile: "some/path/in/container/cfcert.crt",
			}
			var expectedDbUrl = "postgres://foo:bar@postgres.example.com:5432/policy_db?sslcert=%2Ftmp%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fserver_ca.sslrootcert" // #nosec G101

			BeforeEach(func() {
				mockVCAPConfigurationReader.GetPortReturns(3333)
				mockVCAPConfigurationReader.GetInstanceTLSCertsReturns(expectedTLSConfig)
				mockVCAPConfigurationReader.GetInstanceIndexReturns(3)
				mockVCAPConfigurationReader.IsRunningOnCFReturns(true)
				mockVCAPConfigurationReader.GetSpaceGuidReturns("some-space-id")
				mockVCAPConfigurationReader.GetOrgGuidReturns("some-org-id")
				mockVCAPConfigurationReader.MaterializeDBFromServiceReturns(expectedDbUrl, nil)
			})

			JustBeforeEach(func() {
				conf, err = LoadConfig("", mockVCAPConfigurationReader)
			})

			It("should set logging to plain sink", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Logging.PlainTextSink).To(BeTrue())
			})

			It("sets env variable over config file", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.CFServer.Port).To(Equal(3333))
				Expect(conf.Server.Port).To(Equal(0))
			})

			It("send certs to scalingengineScalingEngine TlSClientCert", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.ScalingEngine.TLSClientCerts).To(Equal(expectedTLSConfig))
			})

			It("sets Pool.InstanceIndex with vcap instance index", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Pool.InstanceIndex).To(Equal(3))
			})

			It("sets xfcc space and org guid", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.CFServer.XFCC.ValidOrgGuid).To(Equal("some-org-id"))
				Expect(conf.CFServer.XFCC.ValidSpaceGuid).To(Equal("some-space-id"))
			})

			When("handling available databases", func() {
				It("calls vcapReader ConfigureDatabases with the right arguments", func() {
					testhelpers.ExpectConfigureDatabasesCalledOnce(err, mockVCAPConfigurationReader, "")
				})
			})

			When("service is empty", func() {
				BeforeEach(func() {
					mockVCAPConfigurationReader.GetServiceCredentialContentReturns([]byte(""), fmt.Errorf("not found"))
				})

				It("should error with config service not found", func() {
					Expect(errors.Is(err, configutil.ErrServiceConfigNotFound)).To(BeTrue())
				})
			})
		})

		When("config is read from file", func() {
			JustBeforeEach(func() {
				configFile = testhelpers.BytesToFile(configBytes)
				conf, err = LoadConfig(configFile, mockVCAPConfigurationReader)
			})

			Context("with valid yaml", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
http_client_timeout: 10s
server:
  port: 9080
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
pool:
  total_instances: 2
  instance_index: 1
cf_server:
  port: 9082
health:
  server_config:
    port: 9999
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  save_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
  app_metric_channel_size: 100
  metric_cache_size_per_app: 500
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metricCollector:
  metric_collector_url: log-cache:1234
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
circuitBreaker:
  back_off_initial_interval: 10s
  back_off_max_interval: 60m
  consecutive_failure_count: 5
`)
				})

				It("returns the config", func() {
					expectedTime := 10 * time.Second
					Expect(err).NotTo(HaveOccurred())
					Expect(conf).To(Equal(&Config{
						BaseConfig: configutil.BaseConfig{
							Logging: helpers.LoggingConfig{Level: "info"},
							Server: helpers.ServerConfig{
								Port: 9080,
								TLS: models.TLSCerts{
									KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/server.key",
									CertFile:   "/var/vcap/jobs/autoscaler/config/certs/server.crt",
									CACertFile: "/var/vcap/jobs/autoscaler/config/certs/ca.crt",
								},
							},
							CFServer: helpers.ServerConfig{
								Port: 9082,
							},
							Health: helpers.HealthConfig{
								ServerConfig: helpers.ServerConfig{
									Port: 9999,
								},
							},
							Db: expectedDbConfig,
						},
						HttpClientTimeout: &expectedTime,
						Pool: &PoolConfig{
							InstanceIndex:  1,
							TotalInstances: 2,
						},
						Aggregator: &AggregatorConfig{
							AggregatorExecuteInterval: 30 * time.Second,
							PolicyPollerInterval:      30 * time.Second,
							SaveInterval:              30 * time.Second,
							MetricPollerCount:         10,
							AppMonitorChannelSize:     100,
							AppMetricChannelSize:      100,
							MetricCacheSizePerApp:     500,
						},
						Evaluator: &EvaluatorConfig{
							EvaluationManagerInterval: 30 * time.Second,
							EvaluatorCount:            10,
							TriggerArrayChannelSize:   100},
						ScalingEngine: ScalingEngineConfig{
							ScalingEngineURL: "http://localhost:8082",
							TLSClientCerts: models.TLSCerts{
								KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/se.key",
								CertFile:   "/var/vcap/jobs/autoscaler/config/certs/se.crt",
								CACertFile: "/var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt",
							},
						},
						MetricCollector: MetricCollectorConfig{
							MetricCollectorURL: "log-cache:1234",
							TLSClientCerts: models.TLSCerts{
								KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/mc.key",
								CertFile:   "/var/vcap/jobs/autoscaler/config/certs/mc.crt",
								CACertFile: "/var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt",
							},
						},
						DefaultBreachDurationSecs: 600,
						DefaultStatWindowSecs:     300,
						CircuitBreaker: &CircuitBreakerConfig{
							BackOffInitialInterval:  10 * time.Second,
							BackOffMaxInterval:      1 * time.Hour,
							ConsecutiveFailureCount: 5,
						},
					}))
				})
			})
			Context("with invalid yaml", func() {
				BeforeEach(func() {
					configBytes = []byte(`
  logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  save_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
  app_metric_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("returns an error", func() {
					Expect(err).To(MatchError(MatchRegexp(".*field level not found in type config.Config*")))
				})
			})
			Context("with partial config", func() {
				BeforeEach(func() {
					configBytes = []byte(`
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("returns default values", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.Aggregator.PolicyPollerInterval).To(Equal(DefaultPolicyPollerInterval))

					Expect(err).NotTo(HaveOccurred())
					Expect(conf.Server.Port).To(Equal(8080))
					Expect(conf.Logging.Level).To(Equal("info"))

					expectedTimeout := DefaultHttpClientTimeout
					// Check individual fields instead of entire struct to avoid map ordering issues
					Expect(conf.Logging).To(Equal(helpers.LoggingConfig{Level: "info"}))
					Expect(conf.HttpClientTimeout).To(Equal(&expectedTimeout))
					Expect(conf.Server).To(Equal(helpers.ServerConfig{
						Port: 8080,
						TLS:  models.TLSCerts{},
					}))
					Expect(conf.Pool).To(Equal(&PoolConfig{}))
					Expect(conf.CFServer).To(Equal(helpers.ServerConfig{
						Port: 8082,
					}))
					Expect(conf.Health).To(Equal(helpers.HealthConfig{
						ServerConfig: helpers.ServerConfig{
							Port: 8081,
						},
					}))

					// Check database configs individually to avoid map ordering issues
					Expect(conf.Db).To(HaveKey("policy_db"))
					Expect(conf.Db).To(HaveKey("appmetrics_db"))
					Expect(conf.Db["policy_db"]).To(Equal(db.DatabaseConfig{
						URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
					Expect(conf.Db["appmetrics_db"]).To(Equal(db.DatabaseConfig{
						URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))

					Expect(conf.Aggregator).To(Equal(&AggregatorConfig{
						AggregatorExecuteInterval: DefaultAggregatorExecuteInterval,
						PolicyPollerInterval:      DefaultPolicyPollerInterval,
						MetricPollerCount:         DefaultMetricPollerCount,
						AppMonitorChannelSize:     DefaultAppMonitorChannelSize,
						AppMetricChannelSize:      DefaultAppMetricChannelSize,
						SaveInterval:              DefaultSaveInterval,
						MetricCacheSizePerApp:     DefaultMetricCacheSizePerApp,
					}))
					Expect(conf.Evaluator).To(Equal(&EvaluatorConfig{
						EvaluationManagerInterval: DefaultEvaluationExecuteInterval,
						EvaluatorCount:            DefaultEvaluatorCount,
						TriggerArrayChannelSize:   DefaultTriggerArrayChannelSize,
					}))
					Expect(conf.ScalingEngine).To(Equal(ScalingEngineConfig{
						ScalingEngineURL: "http://localhost:8082",
					}))
					Expect(conf.MetricCollector).To(Equal(MetricCollectorConfig{
						MetricCollectorURL: "log-cache:1234",
					}))
					Expect(conf.DefaultStatWindowSecs).To(Equal(300))
					Expect(conf.DefaultBreachDurationSecs).To(Equal(600))
					Expect(conf.CircuitBreaker).To(Equal(&CircuitBreakerConfig{
						BackOffInitialInterval:  DefaultBackOffInitialInterval,
						BackOffMaxInterval:      DefaultBackOffMaxInterval,
						ConsecutiveFailureCount: DefaultBreakerConsecutiveFailureCount,
					}))
				})
			})
			Context("when http_client_timeout is not a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
http_client_timeout: 10k
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 60s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
				})
			})

			Context("when it gives a non integer max_open_connections of policydb", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 60s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})

			Context("when it gives a non integer max_idle_connections of policydb", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 60s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})

			Context("when connection_max_lifetime of policydb is not a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6k
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 60s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
				})
			})

			Context("when it gives a non integer max_open_connections of app_metrics_db", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 60s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})

			Context("when it gives a non integer max_idle_connections of app_metrics_db", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 60s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})

			Context("when connection_max_lifetime of app_metrics_db is not a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6k
aggregator:
  aggregator_execute_interval: 60s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
				})
			})

			Context("when aggregator_execute_interval is not a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 5k
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
				})
			})

			Context("when policy_poller_interval is not  a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 7u
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
				})
			})

			Context("when save_interval is not  a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  save_interval: 7u
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
				})
			})

			Context("when it gives a non integer metric_poller_count", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: NOT-INTEGER-VALUE
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})

			Context("when it gives a non integer app_monitor_channel_size", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: NOT-INTEGER-VALUE
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})

			Context("when it gives a non integer app_metric_channel_size", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db_url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
  app_metrics_db_url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 10
  app_metric_channel_size: NOT-INTEGER-VALUE
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})

			Context("when it gives a non integer metric_cache_size_per_app", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db_url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
  app_metrics_db_url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 10
  app_metric_channel_size: 10
  metric_cache_size_per_app: NOT-INTEGER-VALUE
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})
			Context("when it gives a non integer evaluation_manager_execute_interval", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: NOT-INTEGER-VALUE
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
				})
			})

			Context("when it gives a non integer evaluator_count", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: "NOT-INTEGER-VALUE"
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})
			Context("when it gives a non integer trigger_array_channel_size", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: NOT-INTEGER-VALUE
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
				})
			})

			Context("when it gives a non integer defaultStatWindowSecs", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: NOT-INTEGER-VALUE
defaultBreachDurationSecs: 600
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal !!str `NOT-INT...` into int")))
				})
			})
			Context("when it gives a non integer defaultBreachDurationSecs", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: NOT-INTEGER-VALUE
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal !!str `NOT-INT...` into int")))
				})
			})

			Context("when it gives a non integer health port", func() {
				BeforeEach(func() {
					configBytes = []byte(`
logging:
  level: info
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  appmetrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
aggregator:
  aggregator_execute_interval: 30s
  policy_poller_interval: 30s
  metric_poller_count: 10
  app_monitor_channel_size: 100
evaluator:
  evaluation_manager_execute_interval: 30s
  evaluator_count: 10
  trigger_array_channel_size: 100
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: log-cache:1234
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 300
health:
  server_config:
    port: NOT-INTEGER-VALUE
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal !!str `NOT-INT...` into int")))
				})
			})

		})

		Describe("Validate", func() {
			BeforeEach(func() {
				expectedTimeout := 10 * time.Second
				conf = &Config{
					BaseConfig: configutil.BaseConfig{
						Logging: helpers.LoggingConfig{Level: "info"},
						Db:      expectedDbConfig,
					},
					Pool: &PoolConfig{
						TotalInstances: 2,
						InstanceIndex:  0,
					},
					Aggregator: &AggregatorConfig{
						AggregatorExecuteInterval: 30 * time.Second,
						PolicyPollerInterval:      30 * time.Second,
						SaveInterval:              30 * time.Second,
						MetricPollerCount:         10,
						AppMonitorChannelSize:     100,
						AppMetricChannelSize:      100,
						MetricCacheSizePerApp:     500,
					},
					Evaluator: &EvaluatorConfig{
						EvaluationManagerInterval: 30 * time.Second,
						EvaluatorCount:            10,
						TriggerArrayChannelSize:   100},
					ScalingEngine: ScalingEngineConfig{
						ScalingEngineURL: "http://localhost:8082"},
					MetricCollector: MetricCollectorConfig{
						MetricCollectorURL: "log-cache:1234",
					},
					DefaultBreachDurationSecs: 600,
					DefaultStatWindowSecs:     300,
					HttpClientTimeout:         &expectedTimeout,
				}
			})

			JustBeforeEach(func() {
				err = conf.Validate()
			})

			Context("when policy db url is not set", func() {
				BeforeEach(func() {
					conf.Db[db.PolicyDb] = db.DatabaseConfig{}
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: db.policy_db.url is empty"))
				})
			})

			Context("when appmetric db url is not set", func() {
				BeforeEach(func() {
					conf.Db[db.AppMetricsDb] = db.DatabaseConfig{}
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: db.app_metrics_db.url is empty"))
				})
			})

			Context("when scaling engine url is not set", func() {

				BeforeEach(func() {
					conf.ScalingEngine.ScalingEngineURL = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: scalingEngine.scaling_engine_url is empty"))
				})
			})
			Context("when metric collector url is not set", func() {

				BeforeEach(func() {
					conf.MetricCollector.MetricCollectorURL = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: metricCollector.metric_collector_url is empty"))
				})
			})

			Context("when AggregatorExecuateInterval <= 0", func() {
				BeforeEach(func() {
					conf.Aggregator.AggregatorExecuteInterval = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: aggregator.aggregator_execute_interval is less-equal than 0"))
				})
			})

			Context("when PolicyPollerInterval is <= 0", func() {
				BeforeEach(func() {
					conf.Aggregator.PolicyPollerInterval = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: aggregator.policy_poller_interval is less-equal than 0"))
				})
			})

			Context("when SaveInterval <= 0", func() {
				BeforeEach(func() {
					conf.Aggregator.SaveInterval = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: aggregator.save_interval is less-equal than 0"))
				})
			})

			Context("when MetricPollerCount <= 0", func() {
				BeforeEach(func() {
					conf.Aggregator.MetricPollerCount = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: aggregator.metric_poller_count is less-equal than 0"))
				})
			})

			Context("when AppMonitorChannelSize <= 0", func() {
				BeforeEach(func() {
					conf.Aggregator.AppMonitorChannelSize = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: aggregator.app_monitor_channel_size is less-equal than 0"))
				})
			})

			Context("when AppMetricChannelSize <= 0", func() {
				BeforeEach(func() {
					conf.Aggregator.AppMetricChannelSize = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: aggregator.app_metric_channel_size is less-equal than 0"))
				})
			})

			Context("when MetricCacheSizePerApp <= 0", func() {
				BeforeEach(func() {
					conf.Aggregator.MetricCacheSizePerApp = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: aggregator.metric_cache_size_per_app is less-equal than 0"))
				})
			})

			Context("when EvaluationManagerInterval <= 0", func() {
				BeforeEach(func() {
					conf.Evaluator.EvaluationManagerInterval = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: evaluator.evaluation_manager_execute_interval is less-equal than 0"))
				})
			})

			Context("when EvaluatorCount <= 0", func() {
				BeforeEach(func() {
					conf.Evaluator.EvaluatorCount = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: evaluator.evaluator_count is less-equal than 0"))
				})
			})

			Context("when TriggerArrayChannelSize <= 0", func() {
				BeforeEach(func() {
					conf.Evaluator.TriggerArrayChannelSize = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: evaluator.trigger_array_channel_size is less-equal than 0"))
				})
			})

			Context("when DefaultBreachDurationSecs < 60", func() {
				BeforeEach(func() {
					conf.DefaultBreachDurationSecs = 10
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: defaultBreachDurationSecs should be between 60 and 3600"))
				})
			})

			Context("when DefaultStatWindowSecs < 60", func() {
				BeforeEach(func() {
					conf.DefaultStatWindowSecs = 10
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: defaultStatWindowSecs should be between 60 and 3600"))
				})
			})

			Context("when DefaultBreachDurationSecs > 3600", func() {
				BeforeEach(func() {
					conf.DefaultBreachDurationSecs = 5000
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: defaultBreachDurationSecs should be between 60 and 3600"))
				})
			})

			Context("when DefaultStatWindowSecs > 3600", func() {
				BeforeEach(func() {
					conf.DefaultStatWindowSecs = 5000
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: defaultStatWindowSecs should be between 60 and 3600"))
				})
			})

			Context("when node index is out of range", func() {
				Context("when node index is negative", func() {
					BeforeEach(func() {
						conf.Pool.InstanceIndex = -1
					})
					It("should error", func() {
						Expect(err).To(MatchError("Configuration error: pool.instance_index out of range"))
					})
				})

				Context("when node index is >= number of nodes", func() {
					BeforeEach(func() {
						conf.Pool.InstanceIndex = 2
						conf.Pool.TotalInstances = 2
					})
					It("should error", func() {
						Expect(err).To(MatchError("Configuration error: pool.instance_index out of range"))
					})
				})

			})

			Context("when HttpClientTimeout is <= 0", func() {
				BeforeEach(func() {
					expectedTimeout := 0 * time.Second
					conf.HttpClientTimeout = &expectedTimeout
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: http_client_timeout is less-equal than 0"))
				})
			})
		})
	})

	Describe("LoadVcapConfig", func() {
		var (
			conf *Config
			err  error
		)

		BeforeEach(func() {
			conf = &Config{
				BaseConfig: configutil.BaseConfig{
					Db: make(map[string]db.DatabaseConfig),
				},
			}
			mockVCAPConfigurationReader.IsRunningOnCFReturns(true)
			mockVCAPConfigurationReader.GetPortReturns(8080)
			mockVCAPConfigurationReader.GetSpaceGuidReturns("space-guid")
			mockVCAPConfigurationReader.GetOrgGuidReturns("org-guid")
			mockVCAPConfigurationReader.GetServiceCredentialContentReturns([]byte(`{"key": "value"}`), nil)
			mockVCAPConfigurationReader.ConfigureDatabasesReturns(nil)
			mockVCAPConfigurationReader.GetInstanceIndexReturns(0)
			mockVCAPConfigurationReader.GetInstanceTLSCertsReturns(models.TLSCerts{})
		})

		It("should apply common VCAP configuration when running on CF", func() {
			err = LoadVcapConfig(conf, mockVCAPConfigurationReader)
			Expect(err).NotTo(HaveOccurred())

			// Verify that various interface methods were called
			Expect(mockVCAPConfigurationReader.IsRunningOnCFCallCount()).To(Equal(1))
			Expect(mockVCAPConfigurationReader.GetPortCallCount()).To(Equal(1))
			Expect(mockVCAPConfigurationReader.GetServiceCredentialContentCallCount()).To(Equal(1))
			Expect(mockVCAPConfigurationReader.ConfigureDatabasesCallCount()).To(Equal(1))
			Expect(mockVCAPConfigurationReader.GetSpaceGuidCallCount()).To(Equal(1))
			Expect(mockVCAPConfigurationReader.GetOrgGuidCallCount()).To(Equal(1))

			// Verify service name passed to GetServiceCredentialContent
			serviceName, credentialKey := mockVCAPConfigurationReader.GetServiceCredentialContentArgsForCall(0)
			Expect(serviceName).To(Equal("eventgenerator-config"))
			Expect(credentialKey).To(Equal("eventgenerator-config"))

			// Verify common configuration was applied
			Expect(conf.Logging.PlainTextSink).To(BeTrue())
			Expect(conf.CFServer.Port).To(Equal(8080))
			Expect(conf.Server.Port).To(Equal(0))
			Expect(conf.CFServer.XFCC.ValidSpaceGuid).To(Equal("space-guid"))
			Expect(conf.CFServer.XFCC.ValidOrgGuid).To(Equal("org-guid"))
		})

		When("not running on CF", func() {
			BeforeEach(func() {
				mockVCAPConfigurationReader.IsRunningOnCFReturns(false)
			})

			It("should not apply VCAP configuration", func() {
				err = LoadVcapConfig(conf, mockVCAPConfigurationReader)
				Expect(err).NotTo(HaveOccurred())

				Expect(mockVCAPConfigurationReader.IsRunningOnCFCallCount()).To(Equal(1))
				// Other methods should not be called when not on CF
				Expect(mockVCAPConfigurationReader.GetPortCallCount()).To(Equal(0))
				Expect(mockVCAPConfigurationReader.GetServiceCredentialContentCallCount()).To(Equal(0))
			})
		})

		When("GetServiceCredentialContent returns an error", func() {
			var expectedError error

			BeforeEach(func() {
				expectedError = errors.New("service credential error")
				mockVCAPConfigurationReader.GetServiceCredentialContentReturns(nil, expectedError)
			})

			It("should return the error", func() {
				err = LoadVcapConfig(conf, mockVCAPConfigurationReader)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, configutil.ErrServiceConfigNotFound)).To(BeTrue())
			})
		})
	})
})
