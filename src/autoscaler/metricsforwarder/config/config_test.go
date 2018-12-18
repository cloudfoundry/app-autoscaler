package config_test

import (
	"bytes"
	"time"

	"autoscaler/db"
	. "autoscaler/metricsforwarder/config"

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

	Describe("LoadConfig", func() {

		JustBeforeEach(func() {
			conf, err = LoadConfig(bytes.NewReader(configBytes))
		})

		Context("with invalid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
  server:
    port: 8081
logging:
  level: info
metron_address: 127.0.0.1:3457
loggregator
  ca_cert: "../testcerts/ca.crt"
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
  port: 8081
  node_index: 0
  node_addrs: [address1, address2]
logging:
  level: debug
metron_address: 127.0.0.1:3457
loggregator:
  ca_cert: "../testcerts/ca.crt"
  client_cert: "../testcerts/client.crt"
  client_key: "../testcerts/client.key"
db:
  policy_db:
    url: "postgres://pqgotest:password@localhost/pqgotest"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
`)
			})

			It("returns the config", func() {
				Expect(conf.Server.Port).To(Equal(8081))
				Expect(conf.Logging.Level).To(Equal("debug"))
				Expect(conf.MetronAddress).To(Equal("127.0.0.1:3457"))
				Expect(conf.Server.NodeAddrs).To(Equal([]string{"address1", "address2"}))
				Expect(conf.Server.NodeIndex).To(Equal(0))
				Expect(conf.Db.PolicyDb).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
loggregator:
  ca_cert: "../testcerts/ca.crt"
  client_cert: "../testcerts/client.crt"
  client_key: "../testcerts/client.key"
db:
  policy_db:
    url: "postgres://pqgotest:password@localhost/pqgotest"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Server.Port).To(Equal(6110))
				Expect(conf.Logging.Level).To(Equal("info"))
				Expect(conf.MetronAddress).To(Equal(DefaultMetronAddress))
				Expect(conf.CacheTTL).To(Equal(DefaultCacheTTL))
				Expect(conf.CacheCleanupInterval).To(Equal(DefaultCacheCleanupInterval))
			})
		})

		Context("when it gives a non integer port", func() {
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

		Context("when it gives a non integer max_open_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
loggregator:
  ca_cert: "../testcerts/ca.crt"
  client_cert: "../testcerts/client.crt"
  client_key: "../testcerts/client.key"
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
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
loggregator:
  ca_cert: "../testcerts/ca.crt"
  client_cert: "../testcerts/client.crt"
  client_key: "../testcerts/client.key"
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
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
loggregator:
  ca_cert: "../testcerts/ca.crt"
  client_cert: "../testcerts/client.crt"
  client_key: "../testcerts/client.key"
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6K
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
			conf.Server.Port = 8081
			conf.Server.NodeIndex = 0
			conf.Server.NodeAddrs = []string{"localhost"}
			conf.Logging.Level = "debug"
			conf.MetronAddress = "127.0.0.1:3458"
			conf.LoggregatorConfig.CACertFile = "../testcerts/ca.crt"
			conf.LoggregatorConfig.ClientCertFile = "../testcerts/client.crt"
			conf.LoggregatorConfig.ClientKeyFile = "../testcerts/client.crt"
			conf.Db.PolicyDb = db.DatabaseConfig{
				URL:                   "postgres://pqgotest:password@localhost/pqgotest",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}
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
				conf.Db.PolicyDb.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Policy DB url is empty")))
			})
		})

		Context("when Loggregator CACert is not set", func() {
			BeforeEach(func() {
				conf.LoggregatorConfig.CACertFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Loggregator CACert is empty")))
			})
		})

		Context("when Loggregator ClientCert is not set", func() {
			BeforeEach(func() {
				conf.LoggregatorConfig.ClientCertFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Loggregator ClientCert is empty")))
			})
		})

		Context("when Loggregator ClientKey is not set", func() {
			BeforeEach(func() {
				conf.LoggregatorConfig.ClientKeyFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Loggregator ClientKey is empty")))
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
})
