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
					testhelpers.ExpectConfigureDatabasesCalledOnce(err, mockVCAPConfigurationReader)
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
					Expect(conf.CredentialHelperConfig).
						To(BeAssignableToTypeOf(models.BasicAuthHandlingNative{}))
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

				It("should set logging to redacted by default", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.Logging.PlainTextSink).To(BeFalse())
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
})
