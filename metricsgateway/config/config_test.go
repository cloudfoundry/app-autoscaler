package config_test

import (
	"autoscaler/db"
	"autoscaler/helpers"
	. "autoscaler/metricsgateway/config"
	"autoscaler/models"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("Config", func() {

	var (
		conf        *Config
		err         error
		configBytes []byte
	)

	Context("Load Config", func() {
		JustBeforeEach(func() {
			conf, err = LoadConfig(configBytes)
		})

		Context("with invalid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
 logging:
  level: "debug"
envelop_chan_size: 500
nozzle_count: 3
`)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
			})
		})

		Context("valid config yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("returns the config", func() {
				Expect(err).ShouldNot(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("debug"))
				Expect(conf.EnvelopChanSize).To(Equal(800))
				Expect(conf.NozzleCount).To(Equal(10))
				Expect(conf.MetricServerAddrs).To(Equal([]string{"wss://localhost:8080", "wss://localhost:9080"}))
				Expect(conf.AppManager.AppRefreshInterval).To(Equal(10 * time.Second))
				Expect(conf.AppManager.PolicyDB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.Dispatcher.AppRefreshInterval).To(Equal(10 * time.Second))
				Expect(conf.Emitter.BufferSize).To(Equal(800))
				Expect(conf.Emitter.HandshakeTimeout).To(Equal(100 * time.Millisecond))
				Expect(conf.Emitter.KeepAliveInterval).To(Equal(10 * time.Second))
				Expect(conf.Emitter.TLS).To(Equal(models.TLSCerts{
					KeyFile:    "metrc_server_client.cert",
					CertFile:   "metrc_server_client.key",
					CACertFile: "autoscaler_ca.cert",
				}))
				Expect(conf.Nozzle.RLPAddr).To(Equal("wss://localhost:9999"))
				Expect(conf.Nozzle.ShardID).To(Equal("autoscaler"))
				Expect(conf.Nozzle.TLS).To(Equal(models.TLSCerts{
					KeyFile:    "loggregator_client.cert",
					CertFile:   "loggregator_client.key",
					CACertFile: "autoscaler_ca.cert",
				}))
				Expect(conf.Health.Port).To(Equal(8081))

			})
		})

		Context("valid partial yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
metric_server_addrs:
  - localhost:8080
  - localhost:9080
app_manager:
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
emitter:
  tls:
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: localhost:9999
health:
  port: 8081
`)
			})
			It("returns the default values", func() {
				Expect(err).To(BeNil())

				Expect(conf.Logging.Level).To(Equal("info"))
				Expect(conf.EnvelopChanSize).To(Equal(500))
				Expect(conf.NozzleCount).To(Equal(3))
				Expect(conf.AppManager.AppRefreshInterval).To(Equal(60 * time.Second))

				Expect(conf.Dispatcher.AppRefreshInterval).To(Equal(60 * time.Second))
				Expect(conf.Emitter.BufferSize).To(Equal(500))
				Expect(conf.Emitter.HandshakeTimeout).To(Equal(500 * time.Millisecond))
				Expect(conf.Emitter.KeepAliveInterval).To(Equal(5 * time.Second))

				Expect(conf.Nozzle.ShardID).To(Equal("CF_AUTOSCALER"))
			})
		})

		Context("when it gives a non integer envelop_chan_size", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: NOT-INTEGER-VALUE
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer nozzle_count", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: NOT-INTEGER-VALUE
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})

		})

		Context("when metric_server_addrs is not an array", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: 3
metric_server_addrs: wss://localhost:8080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into \\[\\]string")))
			})

		})
		Context("when app_manager.app_refresh_interval is a time.Duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10k
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
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
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: NOT-INTEGAER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
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
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})

		})

		Context("when policy_db.connection_max_lifetime is not a time.Duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})

		})

		Context("when dispatcher.app_refresh_interval is not a time.Duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10k
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})

		})

		Context("when it gives a non-integer emitter.buffer_size", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: NOT-INTEGER-VALUE
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})

		})

		Context("when emitter.keep_alive_interval is not a time.Duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10kk
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})

		})

		Context("when emitter.handshake_timeout is not a time.Duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100kk
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: 8081
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into time.Duration")))
			})

		})

		Context("when it gives a non-integer health.port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
envelop_chan_size: 800
nozzle_count: 10
metric_server_addrs:
  - wss://localhost:8080
  - wss://localhost:9080
app_manager:
  app_refresh_interval: 10s
  policy_db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
dispatcher:
  app_refresh_interval: 10s
emitter:
  tls: 
    key_file: "metrc_server_client.cert"
    cert_file: "metrc_server_client.key"
    ca_file: "autoscaler_ca.cert"
  buffer_size: 800
  keep_alive_interval: 10s
  handshake_timeout: 100ms
nozzle:
  tls:
    key_file: "loggregator_client.cert"
    cert_file: "loggregator_client.key"
    ca_file: "autoscaler_ca.cert"
  rlp_addr: wss://localhost:9999
  shard_id: autoscaler
health:
  port: NOT-INTEGER-VALUE
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})

		})

	})

	Context("Validate config", func() {
		BeforeEach(func() {
			conf = &Config{
				Logging: helpers.LoggingConfig{
					Level: "info",
				},
				EnvelopChanSize:   500,
				NozzleCount:       3,
				MetricServerAddrs: []string{"wss://localhost:8080", "wss://localhost:9080"},
				AppManager: AppManagerConfig{
					AppRefreshInterval: 10 * time.Second,
					PolicyDB: db.DatabaseConfig{
						URL:                   "postgres://postgres:password@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					},
				},
				Dispatcher: DispatcherConfig{
					AppRefreshInterval: 10 * time.Second,
				},
				Emitter: EmitterConfig{
					BufferSize:        500,
					KeepAliveInterval: 1 * time.Second,
					HandshakeTimeout:  1 * time.Second,
					TLS: models.TLSCerts{
						KeyFile:    "metrc_server_client.cert",
						CertFile:   "metrc_server_client.key",
						CACertFile: "autoscaler_ca.cert",
					},
				},
				Nozzle: NozzleConfig{
					RLPAddr: "wss://localhost:9999",
					ShardID: DefaultShardID,
					TLS: models.TLSCerts{
						KeyFile:    "loggregator_client.cert",
						CertFile:   "loggregator_client.key",
						CACertFile: "autoscaler_ca.cert",
					},
				},
				Health: models.HealthConfig{
					Port: 8081,
				},
			}
		})
		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when nozzle_count <= 0", func() {
			BeforeEach(func() {
				conf.NozzleCount = -1
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: nozzle_count is less-equal than 0"))
			})
		})

		Context("when envelope_chan_size <= 0", func() {
			BeforeEach(func() {
				conf.EnvelopChanSize = -1
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: envelope_chan_size is less-equal than 0"))
			})
		})

		Context("when metrics_server_addrs is empty", func() {
			BeforeEach(func() {
				conf.MetricServerAddrs = []string{}
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: metrics_server_addrs is empty"))
			})
		})
		Context("when app_manager.policy_db.url is empty", func() {
			BeforeEach(func() {
				conf.AppManager.PolicyDB.URL = ""
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_manager.policy_db.url is empty"))
			})
		})

		Context("when app_manager.policy_db.max_open_connections <= 0", func() {
			BeforeEach(func() {
				conf.AppManager.PolicyDB.MaxOpenConnections = -1
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_manager.policy_db.max_open_connections is less-equal than 0"))
			})
		})

		Context("when app_manager.policy_db.max_idle_connections <= 0", func() {
			BeforeEach(func() {
				conf.AppManager.PolicyDB.MaxIdleConnections = -1
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_manager.policy_db.max_idle_connections is less-equal than 0"))
			})
		})

		Context("when app_manager.policy_db.connection_max_lifetime is 0", func() {
			BeforeEach(func() {
				conf.AppManager.PolicyDB.ConnectionMaxLifetime = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_manager.policy_db.connection_max_lifetime is 0"))
			})
		})

		Context("when app_manager.app_refresh_interval is 0", func() {
			BeforeEach(func() {
				conf.AppManager.AppRefreshInterval = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_manager.app_refresh_interval is 0"))
			})
		})

		Context("when dispatcher.app_refresh_interval is 0", func() {
			BeforeEach(func() {
				conf.Dispatcher.AppRefreshInterval = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: dispatcher.app_refresh_interval is 0"))
			})
		})

		Context("when emitter.buffer_size <= 0", func() {
			BeforeEach(func() {
				conf.Emitter.BufferSize = -1
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: emitter.buffer_size is less-equal than 0"))
			})
		})

		Context("when emitter.handshake_timeout is 0", func() {
			BeforeEach(func() {
				conf.Emitter.HandshakeTimeout = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: emitter.handshake_timeout is 0"))
			})
		})

		Context("when emitter.keep_alive_interval is 0", func() {
			BeforeEach(func() {
				conf.Emitter.KeepAliveInterval = 0
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: emitter.keep_alive_interval is 0"))
			})
		})

		Context("when emitter.tls.cert_file is empty", func() {
			BeforeEach(func() {
				conf.Emitter.TLS.CertFile = ""
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: emitter.tls.cert_file is empty"))
			})
		})
		Context("when emitter.tls.key_file is empty", func() {
			BeforeEach(func() {
				conf.Emitter.TLS.KeyFile = ""
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: emitter.tls.key_file is empty"))
			})
		})
		Context("when emitter.tls.ca_file is empty", func() {
			BeforeEach(func() {
				conf.Emitter.TLS.CACertFile = ""
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: emitter.tls.ca_file is empty"))
			})
		})

		Context("when nozzle.rlp_addr is empty", func() {
			BeforeEach(func() {
				conf.Nozzle.RLPAddr = ""
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: nozzle.rlp_addr is empty"))
			})
		})

		Context("when nozzle.shard_id is empty", func() {
			BeforeEach(func() {
				conf.Nozzle.ShardID = ""
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: nozzle.shard_id is empty"))
			})
		})

		Context("when nozzle.tls.cert_file is empty", func() {
			BeforeEach(func() {
				conf.Nozzle.TLS.CertFile = ""
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: nozzle.tls.cert_file is empty"))
			})
		})
		Context("when nozzle.tls.key_file is empty", func() {
			BeforeEach(func() {
				conf.Nozzle.TLS.KeyFile = ""
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: nozzle.tls.key_file is empty"))
			})
		})
		Context("when nozzle.tls.ca_file is empty", func() {
			BeforeEach(func() {
				conf.Nozzle.TLS.CACertFile = ""
			})
			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: nozzle.tls.ca_file is empty"))
			})
		})

	})
})
