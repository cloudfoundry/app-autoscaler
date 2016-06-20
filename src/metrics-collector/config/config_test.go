package config_test

import (
	"bytes"
	. "metrics-collector/config"

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
			conf = &Config{
				Cf: CfConfig{},
			}
		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when grant type is not supprted", func() {
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
				conf.Cf.Username = "admin"
			})

			It("is valid", func() {
				Expect(err).NotTo(HaveOccurred())
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

			It("is valid", func() {
				Expect(err).NotTo(HaveOccurred())
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
	})
})
