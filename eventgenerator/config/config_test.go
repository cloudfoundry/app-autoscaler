package config_test

import (
	. "autoscaler/eventgenerator/config"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	var (
		conf        *Config
		err         error
		configBytes []byte
	)

	Describe("LoadConfig", func() {

		JustBeforeEach(func() {
			conf, err = LoadConfig(configBytes)
		})

		Context("with valid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
log_level: "debug"
policy_db_url: "postgres://postgres:password@localhost/autoscaler"
appmetric_db_url: "postgres://postgres:password@localhost/autoscaler"
poll_interval: 30
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf).To(Equal(&Config{LogLevel: "debug", PollInterval: 30, PolicyDbUrl: "postgres://postgres:password@localhost/autoscaler", AppMetricDbUrl: "postgres://postgres:password@localhost/autoscaler"}))
			})
		})
		Context("with invalid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
  log_level: "debug"
 policy_db_url: "postgres://postgres:password@localhost/autoscaler"
 appmetric_db_url: "postgres://postgres:password@localhost/autoscaler"
 poll_interval: 30
		`)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(MatchRegexp(".*did not find expected <document start>.*")))
			})
		})

		Context("when it gives a non integer poll interval", func() {
			BeforeEach(func() {
				configBytes = []byte(`
log_level: "debug"
policy_db_url: "postgres://postgres:password@localhost/autoscaler"
appmetric_db_url: "postgres://postgres:password@localhost/autoscaler"
poll_interval: "NotIntegerValue"
`)
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp(".*into time.Duration")))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
log_level: "debug"
policy_db_url: "postgres://postgres:password@localhost/autoscaler"
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(conf.PollInterval).To(Equal(time.Duration(DefaultPollInterval)))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.PolicyDbUrl = "postgres://postgres:password@localhost/autoscaler"
			conf.AppMetricDbUrl = "postgres://postgres:password@localhost/autoscaler"
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when policy db url is not set", func() {

			BeforeEach(func() {
				conf.PolicyDbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Policy DB url is empty")))
			})
		})
		Context("when appmetric db url is not set", func() {

			BeforeEach(func() {
				conf.AppMetricDbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: AppMetric DB url is empty")))
			})
		})
		Context("when poll interval is le than 0", func() {

			BeforeEach(func() {
				conf.PollInterval = 0
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Poll Interval is le than 0")))
			})
		})

	})
})
