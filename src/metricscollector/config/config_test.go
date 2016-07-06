package config_test

import (
	"bytes"
	. "metricscollector/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/candiedyaml"
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
  user: "user"
  pass: "password"
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

				Expect(conf.Cf.GrantType).To(Equal(GrantTypePassword))
				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.Logging.Level).To(Equal(DefaultLoggingLevel))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Cf.Api = "http://api.example.com"
			conf.Cf.GrantType = GrantTypePassword
			conf.Cf.Username = "admin"
			conf.Cf.ClientId = "admin"
			conf.Db.MetricsDbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.Db.PolicyDbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when api is not set", func() {
			BeforeEach(func() {
				conf.Cf.Api = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: cf api is empty")))
			})
		})

		Context("when grant type is not supported", func() {
			BeforeEach(func() {
				conf.Cf.GrantType = "not-supported"
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: unsupported grant type*")))
			})
		})

		Context("when grant type password", func() {
			BeforeEach(func() {
				conf.Cf.GrantType = GrantTypePassword
			})

			Context("when user name is set", func() {
				BeforeEach(func() {
					conf.Cf.ClientId = ""
					conf.Cf.Username = "admin"
				})
				It("is valid", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when the user name is empty", func() {
				BeforeEach(func() {
					conf.Cf.Username = ""
				})

				It("should error", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: user name is empty")))
				})
			})
		})

		Context("when grant type client_credential", func() {
			BeforeEach(func() {
				conf.Cf.GrantType = GrantTypeClientCredentials
				conf.Cf.ClientId = "admin"
			})

			Context("when client id is set", func() {
				BeforeEach(func() {
					conf.Cf.ClientId = "admin"
					conf.Cf.Username = ""
				})
				It("is valid", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("the client id is empty", func() {
				BeforeEach(func() {
					conf.Cf.ClientId = ""
				})

				It("returns error", func() {
					Expect(err).To(MatchError(MatchRegexp("Configuration error: client id is empty")))
				})
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
