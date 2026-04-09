package config_test

import (
	"errors"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
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
	)

	BeforeEach(func() {
		mockVCAPConfigurationReader = &fakes.FakeVCAPConfigurationReader{}
	})

	Describe("LoadConfig", func() {
		When("config is read from env", func() {
			var expectedDbUrl string

			JustBeforeEach(func() {
				mockVCAPConfigurationReader.IsRunningOnCFReturns(true)
				mockVCAPConfigurationReader.MaterializeDBFromServiceReturns(expectedDbUrl, nil)
				conf, err = LoadConfig("", mockVCAPConfigurationReader)
			})

			When("vcap PORT is set to a number ", func() {
				BeforeEach(func() {
					mockVCAPConfigurationReader.GetPortReturns(3333)
				})

				It("sets env variable over config file", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.Server.Port).To(Equal(3333))
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

			When("VCAP_SERVICES has credentials for syslog client", func() {
				var expectedTLSConfig models.TLSCerts

				BeforeEach(func() {
					expectedTLSConfig = models.TLSCerts{
						CertFile:   "/tmp/client_cert.sslcert",
						KeyFile:    "/tmp/client_key.sslkey",
						CACertFile: "/tmp/server_ca.sslrootcert",
					}

					mockVCAPConfigurationReader.MaterializeTLSConfigFromServiceReturns(expectedTLSConfig, nil)
				})

				It("loads the syslog config from VCAP_SERVICES", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.SyslogConfig.TLS).To(Equal(expectedTLSConfig))
				})
			})

			When("handling available databases", func() {
				It("calls vcapReader ConfigureDatabases with the right arguments", func() {
					testhelpers.ExpectConfigureDatabasesCalledOnce(err, mockVCAPConfigurationReader, conf.CredHelperImpl)
				})
			})

			When("VCAP_SERVICES has metricsforwarder config", func() {
				BeforeEach(func() {
					mockVCAPConfigurationReader.GetServiceCredentialContentReturns([]byte(` {
									"cache_cleanup_interval":"10h",
									"cache_ttl":"90s",
									"cred_helper_impl": "default",
									"health":{"password":"health-password","username":"health-user"},
									"logging": {
										"level": "debug"
									},
									"loggregator": {
										"metron_address": "metron-vcap-addrs:3457",
									}
								}`), nil) // #nosec G101
				})

				It("loads the config from VCAP_SERVICES", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.Logging.Level).To(Equal("debug"))
					Expect(conf.LoggregatorConfig.MetronAddress).To(Equal("metron-vcap-addrs:3457"))
					Expect(conf.CacheTTL).To(Equal(90 * time.Second))
				})
			})
		})

		When("config is read from file", func() {
			JustBeforeEach(func() {
				configFile = testhelpers.BytesToFile(configBytes)
				conf, err = LoadConfig(configFile, mockVCAPConfigurationReader)
			})

			AfterEach(func() {
				Expect(os.Remove(configFile)).To(Succeed())
			})

			BeforeEach(func() {
				mockVCAPConfigurationReader.IsRunningOnCFReturns(false)
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
  server_config:
    port: 9999
cred_helper_impl: default
`)
				})

				It("returns the config", func() {
					Expect(conf.Server.Port).To(Equal(8081))
					Expect(conf.Logging.Level).To(Equal("debug"))
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
  server_config:
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
				})
			})

		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Server.Port = 8081
			conf.Logging.Level = "debug"
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
			conf.Db[db.BindingDb] = db.DatabaseConfig{
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

		It("should set logging to redacted by default", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(conf.Logging.PlainTextSink).To(BeFalse())
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
					Expect(err).To(MatchError(MatchRegexp("SyslogServer Loggregator CACert is empty")))
				})
			})

			When("SyslogServer CertFile is not set", func() {
				BeforeEach(func() {
					conf.SyslogConfig.TLS.KeyFile = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("SyslogServer ClientKey is empty")))
				})
			})

			When("SyslogServer ClientCert is not set", func() {
				BeforeEach(func() {
					conf.SyslogConfig.TLS.CertFile = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("SyslogServer ClientCert is empty")))
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
				Expect(err).To(MatchError(MatchRegexp("configuration error: Policy DB url is empty")))
			})
		})

		When("binding db url is not set", func() {
			BeforeEach(func() {
				conf.Db[db.BindingDb] = db.DatabaseConfig{URL: ""}
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("configuration error: Binding DB url is empty")))
			})
		})

		When("Loggregator CACert is not set", func() {
			BeforeEach(func() {
				conf.LoggregatorConfig.TLS.CACertFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Loggregator CACert is empty")))
			})
		})

		When("Loggregator ClientCert is not set", func() {
			BeforeEach(func() {
				conf.LoggregatorConfig.TLS.CertFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Loggregator ClientCert is empty")))
			})
		})

		When("Loggregator ClientKey is not set", func() {
			BeforeEach(func() {
				conf.LoggregatorConfig.TLS.KeyFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Loggregator ClientKey is empty")))
			})
		})

		When("rate_limit.max_amount is <= zero", func() {
			BeforeEach(func() {
				conf.RateLimit.MaxAmount = 0
			})

			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("RateLimit.MaxAmount is less than or equal to zero")))
			})
		})

		When("rate_limit.valid_duration is <= 0 ns", func() {
			BeforeEach(func() {
				conf.RateLimit.ValidDuration = 0 * time.Nanosecond
			})

			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("RateLimit.ValidDuration is less than or equal to zero")))
			})
		})
	})
})
