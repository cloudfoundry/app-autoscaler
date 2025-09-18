package config_test

import (
	"errors"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"time"
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
			var expectedDbUrl = "postgres://foo:bar@postgres.example.com:5432/policy_db?sslcert=%2Ftmp%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fserver_ca.sslrootcert" // #nosec G101

			BeforeEach(func() {
				mockVCAPConfigurationReader.GetPortReturns(3333)
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

			Context("with invalid yaml", func() {
				BeforeEach(func() {
					configBytes = []byte(testhelpers.LoadFile("invalid.txt"))
				})

				It("returns an error", func() {
					Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
				})
			})

			Context("with valid yaml", func() {
				BeforeEach(func() {
					configBytes = []byte(testhelpers.LoadFile("valid.yml"))
				})

				It("returns the config", func() {
					Expect(err).NotTo(HaveOccurred())

					Expect(conf.CF.API).To(Equal("https://api.example.com"))
					Expect(conf.CF.ClientID).To(Equal("autoscaler_client_id"))
					Expect(conf.CF.Secret).To(Equal("autoscaler_client_secret"))
					Expect(conf.CF.SkipSSLValidation).To(Equal(false))

					Expect(conf.Server.Port).To(Equal(8989))
					Expect(conf.Server.TLS.KeyFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.key"))
					Expect(conf.Server.TLS.CertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.crt"))
					Expect(conf.Server.TLS.CACertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/ca.crt"))

					Expect(conf.CFServer.Port).To(Equal(2222))
					Expect(conf.CFServer.XFCC.ValidOrgGuid).To(Equal("valid_org_guid"))
					Expect(conf.CFServer.XFCC.ValidSpaceGuid).To(Equal("valid_space_guid"))

					Expect(conf.Health.ServerConfig.Port).To(Equal(9999))
					Expect(conf.Logging.Level).To(Equal("debug"))

					Expect(conf.Db[db.PolicyDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    10,
							MaxIdleConnections:    5,
							ConnectionMaxLifetime: 60 * time.Second,
						}))
					Expect(conf.Db[db.ScalingEngineDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    10,
							MaxIdleConnections:    5,
							ConnectionMaxLifetime: 60 * time.Second,
						}))
					Expect(conf.Db[db.SchedulerDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    10,
							MaxIdleConnections:    5,
							ConnectionMaxLifetime: 60 * time.Second,
						}))

					Expect(conf.DefaultCoolDownSecs).To(Equal(300))

					Expect(conf.LockSize).To(Equal(32))

					Expect(conf.HttpClientTimeout).To(Equal(10 * time.Second))
				})
			})

			Context("with partial config", func() {
				BeforeEach(func() {
					configBytes = []byte(testhelpers.LoadFile("partial.yml"))
					// conf, err = config.LoadConfig("partial.yml", mockVCAPConfigurationReader)
				})

				It("returns default values", func() {
					Expect(err).NotTo(HaveOccurred())

					Expect(conf.CF.SkipSSLValidation).To(Equal(false))
					Expect(conf.Logging.Level).To(Equal("info"))
					Expect(conf.Db[db.PolicyDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    0,
							MaxIdleConnections:    0,
							ConnectionMaxLifetime: 0 * time.Second,
						}))
					Expect(conf.Db[db.ScalingEngineDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    0,
							MaxIdleConnections:    0,
							ConnectionMaxLifetime: 0 * time.Second,
						}))
					Expect(conf.Db[db.SchedulerDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    0,
							MaxIdleConnections:    0,
							ConnectionMaxLifetime: 0 * time.Second,
						}))

					Expect(conf.HttpClientTimeout).To(Equal(5 * time.Second))
					Expect(conf.Health.ServerConfig.Port).To(Equal(8081))
					Expect(conf.Server.Port).To(Equal(8080))
				})
			})

			When("it gives a non integer server port", func() {
				BeforeEach(func() {
					configBytes = []byte(`
server:
  port: port
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
				})
			})

			When("it gives a non integer health port", func() {
				BeforeEach(func() {
					configBytes = []byte(`
health:
  server_config:
    port: port
`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
				})
			})

			When("it gives a non integer of defaultCoolDownSecs", func() {
				BeforeEach(func() {
					configBytes = []byte(`defaultCoolDownSecs: NOT-INTEGER-VALUE`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
				})
			})

			When("it gives a non integer of lockSize", func() {
				BeforeEach(func() {
					configBytes = []byte(`lockSize: NOT-INTEGER-VALUE`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
				})
			})

			When("http_client_timeout of http is not a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`http_client_timeout: 10k`)
				})

				It("should error", func() {
					Expect(errors.Is(err, ErrReadYaml)).To(BeTrue())
					Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
				})
			})

		})

		Describe("Validate", func() {
			BeforeEach(func() {
				conf = &Config{}
				conf.CF.API = "http://api.example.com"
				conf.CF.SkipSSLValidation = false
				conf.CF.ClientID = "autoscaler_client_id"
				conf.Db = make(map[string]db.DatabaseConfig)
				conf.Db[db.PolicyDb] = db.DatabaseConfig{URL: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"}
				conf.Db[db.ScalingEngineDb] = db.DatabaseConfig{URL: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"}
				conf.Db[db.SchedulerDb] = db.DatabaseConfig{URL: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"}
				conf.DefaultCoolDownSecs = 300
				conf.LockSize = 32
				conf.HttpClientTimeout = 10 * time.Second
			})

			JustBeforeEach(func() {
				err = conf.Validate()
			})

			When("all the configs are valid", func() {
				It("should not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			When("cf config is not valid", func() {
				BeforeEach(func() {
					conf.CF.API = ""
				})

				It("should error", func() {
					Expect(err).To(HaveOccurred())
				})
			})

			When("policy db url is not set", func() {
				BeforeEach(func() {
					conf.Db[db.PolicyDb] = db.DatabaseConfig{
						URL:                   "",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: db.policy_db.url is empty"))
				})
			})

			When("scalingengine db url is not set", func() {
				BeforeEach(func() {
					conf.Db[db.ScalingEngineDb] = db.DatabaseConfig{
						URL:                   "",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: db.scalingengine_db.url is empty"))
				})
			})

			When("scheduler db url is not set", func() {
				BeforeEach(func() {
					conf.Db[db.SchedulerDb] = db.DatabaseConfig{
						URL:                   "",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: db.scheduler_db.url is empty"))
				})
			})

			When("DefaultCoolDownSecs < 60", func() {
				BeforeEach(func() {
					conf.DefaultCoolDownSecs = 10
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: DefaultCoolDownSecs should be between 60 and 3600"))
				})
			})

			When("DefaultCoolDownSecs > 3600", func() {
				BeforeEach(func() {
					conf.DefaultCoolDownSecs = 5000
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: DefaultCoolDownSecs should be between 60 and 3600"))
				})
			})

			When("LockSize <= 0", func() {
				BeforeEach(func() {
					conf.LockSize = 0
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: LockSize is less than or equal to 0"))
				})
			})

			When("HttpClientTimeout is <= 0", func() {
				BeforeEach(func() {
					conf.HttpClientTimeout = 0
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
			Expect(serviceName).To(Equal("scalingengine-config"))
			Expect(credentialKey).To(Equal("scalingengine-config"))

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
