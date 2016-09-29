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
pruner:
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
pruner:
  cutoff_days: "cutoff_days"
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
pruner:
  interval_in_hours: 12
  cutoff_days: 20
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.Db.MetricsDbUrl).To(Equal("postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"))

				Expect(conf.Pruner.IntervalInHours).To(Equal(12))
				Expect(conf.Pruner.CutoffDays).To(Equal(20))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
db:
  metrics_db_url: "postgres://pqgotest:password@localhost/pqgotest" 
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal(DefaultLoggingLevel))

				Expect(conf.Pruner.IntervalInHours).To(Equal(DefaultIntervalInHours))
				Expect(conf.Pruner.CutoffDays).To(Equal(DefaultCutoffDays))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Db.MetricsDbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.Pruner.IntervalInHours = 12
			conf.Pruner.CutoffDays = 30
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

		Context("when interval in hours is set negative value", func() {

			BeforeEach(func() {
				conf.Pruner.IntervalInHours = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Interval in hours is negative")))
			})
		})

		Context("when cutoff days is set negative value", func() {

			BeforeEach(func() {
				conf.Pruner.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Cutoff days is negative")))
			})
		})
	})
})
