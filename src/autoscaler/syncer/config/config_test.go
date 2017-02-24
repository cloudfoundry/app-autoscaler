package config_test

import (
	. "autoscaler/syncer/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"time"
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
  policy_db_url: "test-policy-db-url"
  scheduler_db_url: "test-scheduler-db-url"
synchronize_interval: 12h
`)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
			})
		})

		Context("with valid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
db:
  policy_db_url: "test-policy-db-url"
  scheduler_db_url: "test-scheduler-db-url"
synchronize_interval: 12h
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.Db.PolicyDbUrl).To(Equal("test-policy-db-url"))
				Expect(conf.Db.SchedulerDbUrl).To(Equal("test-scheduler-db-url"))

				Expect(conf.SynchronizeInterval).To(Equal(12 * time.Hour))

			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
db:
  policy_db_url: test-policy-db-url
  scheduler_db_url: test-scheduler-db-url
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("info"))

				Expect(conf.SynchronizeInterval).To(Equal(12 * time.Hour))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Db.PolicyDbUrl = "test-policy-db-url"
			conf.Db.SchedulerDbUrl = "test-scheduler-db-url"
			conf.SynchronizeInterval = 12 * time.Hour
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when all the configs are valid", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when policy db url is not set", func() {
			BeforeEach(func() {
				conf.Db.PolicyDbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Policy DB url is empty")))
			})
		})

		Context("when scheduler db url is not set", func() {
			BeforeEach(func() {
				conf.Db.SchedulerDbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Scheduler DB url is empty")))
			})
		})

		Context("when SynchronizeInterval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.SynchronizeInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: SynchronizeInterval is negative")))
			})
		})

	})

})
