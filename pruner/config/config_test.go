package config_test

import (
	"bytes"
	"time"

	"autoscaler/pruner/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
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
				configBytes = []byte(`
 logging:
  level: "debug"
instance_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 12h
  cutoff_days: 10
app_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 24h
  cutoff_days: 20
scaling_engine_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 36h
  cutoff_days: 30
`)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
			})
		})

		Context("when it gives a non integer cutoff_days", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 12h
  cutoff_days: 10
app_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 24h
  cutoff_days: 20
scaling_engine_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 36h
  cutoff_days: 30s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
			})
		})

		Context("with valid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 36h
  cutoff_days: 30
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.InstanceMetricsDb.DbUrl).To(Equal("postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"))
				Expect(conf.InstanceMetricsDb.RefreshInterval).To(Equal(12 * time.Hour))
				Expect(conf.InstanceMetricsDb.CutoffDays).To(Equal(20))

				Expect(conf.AppMetricsDb.DbUrl).To(Equal("postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"))
				Expect(conf.AppMetricsDb.RefreshInterval).To(Equal(10 * time.Hour))
				Expect(conf.AppMetricsDb.CutoffDays).To(Equal(15))

				Expect(conf.ScalingEngineDb.DbUrl).To(Equal("postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"))
				Expect(conf.ScalingEngineDb.RefreshInterval).To(Equal(36 * time.Hour))
				Expect(conf.ScalingEngineDb.CutoffDays).To(Equal(30))

			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
instance_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
app_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
scaling_engine_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"  
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal(config.DefaultLoggingLevel))

				Expect(conf.InstanceMetricsDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.InstanceMetricsDb.CutoffDays).To(Equal(config.DefaultCutoffDays))

				Expect(conf.AppMetricsDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.AppMetricsDb.CutoffDays).To(Equal(config.DefaultCutoffDays))

				Expect(conf.ScalingEngineDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.ScalingEngineDb.CutoffDays).To(Equal(config.DefaultCutoffDays))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &config.Config{}

			conf.InstanceMetricsDb.DbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.InstanceMetricsDb.RefreshInterval = 12 * time.Hour
			conf.InstanceMetricsDb.CutoffDays = 30

			conf.AppMetricsDb.DbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.AppMetricsDb.RefreshInterval = 10 * time.Hour
			conf.AppMetricsDb.CutoffDays = 15

			conf.ScalingEngineDb.DbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.ScalingEngineDb.RefreshInterval = 36 * time.Hour
			conf.ScalingEngineDb.CutoffDays = 20

		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when all the configs are valid", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when InstanceMetrics db url is not set", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDb.DbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: InstanceMetrics DB url is empty")))
			})
		})

		Context("when AppMetrics db url is not set", func() {

			BeforeEach(func() {
				conf.AppMetricsDb.DbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: AppMetrics DB url is empty")))
			})
		})

		Context("when ScalingEngine db url is not set", func() {

			BeforeEach(func() {
				conf.ScalingEngineDb.DbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: ScalingEngine DB url is empty")))
			})
		})

		Context("when InstanceMetrics db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDb.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: InstanceMetrics DB refresh interval is negative")))
			})
		})

		Context("when AppMetrics db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDb.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: AppMetrics DB refresh interval is negative")))
			})
		})

		Context("when ScalingEngine db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.ScalingEngineDb.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: ScalingEngine DB refresh interval is negative")))
			})
		})

		Context("when InstanceMetrics db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDb.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: InstanceMetrics DB cutoff days is negative")))
			})
		})

		Context("when AppMetrics db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDb.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: AppMetrics DB cutoff days is negative")))
			})
		})

		Context("when ScalingEngine db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.ScalingEngineDb.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: ScalingEngine DB cutoff days is negative")))
			})
		})

	})
})
