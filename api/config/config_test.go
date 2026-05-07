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
					testhelpers.ExpectConfigureDatabasesCalledOnce(err, mockVCAPConfigurationReader)
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
					Expect(conf.BindingRequestSchemaPath).To(Equal("../exampleconfig/policy.schema.json"))
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
					Expect((*conf.CustomMetricsAuthConfig).BasicAuthHandlingImplConfig).
						To(BeAssignableToTypeOf(models.BasicAuthHandlingNative{}))
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
				It("should set logging to redacted by default", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(conf.Logging.PlainTextSink).To(BeFalse())
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
	})
})
