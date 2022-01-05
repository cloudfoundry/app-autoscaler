package config_test

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/eventgenerator/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("Config", func() {

	var (
		conf        *Config
		err         error
		configBytes []byte
	)

	Describe("LoadConfig", func() {

		JustBeforeEach(func() {
			conf, err = LoadConfig(configBytes)
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
  node_addrs: [address1, address2]
  node_index: 1
health:
  port: 9999
db:
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
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
				Expect(err).NotTo(HaveOccurred())
				Expect(conf).To(Equal(&Config{
					Logging:           helpers.LoggingConfig{Level: "info"},
					HttpClientTimeout: 10 * time.Second,
					Server: ServerConfig{
						Port: 9080,
						TLS: models.TLSCerts{
							KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/server.key",
							CertFile:   "/var/vcap/jobs/autoscaler/config/certs/server.crt",
							CACertFile: "/var/vcap/jobs/autoscaler/config/certs/ca.crt",
						},
						NodeAddrs: []string{"address1", "address2"},
						NodeIndex: 1,
					},
					Health: models.HealthConfig{
						Port: 9999,
					},
					DB: DBConfig{
						PolicyDB: db.DatabaseConfig{
							URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    10,
							MaxIdleConnections:    5,
							ConnectionMaxLifetime: 60 * time.Second,
						},
						AppMetricDB: db.DatabaseConfig{
							URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    10,
							MaxIdleConnections:    5,
							ConnectionMaxLifetime: 60 * time.Second,
						},
					},
					Aggregator: AggregatorConfig{
						AggregatorExecuteInterval: 30 * time.Second,
						PolicyPollerInterval:      30 * time.Second,
						SaveInterval:              30 * time.Second,
						MetricPollerCount:         10,
						AppMonitorChannelSize:     100,
						AppMetricChannelSize:      100,
						MetricCacheSizePerApp:     500,
					},
					Evaluator: EvaluatorConfig{
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
						MetricCollectorURL: "http://localhost:8083",
						TLSClientCerts: models.TLSCerts{
							KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/mc.key",
							CertFile:   "/var/vcap/jobs/autoscaler/config/certs/mc.crt",
							CACertFile: "/var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt",
						},
					},
					DefaultBreachDurationSecs: 600,
					DefaultStatWindowSecs:     300,
					CircuitBreaker: CircuitBreakerConfig{
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
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
  app_metrics_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
scalingEngine:
  scaling_engine_url: http://localhost:8082
metricCollector:
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Aggregator.PolicyPollerInterval).To(Equal(DefaultPolicyPollerInterval))

				Expect(err).NotTo(HaveOccurred())
				Expect(conf).To(Equal(&Config{
					Logging:           helpers.LoggingConfig{Level: "info"},
					HttpClientTimeout: 5 * time.Second,
					Server: ServerConfig{
						Port: 8080,
						TLS:  models.TLSCerts{},
					},
					Health: models.HealthConfig{
						Port: 8081,
					},
					DB: DBConfig{
						PolicyDB: db.DatabaseConfig{
							URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    0,
							MaxIdleConnections:    0,
							ConnectionMaxLifetime: 0 * time.Second,
						},
						AppMetricDB: db.DatabaseConfig{
							URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    0,
							MaxIdleConnections:    0,
							ConnectionMaxLifetime: 0 * time.Second,
						},
					},
					Aggregator: AggregatorConfig{
						AggregatorExecuteInterval: DefaultAggregatorExecuteInterval,
						PolicyPollerInterval:      DefaultPolicyPollerInterval,
						MetricPollerCount:         DefaultMetricPollerCount,
						AppMonitorChannelSize:     DefaultAppMonitorChannelSize,
						AppMetricChannelSize:      DefaultAppMetricChannelSize,
						SaveInterval:              DefaultSaveInterval,
						MetricCacheSizePerApp:     DefaultMetricCacheSizePerApp,
					},
					Evaluator: EvaluatorConfig{
						EvaluationManagerInterval: DefaultEvaluationExecuteInterval,
						EvaluatorCount:            DefaultEvaluatorCount,
						TriggerArrayChannelSize:   DefaultTriggerArrayChannelSize,
					},
					ScalingEngine: ScalingEngineConfig{
						ScalingEngineURL: "http://localhost:8082"},
					MetricCollector: MetricCollectorConfig{
						MetricCollectorURL: "http://localhost:8083"},
					DefaultBreachDurationSecs: 600,
					DefaultStatWindowSecs:     300,
					CircuitBreaker: CircuitBreakerConfig{
						BackOffInitialInterval:  DefaultBackOffInitialInterval,
						BackOffMaxInterval:      DefaultBackOffMaxInterval,
						ConsecutiveFailureCount: DefaultBreakerConsecutiveFailureCount,
					},
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: NOT-INTEGER-VALUE
defaultBreachDurationSecs: 600
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: NOT-INTEGER-VALUE
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
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
  app_metrics_db:
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
  metric_collector_url: http://localhost:8083
defaultStatWindowSecs: 300
defaultBreachDurationSecs: 300
health:
  port: NOT-INTEGER-VALUE
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal !!str `NOT-INT...` into int")))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{
				Logging: helpers.LoggingConfig{Level: "info"},
				Server: ServerConfig{
					NodeAddrs: []string{"address1", "address2"},
					NodeIndex: 0,
				},
				DB: DBConfig{
					PolicyDB: db.DatabaseConfig{
						URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					},
					AppMetricDB: db.DatabaseConfig{
						URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					},
				},
				Aggregator: AggregatorConfig{
					AggregatorExecuteInterval: 30 * time.Second,
					PolicyPollerInterval:      30 * time.Second,
					SaveInterval:              30 * time.Second,
					MetricPollerCount:         10,
					AppMonitorChannelSize:     100,
					AppMetricChannelSize:      100,
					MetricCacheSizePerApp:     500,
				},
				Evaluator: EvaluatorConfig{
					EvaluationManagerInterval: 30 * time.Second,
					EvaluatorCount:            10,
					TriggerArrayChannelSize:   100},
				ScalingEngine: ScalingEngineConfig{
					ScalingEngineURL: "http://localhost:8082"},
				MetricCollector: MetricCollectorConfig{
					MetricCollectorURL: "http://localhost:8083",
				},
				DefaultBreachDurationSecs: 600,
				DefaultStatWindowSecs:     300,
				HttpClientTimeout:         10 * time.Second,
			}
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when policy db url is not set", func() {

			BeforeEach(func() {
				conf.DB.PolicyDB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: db.policy_db.url is empty"))
			})
		})

		Context("when appmetric db url is not set", func() {

			BeforeEach(func() {
				conf.DB.AppMetricDB.URL = ""
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
					conf.Server.NodeIndex = -1
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: server.node_index out of range"))
				})
			})

			Context("when node index is >= number of nodes", func() {
				BeforeEach(func() {
					conf.Server.NodeIndex = 2
					conf.Server.NodeAddrs = []string{"address1", "address2"}
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: server.node_index out of range"))
				})
			})

		})

		Context("when HttpClientTimeout is <= 0", func() {
			BeforeEach(func() {
				conf.HttpClientTimeout = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: http_client_timeout is less-equal than 0"))
			})
		})

	})
})
