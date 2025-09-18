package config_test

import (
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var (
		conf                        *config.Config
		err                         error
		configFile                  string
		configBytes                 []byte
		mockVCAPConfigurationReader *fakes.FakeVCAPConfigurationReader
	)

	BeforeEach(func() {
		mockVCAPConfigurationReader = &fakes.FakeVCAPConfigurationReader{}
	})
	Describe("LoadConfig", func() {
		When("config is read from env", func() {
			var expectedTLSConfig = models.TLSCerts{
				KeyFile:    "some/path/in/container/cfcert.key",
				CertFile:   "some/path/in/container/cfcert.crt",
				CACertFile: "some/path/in/container/cfcert.crt",
			}

			BeforeEach(func() {
				mockVCAPConfigurationReader.GetPortReturns(3333)
				mockVCAPConfigurationReader.GetInstanceTLSCertsReturns(expectedTLSConfig)
				mockVCAPConfigurationReader.IsRunningOnCFReturns(true)
			})

			JustBeforeEach(func() {
				conf, err = config.LoadConfig("", mockVCAPConfigurationReader)
			})

			It("should set logging to plain sink", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Logging.PlainTextSink).To(BeTrue())
			})

			It("sets env variable over config file", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Health.ServerConfig.Port).To(Equal(3333))
			})

			It("send certs to scalingengineScalingEngine TlSClientCert", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.ScalingEngine.TLSClientCerts).To(Equal(expectedTLSConfig))
			})

			It("send certs to scheduler TLSClientCert", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.Scheduler.TLSClientCerts).To(Equal(expectedTLSConfig))
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
				conf, err = config.LoadConfig(configFile, mockVCAPConfigurationReader)
			})

			Context("with invalid yaml", func() {
				BeforeEach(func() {
					configBytes = []byte(testhelpers.LoadFile("invalid.yml"))
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
					Expect(conf.CF.ClientID).To(Equal("client-id"))
					Expect(conf.CF.Secret).To(Equal("client-secret"))
					Expect(conf.CF.SkipSSLValidation).To(Equal(false))
					Expect(conf.Health.ServerConfig.Port).To(Equal(9999))
					Expect(conf.Logging.Level).To(Equal("debug"))

					Expect(conf.Db[db.AppMetricsDb]).To(Equal(db.DatabaseConfig{
						URL:                   "postgres://localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}), "appmetrics_db")
					Expect(conf.AppMetricsDb.RefreshInterval).To(Equal(10 * time.Hour))
					Expect(conf.AppMetricsDb.CutoffDuration).To(Equal(15 * time.Hour))

					Expect(conf.Db[db.ScalingEngineDb]).To(Equal(db.DatabaseConfig{
						URL:                   "postgres://localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
					Expect(conf.ScalingEngineDb.RefreshInterval).To(Equal(36 * time.Hour))
					Expect(conf.ScalingEngineDb.CutoffDuration).To(Equal(30 * time.Hour))

					Expect(conf.DBLock.LockTTL).To(Equal(15 * time.Second))
					Expect(conf.DBLock.LockRetryInterval).To(Equal(5 * time.Second))
					Expect(conf.Db[db.LockDb]).To(Equal(db.DatabaseConfig{
						URL:                   "postgres://localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))

					Expect(conf.Db[db.PolicyDb]).To(Equal(db.DatabaseConfig{
						URL:                   "postgres://localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
					Expect(conf.AppSyncer.SyncInterval).To(Equal(60 * time.Second))
					Expect(conf.HttpClientTimeout).To(Equal(10 * time.Second))
				})
			})

			Context("with partial config", func() {
				BeforeEach(func() {
					configBytes = []byte(testhelpers.LoadFile("partial.yml"))
				})

				It("returns default values", func() {
					Expect(err).NotTo(HaveOccurred())

					Expect(conf.Logging.Level).To(Equal(config.DefaultLoggingLevel))
					Expect(conf.Health.ServerConfig.Port).To(Equal(8081))
					Expect(conf.Db[db.AppMetricsDb]).To(Equal(db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}), "appmetrics_db")
					Expect(conf.AppMetricsDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
					Expect(conf.AppMetricsDb.CutoffDuration).To(Equal(config.DefaultCutoffDuration))
					Expect(conf.Db[db.ScalingEngineDb]).To(Equal(db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}), "scalingengine_db")
					Expect(conf.ScalingEngineDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
					Expect(conf.ScalingEngineDb.CutoffDuration).To(Equal(config.DefaultCutoffDuration))

					Expect(conf.ScalingEngine.SyncInterval).To(Equal(config.DefaultSyncInterval))
					Expect(conf.Scheduler.SyncInterval).To(Equal(config.DefaultSyncInterval))

					Expect(conf.DBLock.LockTTL).To(Equal(config.DefaultDBLockTTL))
					Expect(conf.DBLock.LockRetryInterval).To(Equal(config.DefaultDBLockRetryInterval))

					Expect(conf.AppSyncer.SyncInterval).To(Equal(config.DefaultSyncInterval))
					Expect(conf.Db[db.PolicyDb]).To(Equal(db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}), "policy_db")

					Expect(conf.HttpClientTimeout).To(Equal(5 * time.Second))

				})
			})

			When("scaling engine sync interval is not a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`
scaling_engine:
  sync_interval: 60kddd
`)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("failed to read config file")))
				})
			})

			When("scheduler sync interval is not a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`
scheduler:
  sync_interval: 60k
`)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("failed to read config file")))
				})
			})

			When("http_client_timeout of http is not a time duration", func() {
				BeforeEach(func() {
					configBytes = []byte(`http_client_timeout: 10k`)
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("failed to read config file")))
				})
			})

		})

		Describe("Validate", func() {
			BeforeEach(func() {
				conf = &config.Config{}

				conf.Db = make(map[string]db.DatabaseConfig)
				conf.Db[db.PolicyDb] = db.DatabaseConfig{
					URL: "postgres://pqgotest:password@exampl.com/pqgotest",
				}
				conf.Db[db.AppMetricsDb] = db.DatabaseConfig{
					URL: "postgres://pqgotest:password@exampl.com/pqgotest",
				}
				conf.Db[db.LockDb] = db.DatabaseConfig{
					URL: "postgres://pqgotest:password@exampl.com/pqgotest",
				}
				conf.Db[db.ScalingEngineDb] = db.DatabaseConfig{
					URL: "postgres://pqgotest:password@exampl.com/pqgotest",
				}
				conf.AppMetricsDb.RefreshInterval = 10 * time.Hour
				conf.AppMetricsDb.CutoffDuration = 15 * time.Hour

				conf.ScalingEngineDb.RefreshInterval = 36 * time.Hour
				conf.ScalingEngineDb.CutoffDuration = 20 * time.Hour

				conf.ScalingEngine.URL = "http://localhost:8082"
				conf.ScalingEngine.SyncInterval = 15 * time.Minute

				conf.Scheduler.URL = "http://localhost:8083"
				conf.Scheduler.SyncInterval = 15 * time.Minute

				conf.AppSyncer.SyncInterval = 60 * time.Second
				conf.HttpClientTimeout = 10 * time.Second
				conf.Health.ServerConfig.Port = 8081

			})

			JustBeforeEach(func() {
				err = conf.Validate()
			})

			When("all the configs are valid", func() {
				It("should not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			When("AppMetrics db url is not set", func() {

				BeforeEach(func() {
					conf.Db[db.AppMetricsDb] = db.DatabaseConfig{}

				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: app_metrics_db.db.url is empty"))
				})
			})

			When("ScalingEngine db url is not set", func() {

				BeforeEach(func() {
					conf.Db[db.ScalingEngineDb] = db.DatabaseConfig{}
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: scaling_engine_db.db.url is empty"))
				})
			})

			When("AppSyncer db url is not set", func() {

				BeforeEach(func() {
					conf.Db[db.PolicyDb] = db.DatabaseConfig{}
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: app_syncer.db.url is empty"))
				})
			})

			When("AppSyncer sync interval is set to a negative value", func() {

				BeforeEach(func() {
					conf.AppSyncer.SyncInterval = -1
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: appSyncer.sync_interval is less than or equal to 0"))
				})
			})

			When("AppMetrics db refresh interval in hours is set to a negative value", func() {

				BeforeEach(func() {
					conf.AppMetricsDb.RefreshInterval = -1
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: app_metrics_db.refresh_interval is less than or equal to 0"))
				})
			})

			When("ScalingEngine db refresh interval in hours is set to a negative value", func() {

				BeforeEach(func() {
					conf.ScalingEngineDb.RefreshInterval = -1
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: scaling_engine_db.refresh_interval is less than or equal to 0"))
				})
			})

			When("AppMetrics db cutoff duration is set to a negative value", func() {

				BeforeEach(func() {
					conf.AppMetricsDb.CutoffDuration = -1
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: app_metrics_db.cutoff_duration is less than or equal to 0"))
				})
			})

			When("ScalingEngine db cutoff duration is set to a negative value", func() {

				BeforeEach(func() {
					conf.ScalingEngineDb.CutoffDuration = -1
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: scaling_engine_db.cutoff_duration is less than or equal to 0"))
				})
			})

			When("ScalingEngine url is not set", func() {

				BeforeEach(func() {
					conf.ScalingEngine.URL = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: scaling_engine.scaling_engine_url is empty"))
				})
			})

			When("Scheduler url is not set", func() {

				BeforeEach(func() {
					conf.Scheduler.URL = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: scheduler.scheduler_url is empty"))
				})
			})

			When("ScalingEngine sync interval is set to 0", func() {

				BeforeEach(func() {
					conf.ScalingEngine.SyncInterval = 0
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: scaling_engine.sync_interval is less than or equal to 0"))
				})
			})

			When("Scheduler sync interval is set to 0", func() {

				BeforeEach(func() {
					conf.Scheduler.SyncInterval = 0
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: scheduler.sync_interval is less than or equal to 0"))
				})
			})

			When("db lockdb url is empty", func() {

				BeforeEach(func() {
					conf.Db[db.LockDb] = db.DatabaseConfig{}
				})

				It("should error", func() {
					Expect(err).To(MatchError("Configuration error: db_lock.db.url is empty"))
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
})
