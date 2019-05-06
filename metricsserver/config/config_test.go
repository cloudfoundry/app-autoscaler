package config_test

import (
	"bytes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	"autoscaler/db"
	. "autoscaler/metricsserver/config"
)

var _ = Describe("Config", func() {

	var (
		conf        *Config
		err         error
		configBytes []byte
	)

	Describe("LoadConfig", func() {

		JustBeforeEach(func() {
			conf, err = LoadConfig(bytes.NewReader(configBytes))
		})

		Context("with invalid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
 collector:
  refresh_interval: 30s
`)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
			})
		})

		Context("with valid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
  node_addrs: [address1, address2]
  node_index: 1
health:
  port: 9999
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metric_cache_size_per_app: 100
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
http_client_timeout: 10s
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Server.Port).To(Equal(8989))
				Expect(conf.Health.Port).To(Equal(9999))
				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.DB.PolicyDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.DB.InstanceMetricsDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))

				Expect(conf.Collector.RefreshInterval).To(Equal(20 * time.Second))
				Expect(conf.Collector.CollectInterval).To(Equal(10 * time.Second))
				Expect(conf.Collector.MetricCacheSizePerApp).To(Equal(100))
				Expect(conf.Collector.IsMetricsPersistencySupported).To(BeTrue())
				Expect(conf.Collector.KeepAliveTime).To(Equal(180 * time.Second))
				Expect(conf.Collector.EnvelopeProcessorCount).To(Equal(10))

				Expect(conf.Server.TLS.KeyFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.key"))
				Expect(conf.Server.TLS.CertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.crt"))
				Expect(conf.Server.TLS.CACertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/ca.crt"))
				Expect(conf.Server.NodeAddrs).To(Equal([]string{"address1", "address2"}))
				Expect(conf.Server.NodeIndex).To(Equal(1))

				Expect(conf.HttpClientTimeout).To(Equal(10 * time.Second))

			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.Health.Port).To(Equal(8081))
				Expect(conf.Logging.Level).To(Equal(DefaultLoggingLevel))
				Expect(conf.DB.PolicyDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.DB.InstanceMetricsDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.Collector.RefreshInterval).To(Equal(DefaultRefreshInterval))
				Expect(conf.Collector.CollectInterval).To(Equal(DefaultCollectInterval))
				Expect(conf.Collector.MetricCacheSizePerApp).To(Equal(DefaultMetricCacheSizePerApp))
				Expect(conf.Collector.IsMetricsPersistencySupported).To(Equal(DefaultIsMetricsPersistencySupported))
				Expect(conf.Collector.KeepAliveTime).To(Equal(DefaultKeepAliveTime))
				Expect(conf.Collector.EnvelopeProcessorCount).To(Equal(DefaultEnvelopeProcessorCount))

				Expect(conf.HttpClientTimeout).To(Equal(5 * time.Second))
			})
		})

		Context("when it gives a non integer server port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: port
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer health server port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
health:
  port: port
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_open_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
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
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
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
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6K
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when it gives a non integer max_open_connections of instance_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_idle_connections of instance_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when connection_max_lifetime of instance_metrics_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6k
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when refresh_interval of collector is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 2k
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when collect_interval of collector is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10k
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when save_interval of collector is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5k
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when connection_max_lifetime of instance_metrics_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6t
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when keep_alive_time is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6t
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180d
  envelope_processor_count: 10
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when it gives a non integer envelope_processor_count", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: abc3
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when http_client_timeout of http is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 8081
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
  metrics_persistency_support_flag: true
  keep_alive_time: 180s
  envelope_processor_count: 10
http_client_timeout: 10k
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.DB.PolicyDB = db.DatabaseConfig{
				URL:                   "postgres://pqgotest:password@localhost/pqgotest",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}
			conf.DB.InstanceMetricsDB = db.DatabaseConfig{
				URL:                   "postgres://pqgotest:password@localhost/pqgotest",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}
			conf.Collector.CollectInterval = time.Duration(30 * time.Second)
			conf.Collector.RefreshInterval = time.Duration(60 * time.Second)
			conf.Collector.SaveInterval = time.Duration(5 * time.Second)
			conf.Collector.MetricCacheSizePerApp = 100
			conf.Collector.EnvelopeProcessorCount = 5
			conf.Collector.KeepAliveTime = time.Duration(180 * time.Second)
			conf.Server.NodeAddrs = []string{"address1", "address2"}
			conf.Server.NodeIndex = 0
			conf.HttpClientTimeout = 10 * time.Second
			conf.Health.Port = 8081

		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when all the configs are valid", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when policy db url is not set", func() {
			BeforeEach(func() {
				conf.DB.PolicyDB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: db.policy_db.url is empty"))
			})
		})

		Context("when metrics db url is not set", func() {
			BeforeEach(func() {
				conf.DB.InstanceMetricsDB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: db.instance_metrics_db.url is empty"))
			})
		})

		Context("when collect interval is 0", func() {
			BeforeEach(func() {
				conf.Collector.CollectInterval = time.Duration(0)
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: collector.collect_interval is 0"))
			})
		})

		Context("when refresh interval is 0", func() {
			BeforeEach(func() {
				conf.Collector.RefreshInterval = time.Duration(0)
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: collector.refresh_interval is 0"))
			})
		})

		Context("when save interval is 0", func() {
			BeforeEach(func() {
				conf.Collector.SaveInterval = time.Duration(0)
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: collector.save_interval is 0"))
			})
		})

		Context("when metrics cache size per app is invalid", func() {
			BeforeEach(func() {
				conf.Collector.MetricCacheSizePerApp = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: invalid collector.metric_cache_size_per_app"))
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

			Context("when HttpClientTimeout is <= 0", func() {
				BeforeEach(func() {
					conf.HttpClientTimeout = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: http_client_timeout is less-equal than 0"))
				})
			})

			Context("when KeepAliveTime is <= 0", func() {
				BeforeEach(func() {
					conf.Collector.KeepAliveTime = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: keep_alive_time is less-equal than 0"))
				})
			})

			Context("when EnvelopeProcessorCount is <= 0", func() {
				BeforeEach(func() {
					conf.Collector.EnvelopeProcessorCount = 0
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: envelope_processor_count is less-equal than 0"))
				})
			})

		})

	})
})
