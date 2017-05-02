package config_test

import (
	"bytes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

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
  api: https://api.exmaple.com
  grant-type: password
  user: admin
server:
  port: 8989
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

		Context("when it gives an invalid time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
collector:
  refresh_interval: 20a
  collect_interval: 10s  
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
  level: DebuG
db:
  policy_db_url: postgres://pqgotest:password@localhost/pqgotest
  instance_metrics_db_url: postgres://pqgotest:password@localhost/pqgotest
collector:
  refresh_interval: 20s
  collect_interval: 10s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
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

				Expect(conf.Db.PolicyDbUrl).To(Equal("postgres://pqgotest:password@localhost/pqgotest"))
				Expect(conf.Db.InstanceMetricsDbUrl).To(Equal("postgres://pqgotest:password@localhost/pqgotest"))

				Expect(conf.Collector.RefreshInterval).To(Equal(20 * time.Second))
				Expect(conf.Collector.CollectInterval).To(Equal(10 * time.Second))

				Expect(conf.Server.TLS.KeyFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.key"))
				Expect(conf.Server.TLS.CertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.crt"))
				Expect(conf.Server.TLS.CACertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/ca.crt"))

				Expect(conf.Lock.ConsulClusterConfig).To(Equal("http://127.0.0.1:8500"))
				Expect(conf.Lock.LockRetryInterval).To(Equal(10 * time.Second))
				Expect(conf.Lock.LockTTL).To(Equal(15 * time.Second))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
db:
  policy_db_url: postgres://pqgotest:password@localhost/pqgotest
  instance_metrics_db_url: postgres://pqgotest:password@localhost/pqgotest
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.GrantType).To(Equal(cf.GrantTypePassword))
				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.Logging.Level).To(Equal(DefaultLoggingLevel))
				Expect(conf.Collector.RefreshInterval).To(Equal(DefaultRefreshInterval))
				Expect(conf.Collector.CollectInterval).To(Equal(DefaultCollectInterval))

				Expect(conf.Lock.LockRetryInterval).To(Equal(DefaultRetryInterval))
				Expect(conf.Lock.LockTTL).To(Equal(DefaultLockTTL))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Cf.Api = "http://api.example.com"
			conf.Cf.GrantType = cf.GrantTypePassword
			conf.Cf.Username = "admin"
			conf.Db.PolicyDbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.Db.InstanceMetricsDbUrl = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.Lock.ConsulClusterConfig = "http://127.0.0.1:8500"
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
				conf.Db.InstanceMetricsDbUrl = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: InstanceMetrics DB url is empty")))
			})
		})

	})
})
