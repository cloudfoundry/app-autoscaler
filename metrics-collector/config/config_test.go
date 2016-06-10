package config_test

import (
	. "github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
)

var _ = Describe("Config", func() {

	Describe("load config from yaml", func() {
		Context("when it gives an invalid yaml", func() {
			b := []byte(`
cf:
  api: "https://api.bosh-lite.com"
  grant-type: "password"
  user: "admin"
server:
  port: "port"
  user: "user"
  pass: "password"
logging:
  level: 0
  file: "logs/mc.log"
  log_to_stdout: "false"
`)

			It("should error", func() {
				_, err := LoadConfigFromYaml(b)
				Expect(err).To(HaveOccurred())
			})

		})

		Context("when it gives a valid yaml with all fields", func() {
			var b = []byte(`
cf:
  api: "https://api.bosh-lite.com"
  grant_type: "password"
  user: "admin"
  pass: "admin"
  client_id: "client-id"
  secret: "client-secret"
server:
  port: 8080
  user: "user"
  pass: "password"
logging:
  level: "DEBUG"
  file: "logs/mc.log"
  log_to_stdout: false
`)

			It("should not error and populate right config", func() {
				conf, err := LoadConfigFromYaml(b)

				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.Api).To(Equal("https://api.bosh-lite.com"))
				Expect(conf.Cf.GrantType).To(Equal("password"))
				Expect(conf.Cf.User).To(Equal("admin"))
				Expect(conf.Cf.Pass).To(Equal("admin"))
				Expect(conf.Cf.ClientId).To(Equal("client-id"))
				Expect(conf.Cf.Secret).To(Equal("client-secret"))

				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.Server.User).To(Equal("user"))
				Expect(conf.Server.Pass).To(Equal("password"))

				Expect(conf.Logging.Level).To(Equal("DEBUG"))
				Expect(conf.Logging.File).To(Equal("logs/mc.log"))
				Expect(conf.Logging.LogToStdout).To(Equal(false))
			})
		})

		Context("when it gives a valid yaml with part of the fields", func() {
			var b = []byte(`
cf:
  api: "https://api.bosh-lite.com"
  grant_type: "password"
  user: "admin"
  pass: "admin"
server:
  user: "user"
  pass: "password"
logging:
  file: "logs/mc.log"
`)

			It("should not error and  populate the right config", func() {
				defaultConfig := DefaultConfig()
				conf, err := LoadConfigFromYaml(b)

				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.Api).To(Equal("https://api.bosh-lite.com"))
				Expect(conf.Cf.GrantType).To(Equal("password"))
				Expect(conf.Cf.User).To(Equal("admin"))
				Expect(conf.Cf.Pass).To(Equal("admin"))
				Expect(conf.Cf.ClientId).To(Equal(defaultConfig.Cf.ClientId))
				Expect(conf.Cf.Secret).To(Equal(defaultConfig.Cf.Secret))

				Expect(conf.Server.Port).To(Equal(defaultConfig.Server.Port))
				Expect(conf.Server.User).To(Equal("user"))
				Expect(conf.Server.Pass).To(Equal("password"))

				Expect(conf.Logging.Level).To(Equal(defaultConfig.Logging.Level))
				Expect(conf.Logging.File).To(Equal("logs/mc.log"))
				Expect(conf.Logging.LogToStdout).To(Equal(defaultConfig.Logging.LogToStdout))
			})
		})

	})

	Describe("load config from file", func() {

		Context("when configuration file does not exist", func() {
			var path = "not_exist.yml"

			It("should error and return nil config", func() {
				conf, err := LoadConfigFromFile(path)
				Expect(err).To(HaveOccurred())
				Expect(conf).To(BeNil())
			})
		})

		Context("when configuration file exists and is valid", func() {
			var path = "exist_conf.yml"

			BeforeEach(func() {
				file, _ := os.Create(path)
				var b = []byte(`
cf:
  api: "https://api.bosh-lite.com"
  grant_type: "password"
  user: "admin"
  pass: "admin"
  client_id: "client-id"
  secret: "client-secret"
server:
  port: 8080
  user: "user"
  pass: "password"
logging:
  level: "DEBUG"
  file: "logs/mc.log"
  log_to_stdout: false
`)
				file.Write(b)
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
})
