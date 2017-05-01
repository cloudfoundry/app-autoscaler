package config_test

import (
	"autoscaler/cf"
	. "autoscaler/scalingengine/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

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
 cf:
  api: https://api.exmaple.com
  grant-type: password
  user: admin
server:
  port: 8989
consul:
  cluster: http://127.0.0.1:8500
`)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
			})
		})

		Context("when it gives a non integer port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: port
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
			})
		})

		Context("with valid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DeBug
db:
  policy_db_url: test-policy-db-url
  scalingengine_db_url: test-scalingengine-db-url
  scheduler_db_url: test-scheduler-db-url
synchronizer:
  active_schedule_sync_interval: 300s
consul:
  cluster: http://127.0.0.1:8500
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
				Expect(conf.Server.TLS.KeyFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.key"))
				Expect(conf.Server.TLS.CertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.crt"))
				Expect(conf.Server.TLS.CACertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/ca.crt"))

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.Db.PolicyDbUrl).To(Equal("test-policy-db-url"))
				Expect(conf.Db.ScalingEngineDbUrl).To(Equal("test-scalingengine-db-url"))
				Expect(conf.Db.SchedulerDbUrl).To(Equal("test-scheduler-db-url"))

				Expect(conf.Synchronizer.ActiveScheduleSyncInterval).To(Equal(5 * time.Minute))

				Expect(conf.Consul.Cluster).To(Equal("http://127.0.0.1:8500"))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
db:
  policy_db_url: test-policy-db-url
  scalingengine_db_url: test-scalingengine-db-url
  scheduler_db_url: test-scheduler-db-url
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.GrantType).To(Equal(cf.GrantTypePassword))
				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.Logging.Level).To(Equal("info"))
				Expect(conf.Synchronizer.ActiveScheduleSyncInterval).To(Equal(DefaultActiveScheduleSyncInterval))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Cf.Api = "http://api.example.com"
			conf.Cf.GrantType = cf.GrantTypePassword
			conf.Cf.Username = "admin"
			conf.Db.PolicyDbUrl = "test-policy-db-url"
			conf.Db.ScalingEngineDbUrl = "test-scalingengine-db-url"
			conf.Db.SchedulerDbUrl = "test-scheduler-db-url"
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

		Context("when scalingengine db url is not set", func() {
			BeforeEach(func() {
				conf.Db.ScalingEngineDbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: ScalingEngine DB url is empty")))
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
	})

})
