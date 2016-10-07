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
metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 12h
  cutoff_days: 30
app_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 12h
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
metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 12h
  cutoff_days: "cutoff_days"
app_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 12h
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
metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  refresh_interval: 10h
  cutoff_days: 15
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.MetricsDb.DbUrl).To(Equal("postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"))
				Expect(conf.AppMetricsDb.DbUrl).To(Equal("postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"))

				Expect(conf.MetricsDb.RefreshInterval).To(Equal(12 * time.Hour))
				Expect(conf.MetricsDb.CutoffDays).To(Equal(20))

				Expect(conf.AppMetricsDb.RefreshInterval).To(Equal(10 * time.Hour))
				Expect(conf.AppMetricsDb.CutoffDays).To(Equal(15))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
app_metrics_db:
  db_url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal(config.DefaultLoggingLevel))

				Expect(conf.MetricsDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.MetricsDb.CutoffDays).To(Equal(config.DefaultCutoffDays))

				Expect(conf.AppMetricsDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.AppMetricsDb.CutoffDays).To(Equal(config.DefaultCutoffDays))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &config.Config{}
			conf.MetricsDb.DbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.AppMetricsDb.DbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.MetricsDb.RefreshInterval = 12 * time.Hour
			conf.MetricsDb.CutoffDays = 30
			conf.AppMetricsDb.RefreshInterval = 10 * time.Hour
			conf.AppMetricsDb.CutoffDays = 15
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
				conf.MetricsDb.DbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Metrics DB url is empty")))
			})
		})

		Context("when app metrics db url is not set", func() {

			BeforeEach(func() {
				conf.AppMetricsDb.DbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: App Metrics DB url is empty")))
			})
		})

		Context("when metrics db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.MetricsDb.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Metrics DB refresh interval is negative")))
			})
		})

		Context("when app metrics db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDb.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: App Metrics DB refresh interval is negative")))
			})
		})

		Context("when metrics db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.MetricsDb.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Metrics DB cutoff days is negative")))
			})
		})

		Context("when app metrics db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDb.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: App Metrics DB cutoff days is negative")))
			})
		})
	})
})
