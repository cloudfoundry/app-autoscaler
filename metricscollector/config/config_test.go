package config_test

import (
	"bytes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	"autoscaler/cf"
	"autoscaler/db"
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
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
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
				Expect(conf.Cf.SkipSSLValidation).To(Equal(false))

				Expect(conf.Server.Port).To(Equal(8989))

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.Db.PolicyDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.Db.InstanceMetricsDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))

				Expect(conf.Collector.RefreshInterval).To(Equal(20 * time.Second))
				Expect(conf.Collector.CollectInterval).To(Equal(10 * time.Second))
				Expect(conf.Collector.CollectMethod).To(Equal(CollectMethodPolling))

				Expect(conf.Server.TLS.KeyFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.key"))
				Expect(conf.Server.TLS.CertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.crt"))
				Expect(conf.Server.TLS.CACertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/ca.crt"))

				Expect(conf.Lock.ConsulClusterConfig).To(Equal("http://127.0.0.1:8500"))
				Expect(conf.Lock.LockRetryInterval).To(Equal(10 * time.Second))
				Expect(conf.Lock.LockTTL).To(Equal(15 * time.Second))

				Expect(conf.DBLock.LockDB).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.DBLock.LockTTL).To(Equal(30 * time.Second))
				Expect(conf.EnableDBLock).To(Equal(true))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(5 * time.Second))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.GrantType).To(Equal(cf.GrantTypePassword))
				Expect(conf.Cf.SkipSSLValidation).To(Equal(false))
				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.Logging.Level).To(Equal(DefaultLoggingLevel))
				Expect(conf.Db.PolicyDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.Db.InstanceMetricsDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://pqgotest:password@localhost/pqgotest",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.Collector.RefreshInterval).To(Equal(DefaultRefreshInterval))
				Expect(conf.Collector.CollectInterval).To(Equal(DefaultCollectInterval))
				Expect(conf.Collector.CollectMethod).To(Equal(CollectMethodStreaming))

				Expect(conf.Lock.LockRetryInterval).To(Equal(DefaultRetryInterval))
				Expect(conf.Lock.LockTTL).To(Equal(DefaultLockTTL))
				Expect(conf.EnableDBLock).To(Equal(false))
				Expect(conf.DBLock.LockTTL).To(Equal(DefaultDBLockTTL))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(DefaultDBLockRetryInterval))
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
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_open_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_idle_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when connection_max_lifetime of policydb is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6K
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when it gives a non integer max_open_connections of instance_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_idle_connections of instance_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when connection_max_lifetime of instance_metrics_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6k
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when refresh_interval of collector is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 2k
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when collect_interval of collector is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10k
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when save_interval of collector is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5k
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when lock_ttl of lock is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15k
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when lock_retry_interval of lock is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10k
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when ttl of db_lock is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30k
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when retry_interval of db_lock is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5k
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when it gives a non integer max_open_connections of lock_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 1
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_idle_connections of lock_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 10
    connection_max_lifetime: 60s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  retry_interval: 5s
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when connection_max_lifetime of instance_metrics_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
  grant_type: PassWord
  username: admin
  password: admin
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
server:
  port: 8989
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/server.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/server.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/ca.crt
logging:
  level: DebuG
db:
  policy_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  instance_metrics_db:
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 6s
collector:
  refresh_interval: 20s
  collect_interval: 10s
  collect_method: polling
  save_interval: 5s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: http://127.0.0.1:8500
enable_db_lock: true
db_lock:
  ttl: 30s
  lock_db: 
    url: postgres://pqgotest:password@localhost/pqgotest
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
  retry_interval: 5s
`)
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
			conf.Cf.Api = "http://api.example.com"
			conf.Cf.GrantType = cf.GrantTypePassword
			conf.Cf.Username = "admin"
			conf.Cf.SkipSSLValidation = false
			conf.Db.PolicyDb = db.DatabaseConfig{
				Url:                   "postgres://pqgotest:password@localhost/pqgotest",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}
			conf.Db.InstanceMetricsDb = db.DatabaseConfig{
				Url:                   "postgres://pqgotest:password@localhost/pqgotest",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}
			conf.Lock.ConsulClusterConfig = "http://127.0.0.1:8500"
			conf.Collector.CollectInterval = time.Duration(30 * time.Second)
			conf.Collector.RefreshInterval = time.Duration(60 * time.Second)
			conf.Collector.CollectMethod = CollectMethodPolling
			conf.Collector.SaveInterval = time.Duration(5 * time.Second)
			conf.DBLock.LockDB = db.DatabaseConfig{
				Url:                   "postgres://pqgotest:password@localhost/pqgotest",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}
			conf.EnableDBLock = true
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
				conf.Db.PolicyDb.Url = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Policy DB url is empty")))
			})
		})

		Context("when metrics db url is not set", func() {
			BeforeEach(func() {
				conf.Db.InstanceMetricsDb.Url = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: InstanceMetrics DB url is empty")))
			})
		})

		Context("when collect interval is 0", func() {
			BeforeEach(func() {
				conf.Collector.CollectInterval = time.Duration(0)
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: CollectInterval is 0")))
			})
		})

		Context("when refresh interval is 0", func() {
			BeforeEach(func() {
				conf.Collector.RefreshInterval = time.Duration(0)
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: RefreshInterval is 0")))
			})
		})

		Context("when save interval is 0", func() {
			BeforeEach(func() {
				conf.Collector.SaveInterval = time.Duration(0)
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: SaveInterval is 0")))
			})
		})

		Context("when collecting method is not valid", func() {
			BeforeEach(func() {
				conf.Collector.CollectMethod = "method-not-support"
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: invalid collecting method")))
			})
		})

		Context("when lock db url is not set but dblock enabled", func() {
			BeforeEach(func() {
				conf.EnableDBLock = true
				conf.DBLock.LockDB.Url = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Lock DB Url is empty")))
			})
		})
	})
})
