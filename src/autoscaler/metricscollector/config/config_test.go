package config_test

import (
	"bytes"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"autoscaler/cf"
	. "autoscaler/metricscollector/config"
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
 cf:
  api: "https://api.exmaple.com"
  grant-type: "password"
  user: "admin"
server:
  port: 8989
`)
			})

			It("returns an error", func() {
				Expect(err).To(BeAssignableToTypeOf(&candiedyaml.ParserError{}))
			})
		})

		Context("when it gives a non integer port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: "https://api.exmaple.com"
  grant-type: "password"
  user: "admin"
server:
  port: "port"
logging:
  level: "info"
db:
  policy_db_url: "postgres://pqgotest:password@localhost/pqgotest" 
  metrics_db_url: "postgres://pqgotest:password@localhost/pqgotest" 
`)
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Invalid integer:.*")))
			})
		})

		Context("with valid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: "https://api.example.com"
  grant_type: "PassWord"
  username: "admin"
  password: "admin"
  client_id: "client-id"
  secret: "client-secret"
server:
  port: 8989
logging:
  level: "debug"
db:
  policy_db_url: "postgres://pqgotest:password@localhost/pqgotest" 
  metrics_db_url: "postgres://pqgotest:password@localhost/pqgotest" 
collector:
  refresh_interval_in_seconds: 20
  poll_interval_in_seconds: 10
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.Api).To(Equal("https://api.example.com"))
				Expect(conf.Cf.GrantType).To(Equal("password"))
				Expect(conf.Cf.Username).To(Equal("admin"))
				Expect(conf.Cf.Password).To(Equal("admin"))
				Expect(conf.Cf.ClientId).To(Equal("client-id"))
				Expect(conf.Cf.Secret).To(Equal("client-secret"))

				Expect(conf.Server.Port).To(Equal(8989))

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.Collector.RefreshInterval).To(Equal(20 * time.Second))
				Expect(conf.Collector.PollInterval).To(Equal(10 * time.Second))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: "https://api.example.com"
db:
  policy_db_url: "postgres://pqgotest:password@localhost/pqgotest" 
  metrics_db_url: "postgres://pqgotest:password@localhost/pqgotest" 
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.GrantType).To(Equal(cf.GrantTypePassword))
				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.Logging.Level).To(Equal(DefaultLoggingLevel))
				Expect(conf.Collector.RefreshInterval).To(Equal(DefaultRefreshInterval))
				Expect(conf.Collector.PollInterval).To(Equal(DefaultPollInterval))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Cf.Api = "http://api.example.com"
			conf.Cf.GrantType = cf.GrantTypePassword
			conf.Cf.Username = "admin"
			conf.Db.MetricsDbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.Db.PolicyDbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when all the configs are valid", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when cf config is not valid", func() {
			BeforeEach(func() {
				conf.Cf.Api = ""
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
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

		Context("when metrics db url is not set", func() {

			BeforeEach(func() {
				conf.Db.MetricsDbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Metrics DB url is empty")))
			})
		})

	})
})
