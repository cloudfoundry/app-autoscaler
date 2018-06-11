package config_test

import (
	"autoscaler/cf"
	"autoscaler/db"
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
defaultCoolDownSecs: 300
lockSize: 32
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
db_lock:
  ttl: 15s
  url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
  retry_interval: 5s
enable_db_lock: false
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
				Expect(conf.Server.TLS.KeyFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.key"))
				Expect(conf.Server.TLS.CertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/server.crt"))
				Expect(conf.Server.TLS.CACertFile).To(Equal("/var/vcap/jobs/autoscaler/config/certs/ca.crt"))

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.Db.PolicyDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.Db.ScalingEngineDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.Db.SchedulerDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))

				Expect(conf.Synchronizer.ActiveScheduleSyncInterval).To(Equal(5 * time.Minute))

				Expect(conf.DefaultCoolDownSecs).To(Equal(300))

				Expect(conf.LockSize).To(Equal(32))

				Expect(conf.DBLock.LockTTL).To(Equal(15 * time.Second))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(5 * time.Second))
				Expect(conf.EnableDBLock).To(BeFalse())
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
cf:
  api: https://api.example.com
db:
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
  scalingengine_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
  scheduler_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
defaultCoolDownSecs: 300
lockSize: 32
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Cf.GrantType).To(Equal(cf.GrantTypePassword))
				Expect(conf.Cf.SkipSSLValidation).To(Equal(false))
				Expect(conf.Server.Port).To(Equal(8080))
				Expect(conf.Logging.Level).To(Equal("info"))
				Expect(conf.Db.PolicyDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.Db.ScalingEngineDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.Db.SchedulerDb).To(Equal(
					db.DatabaseConfig{
						Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.Synchronizer.ActiveScheduleSyncInterval).To(Equal(DefaultActiveScheduleSyncInterval))

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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when it gives a non integer max_open_connections of scalingengine_db", func() {
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_idle_connections of scalingengine_db", func() {
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when connection_max_lifetime of scalingengine_db is not a time duration", func() {
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})
		Context("when it gives a non integer max_open_connections of scheduler_db", func() {
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_idle_connections of scheduler_db", func() {
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when connection_max_lifetime of scheduler_db is not a time duration", func() {
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
synchronizer:
  active_schedule_sync_interval: 300s
defaultCoolDownSecs: 300
lockSize: 32
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when active_schedule_sync_interval of synchronizer is not a time duration", func() {
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300k
defaultCoolDownSecs: 300
lockSize: 32
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when it gives a non integer of defaultCoolDownSecs", func() {
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300
defaultCoolDownSecs: NOT-INTEGER-VALUE
lockSize: 32
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer of lockSize", func() {
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
  level: DeBug
db:
  policy_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  scalingengine_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  scheduler_db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
synchronizer:
  active_schedule_sync_interval: 300
defaultCoolDownSecs: 300
lockSize: NOT-INTEGER-VALUE
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.Cf.Api = "http://api.example.com"
			conf.Cf.GrantType = cf.GrantTypePassword
			conf.Cf.SkipSSLValidation = false
			conf.Cf.Username = "admin"
			conf.Db.PolicyDb.Url = "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
			conf.Db.ScalingEngineDb.Url = "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
			conf.Db.SchedulerDb.Url = "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
			conf.DefaultCoolDownSecs = 300
			conf.LockSize = 32
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

		Context("when scalingengine db url is not set", func() {
			BeforeEach(func() {
				conf.Db.ScalingEngineDb.Url = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: ScalingEngine DB url is empty")))
			})
		})

		Context("when scheduler db url is not set", func() {
			BeforeEach(func() {
				conf.Db.SchedulerDb.Url = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: Scheduler DB url is empty")))
			})
		})

		Context("when DefaultCoolDownSecs < 60", func() {
			BeforeEach(func() {
				conf.DefaultCoolDownSecs = 10
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: DefaultCoolDownSecs should be between 60 and 3600")))
			})
		})

		Context("when DefaultCoolDownSecs > 3600", func() {
			BeforeEach(func() {
				conf.DefaultCoolDownSecs = 5000
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: DefaultCoolDownSecs should be between 60 and 3600")))
			})
		})

		Context("when LockSize <= 0", func() {
			BeforeEach(func() {
				conf.LockSize = 0
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: LockSize is less than or equal to 0")))
			})
		})
	})

})
