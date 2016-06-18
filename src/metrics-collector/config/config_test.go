package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "metrics-collector/config"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"os"
)

var _ = Describe("Config", func() {

	var (
		conf  *Config
		err   error
		bytes []byte
		path  string
	)

	Describe("LoadConfigFromYaml", func() {

		JustBeforeEach(func() {
			conf, err = LoadConfigFromYaml(bytes)
		})

		Context("when it gives an invalid yml", func() {
			BeforeEach(func() {
				bytes = []byte(`
 cf:
  api: "https://api.exmaple.com"
  grant-type: "password"
  user: "admin"
server:
  port: 8989
  user: "user"
  pass: "password"

`)
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&candiedyaml.ParserError{}))
			})

		})

		Context("when it gives a non integer port", func() {
			BeforeEach(func() {
				bytes = []byte(`
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
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("Invalid integer:.*")))
			})

		})

		Context("when it gives a valid yaml with all fields", func() {
			BeforeEach(func() {
				bytes = []byte(`
cf:
  api: "https://api.example.com"
  grant_type: "PassWord"
  user: "admin"
  pass: "admin"
  client_id: "client-id"
  secret: "client-secret"
server:
  port: 8989
  user: "user"
  pass: "password"
logging:
  level: "debug"
`)
			})

			It("should not error and populate right config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.Api).To(Equal("https://api.example.com"))
				Expect(conf.Cf.GrantType).To(Equal("password"))
				Expect(conf.Cf.User).To(Equal("admin"))
				Expect(conf.Cf.Pass).To(Equal("admin"))
				Expect(conf.Cf.ClientId).To(Equal("client-id"))
				Expect(conf.Cf.Secret).To(Equal("client-secret"))

				Expect(conf.Server.Port).To(Equal(8989))
				Expect(conf.Server.User).To(Equal("user"))
				Expect(conf.Server.Pass).To(Equal("password"))

				Expect(conf.Logging.Level).To(Equal("debug"))
			})
		})

		Context("when it gives a valid yaml with part of the fields", func() {
			BeforeEach(func() {
				bytes = []byte(`
cf:
  api: "https://api.example.com"
  user: "admin"
  pass: "admin"
server:
  user: "user"
  pass: "password"
`)
			})

			It("should not error and  populate the right config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.Api).To(Equal("https://api.example.com"))
				Expect(conf.Cf.GrantType).To(Equal(DefaultCfConfig.GrantType))
				Expect(conf.Cf.User).To(Equal("admin"))
				Expect(conf.Cf.Pass).To(Equal("admin"))
				Expect(conf.Cf.ClientId).To(Equal(DefaultCfConfig.ClientId))
				Expect(conf.Cf.Secret).To(Equal(DefaultCfConfig.Secret))

				Expect(conf.Server.Port).To(Equal(DefaultServerConfig.Port))
				Expect(conf.Server.User).To(Equal("user"))
				Expect(conf.Server.Pass).To(Equal("password"))

				Expect(conf.Logging.Level).To(Equal(DefaultLoggingConfig.Level))
			})
		})

	})

	Describe("LoadConfigFromFile", func() {

		JustBeforeEach(func() {
			conf, err = LoadConfigFromFile(path)
		})

		Context("when configuration file does not exist", func() {
			BeforeEach(func() {
				path = "not_exist.yml"
			})

			It("should error and return nil config", func() {

				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
				Expect(conf).To(BeNil())
			})
		})

		Context("when configuration file exists and is valid", func() {
			BeforeEach(func() {
				path = "does_exist.yml"
				file, _ := os.Create(path)
				bytes = []byte(`
cf:
  api: "https://api.example.com"
  grant_type: "PassWord"
  user: "admin"
  pass: "admin"
  client_id: "client-id"
  secret: "client-secret"
server:
  port: 8989
  user: "user"
  pass: "password"
logging:
  level: "info"
`)
				file.Write(bytes)
				file.Close()
			})

			AfterEach(func() {
				os.Remove(path)
			})

			It("should not error, and return the config", func() {
				conf, err := LoadConfigFromFile(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(conf).NotTo(BeNil())
			})
		})
	})

	Describe("Verify", func() {
		JustBeforeEach(func() {
			err = conf.Verify()
		})

		Context("when grant type is not supprted", func() {
			BeforeEach(func() {
				conf = &Config{
					Cf:      DefaultCfConfig,
					Logging: DefaultLoggingConfig,
					Server:  DefaultServerConfig,
				}

				conf.Cf.GrantType = "not-supported-grant-type"
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("Error in configuration file: unsupported grant type*")))
			})
		})

		Context("when grant type password but user name is empty", func() {
			BeforeEach(func() {
				conf = &Config{
					Cf:      DefaultCfConfig,
					Logging: DefaultLoggingConfig,
					Server:  DefaultServerConfig,
				}
				conf.Cf.GrantType = GRANT_TYPE_PASSWORD
				conf.Cf.User = ""
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("Error in configuration file: user name is empty")))
			})
		})

		Context("when grant type client_credential but client id is empty", func() {
			BeforeEach(func() {
				conf = &Config{
					Cf:      DefaultCfConfig,
					Logging: DefaultLoggingConfig,
					Server:  DefaultServerConfig,
				}
				conf.Cf.GrantType = GRANT_TYPE_CLIENT_CREDENTIALS
				conf.Cf.ClientId = ""
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("Error in configuration file: client id is empty")))
			})
		})

		Context("when config is valid with password grant", func() {
			BeforeEach(func() {
				conf = &Config{
					Cf:      DefaultCfConfig,
					Logging: DefaultLoggingConfig,
					Server:  DefaultServerConfig,
				}
				conf.Cf.GrantType = GRANT_TYPE_PASSWORD
				conf.Cf.User = "admin"
				conf.Logging.Level = "debug"
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when config is valid with client_credential grant ", func() {
			BeforeEach(func() {
				conf = &Config{
					Cf:      DefaultCfConfig,
					Logging: DefaultLoggingConfig,
					Server:  DefaultServerConfig,
				}
				conf.Cf.GrantType = GRANT_TYPE_CLIENT_CREDENTIALS
				conf.Cf.ClientId = "admin"
				conf.Logging.Level = "error"
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})

})
