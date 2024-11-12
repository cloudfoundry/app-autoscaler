package config_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/config"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/testhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

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
				configBytes = []byte(LoadFile("invalid.txt"))
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
			})
		})

		Context("with valid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(LoadFile("valid.yml"))
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.CF.API).To(Equal("https://api.example.com"))
				Expect(conf.CF.ClientID).To(Equal("autoscaler_client_id"))
				Expect(conf.CF.Secret).To(Equal("autoscaler_client_secret"))
				Expect(conf.CF.SkipSSLValidation).To(Equal(false))

				Expect(conf.Server.Port).To(Equal(8989))
				Expect(conf.Server.TLS.KeyFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.key"))
				Expect(conf.Server.TLS.CertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.crt"))
				Expect(conf.Server.TLS.CACertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/ca.crt"))

				Expect(conf.Health.ServerConfig.Port).To(Equal(9999))
				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.DB.PolicyDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.DB.ScalingEngineDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.DB.SchedulerDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))

				Expect(conf.DefaultCoolDownSecs).To(Equal(300))

				Expect(conf.LockSize).To(Equal(32))

				Expect(conf.HttpClientTimeout).To(Equal(10 * time.Second))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(LoadFile("partial.yml"))
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.CF.SkipSSLValidation).To(Equal(false))
				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.Health.ServerConfig.Port).To(Equal(8081))
				Expect(conf.Logging.Level).To(Equal("info"))
				Expect(conf.DB.PolicyDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.DB.ScalingEngineDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.DB.SchedulerDB).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))

				Expect(conf.HttpClientTimeout).To(Equal(5 * time.Second))
			})
		})

		Context("when it gives a non integer server port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: port
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when it gives a non integer health port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
health:
  server_config:
    port: port
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when it gives a non integer of defaultCoolDownSecs", func() {
			BeforeEach(func() {
				configBytes = []byte(`defaultCoolDownSecs: NOT-INTEGER-VALUE`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when it gives a non integer of lockSize", func() {
			BeforeEach(func() {
				configBytes = []byte(`lockSize: NOT-INTEGER-VALUE`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
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
			conf = &Config{}
			conf.CF.API = "http://api.example.com"
			conf.CF.SkipSSLValidation = false
			conf.CF.ClientID = "autoscaler_client_id"
			conf.DB.PolicyDB.URL = "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
			conf.DB.ScalingEngineDB.URL = "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
			conf.DB.SchedulerDB.URL = "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
			conf.DefaultCoolDownSecs = 300
			conf.LockSize = 32
			conf.HttpClientTimeout = 10 * time.Second
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
				conf.CF.API = ""
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when policy db url is not set", func() {
			BeforeEach(func() {
				conf.DB.PolicyDB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: db.policy_db.url is empty"))
			})
		})

		Context("when scalingengine db url is not set", func() {
			BeforeEach(func() {
				conf.DB.ScalingEngineDB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: db.scalingengine_db.url is empty"))
			})
		})

		Context("when scheduler db url is not set", func() {
			BeforeEach(func() {
				conf.DB.SchedulerDB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: db.scheduler_db.url is empty"))
			})
		})

		Context("when DefaultCoolDownSecs < 60", func() {
			BeforeEach(func() {
				conf.DefaultCoolDownSecs = 10
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: DefaultCoolDownSecs should be between 60 and 3600"))
			})
		})

		Context("when DefaultCoolDownSecs > 3600", func() {
			BeforeEach(func() {
				conf.DefaultCoolDownSecs = 5000
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: DefaultCoolDownSecs should be between 60 and 3600"))
			})
		})

		Context("when LockSize <= 0", func() {
			BeforeEach(func() {
				conf.LockSize = 0
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: LockSize is less than or equal to 0"))
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
