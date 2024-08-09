package config_test

import (
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func bytesToFile(b []byte) string {
	file, err := os.CreateTemp("", "")
	Expect(err).NotTo(HaveOccurred())
	_, err = file.Write(b)
	Expect(err).NotTo(HaveOccurred())
	return file.Name()
}

var _ = Describe("Config", func() {

	var (
		conf        *Config
		err         error
		configBytes []byte
		configFile  string
	)

	Describe("LoadConfig", func() {

		JustBeforeEach(func() {
			configFile = bytesToFile(configBytes)
			conf, err = LoadConfig(configFile)
		})

		AfterEach(func() {
			//clean up config file
			Expect(os.Remove(configFile)).To(Succeed())
		})

		Context("with invalid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
  server:
    port: 8081
  logging:
  level: info

loggregator
	metron_address: 127.0.0.1:3457
	tls:
	  cert_file: "../testcerts/ca.crt"
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
logging:
  level: debug
loggregator:
  metron_address: 127.0.0.1:3457
  tls:
    ca_file: "../testcerts/ca.crt"
    cert_file: "../testcerts/client.crt"
    key_file: "../testcerts/client.key"
db:
  policy_db:
    url: "postgres://pqgotest:password@localhost/pqgotest"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
health:
  port: 9999
cred_helper_impl: default
`)
			})

			It("returns the config", func() {
				Expect(conf.Server.Port).To(Equal(8081))
				Expect(conf.Logging.Level).To(Equal("debug"))
				Expect(conf.Health.Port).To(Equal(9999))
				Expect(conf.LoggregatorConfig.MetronAddress).To(Equal("127.0.0.1:3457"))
				Expect(conf.Db[db.PolicyDb]).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.CredHelperImpl).To(Equal("default"))
			})

			When("PORT env variable is set", func() {

				When("PORT env is a number", func() {
					BeforeEach(func() {
						os.Setenv("PORT", "3333")
					})
					It("prioritize env variable over config file", func() {
						Expect(conf.Server.Port).To(Equal(3333))
					})
				})

				When("PORT env is not number", func() {
					BeforeEach(func() {
						os.Setenv("PORT", "NAN")
					})

					It("return invalid port error", func() {
						Expect(err).To(MatchError(ErrReadEnvironment))
						Expect(err).To(MatchError(MatchRegexp("converting 'NAN' to type int")))
					})

					AfterEach(func() {
						os.Unsetenv("PORT")
					})
				})
			})

			When("VCAP_SERVICES is set", func() {
				BeforeEach(func() {
					// vcap services has a postgres service provisioned by
					// a service broker binding
					vcapServices := `{
					  "autoscaler": [
						  {
							"credentials": {
								"uri":"postgres://foo:bar@postgres.example.com:5432/policy_db"
							},
							"label": "postgres",
							"name": "policy_db",
							"syslog_drain_url": "",
							"tags": ["postgres","postgresql","relational"]
						  }
						]
					}` // #nosec G101

					os.Setenv("VCAP_APPLICATION", "{}")
					os.Setenv("VCAP_SERVICES", vcapServices)
				})

				It("loads the db config from VCAP_SERVICES", func() {
					expectedDbConfig := db.DatabaseConfig{
						URL: "postgres://foo:bar@postgres.example.com:5432/policy_db",
					}

					Expect(err).NotTo(HaveOccurred())
					Expect(conf.Db[db.PolicyDb]).To(Equal(expectedDbConfig))
				})

				AfterEach(func() {
					os.Unsetenv("VCAP_SERVICES")
					os.Unsetenv("VCAP_APPLICATION")
				})
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
loggregator:
  tls:
    ca_file: "../testcerts/ca.crt"
    cert_file: "../testcerts/client.crt"
    key_file: "../testcerts/client.key"
db:
  policy_db:
    url: "postgres://pqgotest:password@localhost/pqgotest"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
health:
  port: 8081
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Server.Port).To(Equal(6110))
				Expect(conf.Logging.Level).To(Equal("info"))
				Expect(conf.LoggregatorConfig.MetronAddress).To(Equal(DefaultMetronAddress))
				Expect(conf.CacheTTL).To(Equal(DefaultCacheTTL))
				Expect(conf.CacheCleanupInterval).To(Equal(DefaultCacheCleanupInterval))
				Expect(conf.Health.Port).To(Equal(8081))
			})
		})

		When("it gives a non integer port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: port
`)
			})

			It("should error", func() {
				Expect(err).To(MatchError(ErrReadYaml))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		When("it gives a non integer health server port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
health:
  port: port
`)
			})

			It("should error", func() {
				Expect(err).To(MatchError(ErrReadYaml))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		When("it gives a non integer max_open_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
loggregator:
  metron_address: 127.0.0.1:3457
  tls:
    ca_file: "../testcerts/ca.crt"
    cert_file: "../testcerts/client.crt"
    key_file: "../testcerts/client.key"
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
health:
  port: 8081
`)
			})

			It("should error", func() {
				Expect(err).To(MatchError(ErrReadYaml))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		When("it gives a non integer max_idle_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
loggregator:
  metron_address: 127.0.0.1:3457
  tls:
    ca_file: "../testcerts/ca.crt"
    cert_file: "../testcerts/client.crt"
    key_file: "../testcerts/client.key"
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
health:
  port: 8081
`)
			})

			It("should error", func() {
				Expect(err).To(MatchError(ErrReadYaml))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		When("connection_max_lifetime of policydb is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
loggregator:
  metron_address: 127.0.0.1:3457
  tls:
    ca_file: "../testcerts/ca.crt"
    cert_file: "../testcerts/client.crt"
    key_file: "../testcerts/client.key"
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6K
health:
  port: 8081
`)
			})

			It("should error", func() {
				Expect(err).To(MatchError(ErrReadYaml))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		When("max_amount of rate_limit is not an interger", func() {
			BeforeEach(func() {
				configBytes = []byte(`
loggregator:
  metron_address: 127.0.0.1:3457
  tls:
    ca_file: "../testcerts/ca.crt"
    cert_file: "../testcerts/client.crt"
    key_file: "../testcerts/client.key"
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
health:
  port: 8081
rate_limit:
  max_amount: NOT-INTEGER
  valid_duration: 1s
`)
			})

			It("should error", func() {
				Expect(err).To(MatchError(ErrReadYaml))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		When("valid_duration of rate_limit is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
loggregator:
  metron_address: 127.0.0.1:3457
  tls:
    ca_file: "../testcerts/ca.crt"
    cert_file: "../testcerts/client.crt"
    key_file: "../testcerts/client.key"
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
health:
  port: 8081
rate_limit:
  max_amount: 2
  valid_duration: NOT-TIME-DURATION
`)
			})

			It("should error", func() {
				Expect(err).To(MatchError(ErrReadYaml))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})
	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Server.Port = 8081
			conf.Logging.Level = "debug"
			conf.Health.Port = 8081
			conf.LoggregatorConfig.MetronAddress = "127.0.0.1:3458"
			conf.LoggregatorConfig.TLS.CACertFile = "../testcerts/ca.crt"
			conf.LoggregatorConfig.TLS.CertFile = "../testcerts/client.crt"
			conf.LoggregatorConfig.TLS.KeyFile = "../testcerts/client.crt"
			conf.Db = make(map[string]db.DatabaseConfig)
			conf.Db[db.PolicyDb] = db.DatabaseConfig{
				URL:                   "postgres://pqgotest:password@localhost/pqgotest",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}
			conf.RateLimit.MaxAmount = 10
			conf.RateLimit.ValidDuration = 1 * time.Second

			conf.CredHelperImpl = "path/to/plugin"
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		When("syslog is available", func() {
			BeforeEach(func() {
				conf.SyslogConfig = SyslogConfig{
					ServerAddress: "localhost",
					Port:          514,
					TLS: models.TLSCerts{
						CACertFile: "../testcerts/ca.crt",
						CertFile:   "../testcerts/client.crt",
						KeyFile:    "../testcerts/client.crt",
					},
				}
				conf.LoggregatorConfig = LoggregatorConfig{}
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			When("SyslogServer CACert is not set", func() {
				BeforeEach(func() {
					conf.SyslogConfig.TLS.CACertFile = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: SyslogServer Loggregator CACert is empty")))
				})
			})

			When("SyslogServer CertFile is not set", func() {
				BeforeEach(func() {
					conf.SyslogConfig.TLS.KeyFile = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: SyslogServer ClientKey is empty")))
				})
			})

			When("SyslogServer ClientCert is not set", func() {
				BeforeEach(func() {
					conf.SyslogConfig.TLS.CertFile = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: SyslogServer ClientCert is empty")))
				})
			})
		})

		When("all the configs are valid", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("policy db url is not set", func() {
			BeforeEach(func() {
				conf.Db[db.PolicyDb] = db.DatabaseConfig{URL: ""}
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Policy DB url is empty")))
			})
		})

		When("Loggregator CACert is not set", func() {
			BeforeEach(func() {
				conf.LoggregatorConfig.TLS.CACertFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Loggregator CACert is empty")))
			})
		})

		When("Loggregator ClientCert is not set", func() {
			BeforeEach(func() {
				conf.LoggregatorConfig.TLS.CertFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Loggregator ClientCert is empty")))
			})
		})

		When("Loggregator ClientKey is not set", func() {
			BeforeEach(func() {
				conf.LoggregatorConfig.TLS.KeyFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Loggregator ClientKey is empty")))
			})
		})

		When("rate_limit.max_amount is <= zero", func() {
			BeforeEach(func() {
				conf.RateLimit.MaxAmount = 0
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: RateLimit.MaxAmount is equal or less than zero")))
			})
		})

		When("rate_limit.valid_duration is <= 0 ns", func() {
			BeforeEach(func() {
				conf.RateLimit.ValidDuration = 0 * time.Nanosecond
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: RateLimit.ValidDuration is equal or less than zero nanosecond")))
			})
		})
	})
})
