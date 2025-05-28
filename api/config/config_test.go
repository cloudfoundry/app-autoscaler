package config_test

import (
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

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

	Describe("Load Config", func() {
		When("runnning in a cf container", func() {
			var expectedDbUrl = "postgres://foo:bar@postgres.example.com:5432/policy_db?sslcert=%2Ftmp%2Fclient_cert.sslcert&sslkey=%2Ftmp%2Fclient_key.sslkey&sslrootcert=%2Ftmp%2Fserver_ca.sslrootcert" // #nosec G101
			var expectedTLSConfig = models.TLSCerts{
				KeyFile:    "some/path/in/container/cfcert.key",
				CertFile:   "some/path/in/container/cfcert.crt",
				CACertFile: "some/path/in/container/cfcert.crt",
			}

			BeforeEach(func() {
				mockVCAPConfigurationReader.GetPortReturns(3333)
				mockVCAPConfigurationReader.GetInstanceTLSCertsReturns(expectedTLSConfig)
				mockVCAPConfigurationReader.IsRunningOnCFReturns(true)
				mockVCAPConfigurationReader.MaterializeDBFromServiceReturns(expectedDbUrl, nil)
			})

			JustBeforeEach(func() {
				conf, err = LoadConfig("", mockVCAPConfigurationReader)
			})

			It("should set logging to plain sink", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Logging.PlainTextSink).To(BeTrue())
			})

			It("send certs to scalingengineScalingEngine TlSClientCert", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.ScalingEngine.TLSClientCerts).To(Equal(expectedTLSConfig))
			})
			It("send certs to Scheduler TlSClientCert", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Scheduler.TLSClientCerts).To(Equal(expectedTLSConfig))
			})

			It("sets EventGenerator TlSClientCert", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.EventGenerator.TLSClientCerts).To(Equal(expectedTLSConfig))
			})

			It("sets env variable over config file", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.CFServer.Port).To(Equal(3333))
				Expect(conf.Server.Port).To(Equal(0))
			})

			When("handling available databases", func() {
				It("calls vcapReader ConfigureDatabases with the right arguments", func() {
					testhelpers.ExpectConfigureDatabasesCalledOnce(err, mockVCAPConfigurationReader, conf.CredHelperImpl)
				})
			})

			When("service is empty", func() {
				var expectedErr error
				BeforeEach(func() {
					expectedErr = fmt.Errorf("publicapiserver config service not found")
					mockVCAPConfigurationReader.GetServiceCredentialContentReturns([]byte(""), expectedErr)
				})

				It("should error with config service not found", func() {
					Expect(err).To(MatchError(MatchRegexp("publicapiserver config service not found")))
				})
			})

			When("VCAP_SERVICES has catalog", func() {
				var expectedCatalogContent string

				BeforeEach(func() {
					expectedCatalogContent = `{"services":[{"id":"1","name":"autoscaler","description":"Autoscaler service","bindable":true,"plans":[{"id":"1","name":"standard","description":"Standard plan"}]}]}` // #nosec G101
					expectedPublicapiConfigContent := `{ "cred_helper_impl": "default" }`

					mockVCAPConfigurationReader.GetServiceCredentialContentReturnsOnCall(0, []byte(expectedPublicapiConfigContent), nil) // #nosec G101
					mockVCAPConfigurationReader.GetServiceCredentialContentReturnsOnCall(1, []byte(expectedCatalogContent), nil)         // #nosec G101
				})

				It("loads the db config from VCAP_SERVICES successfully", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.CatalogPath).NotTo(BeEmpty())
					actualCatalogContent, err := os.ReadFile(conf.CatalogPath)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(actualCatalogContent)).To(Equal(expectedCatalogContent))
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
					configBytes = []byte(testhelpers.LoadFile("invalid_config.yml"))
				})

				It("returns an error", func() {
					Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
				})
			})
			Context("with valid yaml", func() {
				BeforeEach(func() {
					configBytes = []byte(testhelpers.LoadFile("valid_config.yml"))
				})

				It("It returns the config", func() {
					Expect(err).NotTo(HaveOccurred())

					Expect(conf.Logging.Level).To(Equal("debug"))
					Expect(conf.BrokerServer.Port).To(Equal(8080))
					Expect(conf.BrokerServer.TLS).To(Equal(
						models.TLSCerts{
							KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/broker.key",
							CACertFile: "/var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt",
							CertFile:   "/var/vcap/jobs/autoscaler/config/certs/broker.crt",
						},
					))
					Expect(conf.Server.Port).To(Equal(8081))
					Expect(conf.Server.TLS).To(Equal(
						models.TLSCerts{
							KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/api.key",
							CACertFile: "/var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt",
							CertFile:   "/var/vcap/jobs/autoscaler/config/certs/api.crt",
						},
					))
					Expect(conf.Db[db.BindingDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    10,
							MaxIdleConnections:    5,
							ConnectionMaxLifetime: 60 * time.Second,
						}))
					Expect(conf.Db[db.PolicyDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    10,
							MaxIdleConnections:    5,
							ConnectionMaxLifetime: 60 * time.Second,
						}))
					Expect(conf.BrokerCredentials[0].BrokerUsername).To(Equal("broker_username"))
					Expect(conf.BrokerCredentials[0].BrokerPassword).To(Equal("broker_password"))
					Expect(conf.CatalogPath).To(Equal("../exampleconfig/catalog-example.json"))
					Expect(conf.CatalogSchemaPath).To(Equal("../schemas/catalog.schema.json"))
					Expect(conf.PolicySchemaPath).To(Equal("../exampleconfig/policy.schema.json"))
					Expect(conf.Scheduler).To(Equal(
						SchedulerConfig{
							SchedulerURL: "https://localhost:8083",
							TLSClientCerts: models.TLSCerts{
								KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/sc.key",
								CACertFile: "/var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt",
								CertFile:   "/var/vcap/jobs/autoscaler/config/certs/sc.crt",
							},
						},
					))
					Expect(conf.MetricsForwarder).To(Equal(
						MetricsForwarderConfig{
							MetricsForwarderUrl:     "https://localhost:8088",
							MetricsForwarderMtlsUrl: "https://mtlssdsdds:8084",
						},
					))
					Expect(conf.InfoFilePath).To(Equal("/var/vcap/jobs/autoscaer/config/info-file.json"))
					Expect(conf.CF).To(Equal(
						cf.Config{
							API:      "https://api.example.com",
							ClientID: "client-id",
							Secret:   "client-secret",
							ClientConfig: cf.ClientConfig{
								SkipSSLValidation: false,
								MaxRetries:        3,
								MaxRetryWaitMs:    27,
							},
						},
					))
					Expect(conf.CredHelperImpl).To(Equal("default"))
					Expect(conf.ScalingRules.CPU.LowerThreshold).To(Equal(22))
					Expect(conf.ScalingRules.CPU.UpperThreshold).To(Equal(33))
					Expect(conf.ScalingRules.CPUUtil.LowerThreshold).To(Equal(22))
					Expect(conf.ScalingRules.CPUUtil.UpperThreshold).To(Equal(33))
					Expect(conf.ScalingRules.DiskUtil.LowerThreshold).To(Equal(22))
					Expect(conf.ScalingRules.DiskUtil.UpperThreshold).To(Equal(33))
					Expect(conf.ScalingRules.Disk.LowerThreshold).To(Equal(22))
					Expect(conf.ScalingRules.Disk.UpperThreshold).To(Equal(33))
				})
			})

			Context("with partial config", func() {
				BeforeEach(func() {
					configBytes = []byte(testhelpers.LoadFile("partial_config.yml"))
				})
				It("It returns the default values", func() {
					Expect(err).NotTo(HaveOccurred())

					Expect(conf.Logging.Level).To(Equal("info"))
					Expect(conf.BrokerServer.Port).To(Equal(8080))
					Expect(conf.Server.Port).To(Equal(8081))
					Expect(conf.Db[db.BindingDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    0,
							MaxIdleConnections:    0,
							ConnectionMaxLifetime: 0 * time.Second,
						}))
					Expect(conf.Db[db.PolicyDb]).To(Equal(
						db.DatabaseConfig{
							URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
							MaxOpenConnections:    0,
							MaxIdleConnections:    0,
							ConnectionMaxLifetime: 0 * time.Second,
						}))
					Expect(conf.ScalingRules.CPU.LowerThreshold).To(Equal(1))
					Expect(conf.ScalingRules.CPU.UpperThreshold).To(Equal(100))
					Expect(conf.ScalingRules.CPUUtil.LowerThreshold).To(Equal(1))
					Expect(conf.ScalingRules.CPUUtil.UpperThreshold).To(Equal(100))
					Expect(conf.ScalingRules.DiskUtil.LowerThreshold).To(Equal(1))
					Expect(conf.ScalingRules.DiskUtil.UpperThreshold).To(Equal(100))
					Expect(conf.ScalingRules.Disk.LowerThreshold).To(Equal(1))
					Expect(conf.ScalingRules.Disk.UpperThreshold).To(Equal(2 * 1024))
				})
			})

			Context("when max_amount of rate_limit is not an integer", func() {
				BeforeEach(func() {
					configBytes = []byte(`
rate_limit:
  max_amount: NOT-INTEGER
`)
				})
				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("failed to read config file")))
				})
			})

			Context("when valid_duration of rate_limit is not a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`
rate_limit:
  valid_duration: NOT-TIME-DURATION
`)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("failed to read config file")))
				})
			})

		})

		Describe("Validate", func() {
			BeforeEach(func() {
				conf = &Config{}
				conf.Db = make(map[string]db.DatabaseConfig)
				conf.Db[db.BindingDb] = db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}
				conf.Db[db.PolicyDb] = db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}

				brokerCred1 := BrokerCredentialsConfig{
					BrokerUsernameHash: []byte("$2a$10$WNO1cPko4iDAT6MkhaDojeJMU8ZdNH6gt.SapsFOsC0OF4cQ9qQwu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_username")'
					BrokerPasswordHash: []byte("$2a$10$evLviRLcIPKnWQqlBl3DJOvBZir9vJ4gdEeyoGgvnK/CGBnxIAFRu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_password")'
				}
				var brokerCreds []BrokerCredentialsConfig
				brokerCreds = append(brokerCreds, brokerCred1)
				conf.BrokerCredentials = brokerCreds

				conf.CatalogSchemaPath = "../schemas/catalog.schema.json"
				conf.CatalogPath = "../exampleconfig/catalog-example.json"
				conf.PolicySchemaPath = "../exampleconfig/policy.schema.json"

				conf.Scheduler.SchedulerURL = "https://localhost:8083"

				conf.ScalingEngine.ScalingEngineUrl = "https://localhost:8084"
				conf.EventGenerator.EventGeneratorUrl = "https://localhost:8085"
				conf.MetricsForwarder.MetricsForwarderUrl = "https://localhost:8088"

				conf.CF.API = "https://api.bosh-lite.com"
				conf.CF.ClientID = "client-id"
				conf.CF.Secret = "secret"

				conf.InfoFilePath = "../exampleconfig/info-file.json"

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

			Context("When all the configs are valid", func() {
				It("should not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when bindingdb url is not set", func() {
				BeforeEach(func() {
					conf.Db[db.BindingDb] = db.DatabaseConfig{URL: ""}
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: BindingDB URL is empty")))
				})
			})

			Context("when policydb url is not set", func() {
				BeforeEach(func() {
					conf.Db[db.PolicyDb] = db.DatabaseConfig{URL: ""}
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: PolicyDB URL is empty")))
				})
			})

			Context("when scheduler url is not set", func() {
				BeforeEach(func() {
					conf.Scheduler.SchedulerURL = ""
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: scheduler.scheduler_url is empty")))
				})
			})

			Context("when neither the broker username nor its hash is set", func() {
				BeforeEach(func() {
					brokerCred1 := BrokerCredentialsConfig{
						BrokerPasswordHash: []byte(""),
						BrokerPassword:     "",
					}
					var brokerCreds []BrokerCredentialsConfig
					brokerCreds = append(brokerCreds, brokerCred1)
					conf.BrokerCredentials = brokerCreds
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: both broker_username and broker_username_hash are empty, please provide one of them")))
				})
			})

			Context("when both the broker username and its hash are set", func() {
				BeforeEach(func() {
					brokerCred1 := BrokerCredentialsConfig{
						BrokerUsername:     "broker_username",
						BrokerUsernameHash: []byte("$2a$10$WNO1cPko4iDAT6MkhaDojeJMU8ZdNH6gt.SapsFOsC0OF4cQ9qQwu"),
					}
					var brokerCreds []BrokerCredentialsConfig
					brokerCreds = append(brokerCreds, brokerCred1)
					conf.BrokerCredentials = brokerCreds
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: both broker_username and broker_username_hash are set, please provide only one of them")))
				})
			})

			Context("when just the broker username is set", func() {
				BeforeEach(func() {
					conf.BrokerCredentials[0].BrokerUsername = "broker_username"
					conf.BrokerCredentials[0].BrokerUsernameHash = []byte("")
				})
				It("should not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when the broker username hash is set to an invalid value", func() {
				BeforeEach(func() {
					conf.BrokerCredentials[0].BrokerUsernameHash = []byte("not a bcrypt hash")
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: broker_username_hash is not a valid bcrypt hash")))
				})
			})

			Context("when neither the broker password nor its hash is set", func() {
				BeforeEach(func() {
					conf.BrokerCredentials[0].BrokerPassword = ""
					conf.BrokerCredentials[0].BrokerPasswordHash = []byte("")
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: both broker_password and broker_password_hash are empty, please provide one of them")))
				})
			})

			Context("when both the broker password and its hash are set", func() {
				BeforeEach(func() {
					conf.BrokerCredentials[0].BrokerPassword = "broker_password"
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: both broker_password and broker_password_hash are set, please provide only one of them")))
				})
			})

			Context("when just the broker password is set", func() {
				BeforeEach(func() {
					conf.BrokerCredentials[0].BrokerPassword = "broker_password"
					conf.BrokerCredentials[0].BrokerPasswordHash = []byte("")
				})
				It("should not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when the broker password hash is set to an invalid value", func() {
				BeforeEach(func() {
					brokerCred1 := BrokerCredentialsConfig{
						BrokerUsername:     "broker_username",
						BrokerPasswordHash: []byte("not a bcrypt hash"),
					}
					var brokerCreds []BrokerCredentialsConfig
					brokerCreds = append(brokerCreds, brokerCred1)
					conf.BrokerCredentials = brokerCreds
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: broker_password_hash is not a valid bcrypt hash")))
				})
			})

			Context("when eventgenerator url is not set", func() {
				BeforeEach(func() {
					conf.EventGenerator.EventGeneratorUrl = ""
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: event_generator.event_generator_url is empty")))
				})
			})

			Context("when scalingengine url is not set", func() {
				BeforeEach(func() {
					conf.ScalingEngine.ScalingEngineUrl = ""
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: scaling_engine.scaling_engine_url is empty")))
				})
			})

			Context("when metricsforwarder url is not set", func() {
				BeforeEach(func() {
					conf.MetricsForwarder.MetricsForwarderUrl = ""
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: metrics_forwarder.metrics_forwarder_url is empty")))
				})
			})

			Context("when catalog schema path is not set", func() {
				BeforeEach(func() {
					conf.CatalogSchemaPath = ""
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: CatalogSchemaPath is empty")))
				})
			})

			Context("when catalog path is not set", func() {
				BeforeEach(func() {
					conf.CatalogPath = ""
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: CatalogPath is empty")))
				})
			})

			Context("when policy schema path is not set", func() {
				BeforeEach(func() {
					conf.PolicySchemaPath = ""
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: PolicySchemaPath is empty")))
				})
			})

			Context("when catalog is not valid json", func() {
				BeforeEach(func() {
					conf.CatalogPath = "../exampleconfig/catalog-invalid-json-example.json"
				})
				It("should err", func() {
					Expect(err).To(MatchError("invalid character '[' after object key"))
				})
			})

			Context("when catalog is missing required fields", func() {
				BeforeEach(func() {
					conf.CatalogPath = "../exampleconfig/catalog-missing-example.json"
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("{\"name is required\"}")))
				})
			})

			Context("when catalog has invalid type fields", func() {
				BeforeEach(func() {
					conf.CatalogPath = "../exampleconfig/catalog-invalid-example.json"
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("{\"Invalid type. Expected: boolean, given: integer\"}")))
				})
			})

			Context("when info_file_path is not set", func() {
				BeforeEach(func() {
					conf.InfoFilePath = ""
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: InfoFilePath is empty")))
				})
			})

			Context("when cf.client_id is not set", func() {
				BeforeEach(func() {
					conf.CF.ClientID = ""
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: client_id is empty")))
				})
			})

			Context("when rate_limit.max_amount is <= zero", func() {
				BeforeEach(func() {
					conf.RateLimit.MaxAmount = 0
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: RateLimit.MaxAmount is equal or less than zero")))
				})
			})

			Context("when rate_limit.valid_duration is <= 0 ns", func() {
				BeforeEach(func() {
					conf.RateLimit.ValidDuration = 0 * time.Nanosecond
				})
				It("should err", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: RateLimit.ValidDuration is equal or less than zero nanosecond")))
				})
			})
		})
	})
})
