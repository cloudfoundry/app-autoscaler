package config_test

import (
	"bytes"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsserver/config"
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
node_addrs:
  - 10.0.2.3:8080
	- 10.0.2.5:8080
node_index: 0
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
logging:
  level: DebuG
http_client_timeout: 10s
node_addrs:
  - address1
  - address2
node_index: 0
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
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
  keep_alive_time: 180s
  refresh_interval: 20s
  collect_interval: 10s
  save_interval: 5s
  metric_cache_size_per_app: 100
  persist_metrics: true
  envelope_processor_count: 10
  envelope_channel_size: 500
  metric_channel_size: 500
server:
  port: 8888
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
health:
  port: 9999
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Logging.Level).To(Equal("debug"))
				Expect(conf.HttpClientTimeout).To(Equal(10 * time.Second))
				Expect(conf.NodeAddrs).To(Equal([]string{"address1", "address2"}))
				Expect(conf.NodeIndex).To(Equal(0))

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

				Expect(conf.Collector.WSPort).To(Equal(8989))
				Expect(conf.Collector.TLS.KeyFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.key"))
				Expect(conf.Collector.TLS.CertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.crt"))
				Expect(conf.Collector.TLS.CACertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/ca.crt"))
				Expect(conf.Collector.WSKeepAliveTime).To(Equal(180 * time.Second))
				Expect(conf.Collector.RefreshInterval).To(Equal(20 * time.Second))
				Expect(conf.Collector.CollectInterval).To(Equal(10 * time.Second))
				Expect(conf.Collector.SaveInterval).To(Equal(5 * time.Second))
				Expect(conf.Collector.MetricCacheSizePerApp).To(Equal(100))
				Expect(conf.Collector.PersistMetrics).To(BeTrue())
				Expect(conf.Collector.EnvelopeProcessorCount).To(Equal(10))
				Expect(conf.Collector.EnvelopeChannelSize).To(Equal(500))
				Expect(conf.Collector.MetricChannelSize).To(Equal(500))

				Expect(conf.Server.Port).To(Equal(8888))
				Expect(conf.Server.TLS.KeyFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.key"))
				Expect(conf.Server.TLS.CertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.crt"))
				Expect(conf.Server.TLS.CACertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/ca.crt"))

				Expect(conf.Health.Port).To(Equal(9999))

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

				Expect(conf.Logging.Level).To(Equal(DefaultLoggingLevel))
				Expect(conf.HttpClientTimeout).To(Equal(DefaultHttpClientTimeout))

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
				Expect(conf.Collector.WSPort).To(Equal(DefaultWSPort))
				Expect(conf.Collector.RefreshInterval).To(Equal(DefaultRefreshInterval))
				Expect(conf.Collector.CollectInterval).To(Equal(DefaultCollectInterval))
				Expect(conf.Collector.WSKeepAliveTime).To(Equal(DefaultWSKeepAliveTime))
				Expect(conf.Collector.RefreshInterval).To(Equal(DefaultRefreshInterval))
				Expect(conf.Collector.MetricCacheSizePerApp).To(Equal(DefaultMetricCacheSizePerApp))
				Expect(conf.Collector.PersistMetrics).To(Equal(DefaultIsMetricsPersistencySupported))
				Expect(conf.Collector.EnvelopeProcessorCount).To(Equal(DefaultEnvelopeProcessorCount))
				Expect(conf.Collector.EnvelopeChannelSize).To(Equal(DefaultEnvelopeChannelSize))
				Expect(conf.Collector.MetricChannelSize).To(Equal(DefaultMetricChannelSize))

				Expect(conf.Server.Port).To(Equal(DefaultHTTPServerPort))

				Expect(conf.Health.Port).To(Equal(DefaultHealthPort))

			})
		})
	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.NodeAddrs = []string{"address1", "address2"}
			conf.NodeIndex = 0
			conf.HttpClientTimeout = 10 * time.Second
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
			conf.Collector.WSKeepAliveTime = 180 * time.Second
			conf.Collector.RefreshInterval = 60 * time.Second
			conf.Collector.CollectInterval = 30 * time.Second
			conf.Collector.SaveInterval = 5 * time.Second
			conf.Collector.MetricCacheSizePerApp = 100
			conf.Collector.EnvelopeProcessorCount = 5
			conf.Collector.EnvelopeChannelSize = 300
			conf.Collector.MetricChannelSize = 300
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

		Context("when HttpClientTimeout is <= 0", func() {
			BeforeEach(func() {
				conf.HttpClientTimeout = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: http_client_timeout is less-equal than 0"))
			})
		})

		Context("when node index is out of range", func() {
			Context("when node index is negative", func() {
				BeforeEach(func() {
					conf.NodeIndex = -1
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: node_index out of range"))
				})
			})

			Context("when node index is >= number of nodes", func() {
				BeforeEach(func() {
					conf.NodeIndex = 2
					conf.NodeAddrs = []string{"address1", "address2"}
				})
				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: node_index out of range"))
				})
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

		Context("when KeepAliveTime is <= 0", func() {
			BeforeEach(func() {
				conf.Collector.WSKeepAliveTime = 0
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

		Context("when EnvelopeChannelSize is <= 0", func() {
			BeforeEach(func() {
				conf.Collector.EnvelopeChannelSize = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: envelope_channel_size is less-equal than 0"))
			})
		})

		Context("when MetricChannelSize is <= 0", func() {
			BeforeEach(func() {
				conf.Collector.MetricChannelSize = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: metric_channel_size is less-equal than 0"))
			})
		})

	})
})
