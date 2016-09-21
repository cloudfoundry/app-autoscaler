package config_test

import (
	"bytes"
	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "autoscaler/pruner/config"
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
 logging:
  level: "debug"
db:
  metrics_db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  app_metrics_db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
pruner:
  metrics_db:
    refresh_interval_in_hours: 12
    cutoff_days: 30
  app_metrics_db:
    refresh_interval_in_hours: 12
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
db:
  metrics_db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  app_metrics_db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
pruner:
  metrics_db:
    refresh_interval_in_hours: 12
    cutoff_days: "cutoff_days"
  app_metrics_db:
    refresh_interval_in_hours: 12
    cutoff_days: 30
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
db:
  metrics_db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  app_metrics_db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
pruner:
  metrics_db:
    refresh_interval_in_hours: 12
    cutoff_days: 20
  app_metrics_db:
    refresh_interval_in_hours: 10
    cutoff_days: 15
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.Db.MetricsDbUrl).To(Equal("postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"))
				Expect(conf.Db.AppMetricsDbUrl).To(Equal("postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"))

				Expect(conf.Pruner.MetricsDbPruner.RefreshIntervalInHours).To(Equal(12))
				Expect(conf.Pruner.MetricsDbPruner.CutoffDays).To(Equal(20))

				Expect(conf.Pruner.AppMetricsDbPruner.RefreshIntervalInHours).To(Equal(10))
				Expect(conf.Pruner.AppMetricsDbPruner.CutoffDays).To(Equal(15))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
db:
  metrics_db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable" 
  app_metrics_db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal(DefaultLoggingLevel))

				Expect(conf.Pruner.MetricsDbPruner.RefreshIntervalInHours).To(Equal(DefaultRefreshIntervalInHours))
				Expect(conf.Pruner.MetricsDbPruner.CutoffDays).To(Equal(DefaultCutoffDays))

				Expect(conf.Pruner.AppMetricsDbPruner.RefreshIntervalInHours).To(Equal(DefaultRefreshIntervalInHours))
				Expect(conf.Pruner.AppMetricsDbPruner.CutoffDays).To(Equal(DefaultCutoffDays))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Db.MetricsDbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.Db.AppMetricsDbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.Pruner.MetricsDbPruner.RefreshIntervalInHours = 12
			conf.Pruner.MetricsDbPruner.CutoffDays = 30
			conf.Pruner.AppMetricsDbPruner.RefreshIntervalInHours = 10
			conf.Pruner.AppMetricsDbPruner.CutoffDays = 15
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when all the configs are valid", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when metrics db url is not set", func() {

			BeforeEach(func() {
				conf.Db.MetricsDbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Metrics DB url is empty")))
			})
		})

		Context("when app metrics db url is not set", func() {

			BeforeEach(func() {
				conf.Db.AppMetricsDbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: App Metrics DB url is empty")))
			})
		})

		Context("when metrics db refresh interval in hours is set negative value", func() {

			BeforeEach(func() {
				conf.Pruner.MetricsDbPruner.RefreshIntervalInHours = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Metrics DB refresh interval in hours is negative")))
			})
		})

		Context("when app metrics db refresh interval in hours is set negative value", func() {

			BeforeEach(func() {
				conf.Pruner.AppMetricsDbPruner.RefreshIntervalInHours = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: App Metrics DB refresh interval in hours is negative")))
			})
		})

		Context("when metrics db cutoff days is set negative value", func() {

			BeforeEach(func() {
				conf.Pruner.MetricsDbPruner.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Metrics DB cutoff days is negative")))
			})
		})

		Context("when app metrics db cutoff days is set negative value", func() {

			BeforeEach(func() {
				conf.Pruner.AppMetricsDbPruner.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: App Metrics DB cutoff days is negative")))
			})
		})
	})
})
