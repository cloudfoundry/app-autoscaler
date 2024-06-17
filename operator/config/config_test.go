package config_test

import (
	"bytes"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Config", func() {

	var (
		conf        *config.Config
		err         error
		configBytes []byte
	)

	Describe("LoadConfig", func() {

		JustBeforeEach(func() {
			conf, err = config.LoadConfig(bytes.NewReader(configBytes))
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
				Expect(conf.Health.Port).To(Equal(9999))
				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.AppMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.AppMetricsDB.RefreshInterval).To(Equal(10 * time.Hour))
				Expect(conf.AppMetricsDB.CutoffDuration).To(Equal(15 * time.Hour))

				Expect(conf.ScalingEngineDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.ScalingEngineDB.RefreshInterval).To(Equal(36 * time.Hour))
				Expect(conf.ScalingEngineDB.CutoffDuration).To(Equal(30 * time.Hour))

				Expect(conf.DBLock.LockTTL).To(Equal(15 * time.Second))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(5 * time.Second))
				Expect(conf.DBLock.DB.URL).To(Equal("postgres://postgres:password@localhost/autoscaler?sslmode=disable"))

				Expect(conf.AppSyncer.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
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
				Expect(conf.Health.Port).To(Equal(8081))
				Expect(conf.AppMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.AppMetricsDB.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.AppMetricsDB.CutoffDuration).To(Equal(config.DefaultCutoffDuration))
				Expect(conf.ScalingEngineDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.ScalingEngineDB.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.ScalingEngineDB.CutoffDuration).To(Equal(config.DefaultCutoffDuration))

				Expect(conf.ScalingEngine.SyncInterval).To(Equal(config.DefaultSyncInterval))
				Expect(conf.Scheduler.SyncInterval).To(Equal(config.DefaultSyncInterval))

				Expect(conf.DBLock.LockTTL).To(Equal(config.DefaultDBLockTTL))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(config.DefaultDBLockRetryInterval))

				Expect(conf.AppSyncer.SyncInterval).To(Equal(config.DefaultSyncInterval))
				Expect(conf.AppSyncer.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))

				Expect(conf.HttpClientTimeout).To(Equal(5 * time.Second))

			})
		})

		Context("when scaling engine sync interval is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
scaling_engine:
  sync_interval: 60kddd
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when scheduler sync interval is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
scheduler:
  sync_interval: 60k
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when http_client_timeout of http is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`http_client_timeout: 10k`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &config.Config{}

			conf.AppMetricsDB.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.AppMetricsDB.RefreshInterval = 10 * time.Hour
			conf.AppMetricsDB.CutoffDuration = 15 * time.Hour

			conf.ScalingEngineDB.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.ScalingEngineDB.RefreshInterval = 36 * time.Hour
			conf.ScalingEngineDB.CutoffDuration = 20 * time.Hour

			conf.ScalingEngine.URL = "http://localhost:8082"
			conf.ScalingEngine.SyncInterval = 15 * time.Minute

			conf.Scheduler.URL = "http://localhost:8083"
			conf.Scheduler.SyncInterval = 15 * time.Minute

			conf.AppSyncer.SyncInterval = 60 * time.Second
			conf.AppSyncer.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.DBLock.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.HttpClientTimeout = 10 * time.Second
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

		Context("when AppMetrics db url is not set", func() {

			BeforeEach(func() {
				conf.AppMetricsDB.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_metrics_db.db.url is empty"))
			})
		})

		Context("when ScalingEngine db url is not set", func() {

			BeforeEach(func() {
				conf.ScalingEngineDB.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine_db.db.url is empty"))
			})
		})

		Context("when AppSyncer db url is not set", func() {

			BeforeEach(func() {
				conf.AppSyncer.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: appSyncer.db.url is empty"))
			})
		})

		Context("when AppSyncer sync interval is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppSyncer.SyncInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: appSyncer.sync_interval is less than or equal to 0"))
			})
		})

		Context("when AppMetrics db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDB.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_metrics_db.refresh_interval is less than or equal to 0"))
			})
		})

		Context("when ScalingEngine db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.ScalingEngineDB.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine_db.refresh_interval is less than or equal to 0"))
			})
		})

		Context("when AppMetrics db cutoff duration is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDB.CutoffDuration = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_metrics_db.cutoff_duration is less than or equal to 0"))
			})
		})

		Context("when ScalingEngine db cutoff duration is set to a negative value", func() {

			BeforeEach(func() {
				conf.ScalingEngineDB.CutoffDuration = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine_db.cutoff_duration is less than or equal to 0"))
			})
		})

		Context("when ScalingEngine url is not set", func() {

			BeforeEach(func() {
				conf.ScalingEngine.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine.scaling_engine_url is empty"))
			})
		})

		Context("when Scheduler url is not set", func() {

			BeforeEach(func() {
				conf.Scheduler.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scheduler.scheduler_url is empty"))
			})
		})

		Context("when ScalingEngine sync interval is set to 0", func() {

			BeforeEach(func() {
				conf.ScalingEngine.SyncInterval = 0
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine.sync_interval is less than or equal to 0"))
			})
		})

		Context("when Scheduler sync interval is set to 0", func() {

			BeforeEach(func() {
				conf.Scheduler.SyncInterval = 0
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scheduler.sync_interval is less than or equal to 0"))
			})
		})

		Context("when db lockdb url is empty", func() {

			BeforeEach(func() {
				conf.DBLock.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: db_lock.db.url is empty"))
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

	})
})
