package config_test

import (
	"bytes"
	"time"

	"autoscaler/db"
	"autoscaler/operator/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("Config", func() {

	var (
		conf        *config.Config
		err         error
		configBytes []byte
	)

	Describe("LoadConfig", func() {

		JustBeforeEach(func() {
			conf, err = config.LoadConfig(bytes.NewReader(configBytes))
		})

		Context("with invalid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
 logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 10
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 24h
  cutoff_days: 20
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
lock:
  consul_cluster_config: "http://127.0.0.1:8500"
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
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
app_syncer:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  sync_interval: 60s
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
db_lock:
  ttl: 15s
  db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
  retry_interval: 5s
enable_db_lock: false
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.CF.API).To(Equal("https://api.example.com"))
				Expect(conf.CF.GrantType).To(Equal("PassWord"))
				Expect(conf.CF.Username).To(Equal("admin"))
				Expect(conf.CF.Password).To(Equal("admin"))
				Expect(conf.CF.ClientID).To(Equal("client-id"))
				Expect(conf.CF.Secret).To(Equal("client-secret"))
				Expect(conf.CF.SkipSSLValidation).To(Equal(false))

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.InstanceMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.InstanceMetricsDB.RefreshInterval).To(Equal(12 * time.Hour))
				Expect(conf.InstanceMetricsDB.CutoffDays).To(Equal(20))

				Expect(conf.AppMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.AppMetricsDB.RefreshInterval).To(Equal(10 * time.Hour))
				Expect(conf.AppMetricsDB.CutoffDays).To(Equal(15))

				Expect(conf.ScalingEngineDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.ScalingEngineDB.RefreshInterval).To(Equal(36 * time.Hour))
				Expect(conf.ScalingEngineDB.CutoffDays).To(Equal(30))

				Expect(conf.Lock.LockTTL).To(Equal(15 * time.Second))
				Expect(conf.Lock.LockRetryInterval).To(Equal(10 * time.Second))
				Expect(conf.Lock.ConsulClusterConfig).To(Equal("http://127.0.0.1:8500"))

				Expect(conf.DBLock.LockTTL).To(Equal(15 * time.Second))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(5 * time.Second))
				Expect(conf.DBLock.DB.URL).To(Equal("postgres://postgres:password@localhost/autoscaler?sslmode=disable"))
				Expect(conf.EnableDBLock).To(BeFalse())

				Expect(conf.AppSyncer.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.AppSyncer.SyncInterval).To(Equal(60 * time.Second))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
app_syncer:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal(config.DefaultLoggingLevel))

				Expect(conf.InstanceMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.InstanceMetricsDB.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.InstanceMetricsDB.CutoffDays).To(Equal(config.DefaultCutoffDays))
				Expect(conf.AppMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.AppMetricsDB.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.AppMetricsDB.CutoffDays).To(Equal(config.DefaultCutoffDays))
				Expect(conf.ScalingEngineDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.ScalingEngineDB.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.ScalingEngineDB.CutoffDays).To(Equal(config.DefaultCutoffDays))

				Expect(conf.ScalingEngine.SyncInterval).To(Equal(config.DefaultSyncInterval))
				Expect(conf.Scheduler.SyncInterval).To(Equal(config.DefaultSyncInterval))

				Expect(conf.Lock.LockTTL).To(Equal(config.DefaultLockTTL))
				Expect(conf.Lock.LockRetryInterval).To(Equal(config.DefaultRetryInterval))

				Expect(conf.DBLock.LockTTL).To(Equal(config.DefaultDBLockTTL))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(config.DefaultDBLockRetryInterval))

				Expect(conf.AppSyncer.SyncInterval).To(Equal(config.DefaultSyncInterval))
				Expect(conf.AppSyncer.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))

			})
		})
		Context("when it gives a non integer cutoff_days of instance_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: NOT-INTEGER-VALUE
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
			})
		})

		Context("when refresh_interval of instance_metrics_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12k
  cutoff_days: 15
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
			})
		})

		Context("when sync_interval of app_syncer is not a time duration", func() {
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
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12k
  cutoff_days: 15
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
app_syncer:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  sync_interval: 60kl
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
			})
		})

		Context("when it gives a non integer max_open_connections of instance_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when it gives a non integer max_idle_connections of instance_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when connection_max_lifetime of instance_metrics_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})
		Context("when it gives a non integer cutoff_days of app_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 15
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: NOT-INTEGER-VALUE
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when refresh_interval of app_metrics_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 15
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10k
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when it gives a non integer max_open_connections of app_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when it gives a non integer max_idle_connections of app_metrics_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when connection_max_lifetime of app_metrics_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when it gives a non integer cutoff_days of scaling_engine_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 15
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: NOT-INTEGER-VALUE
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when refresh_interval of scaling_engine_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 15
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36k
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when it gives a non integer max_open_connections of scaling_engine_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when it gives a non integer max_idle_connections of scaling_engine_db", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into int")))
			})
		})

		Context("when connection_max_lifetime of scaling_engine_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 20
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  refresh_interval: 10h
  cutoff_days: 15
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
  refresh_interval: 36h
  cutoff_days: 30
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
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
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 10
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 24h
  cutoff_days: 20
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30s
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 10k
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
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
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_days: 10
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 24h
  cutoff_days: 20
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_days: 30s
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10k
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when scaling engine sync interval is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
level: "debug"
instance_metrics_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 12h
cutoff_days: 10
app_metrics_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 24h
cutoff_days: 20
scaling_engine_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 36h
cutoff_days: 30s
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60k
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10k
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when scheduler sync interval is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
level: "debug"
instance_metrics_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 12h
cutoff_days: 10
app_metrics_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 24h
cutoff_days: 20
scaling_engine_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 36h
cutoff_days: 30s
scalingEngine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60s
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scheduler:
  scheduler_url: http://localhost:8083
  sync_interval: 60k
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/scheduler.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
lock:
  lock_ttl: 15s
  lock_retry_interval: 10k
  consul_cluster_config: "http://127.0.0.1:8500"
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
			conf = &config.Config{}

			conf.InstanceMetricsDB.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.InstanceMetricsDB.RefreshInterval = 12 * time.Hour
			conf.InstanceMetricsDB.CutoffDays = 30

			conf.AppMetricsDB.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.AppMetricsDB.RefreshInterval = 10 * time.Hour
			conf.AppMetricsDB.CutoffDays = 15

			conf.ScalingEngineDB.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.ScalingEngineDB.RefreshInterval = 36 * time.Hour
			conf.ScalingEngineDB.CutoffDays = 20

			conf.ScalingEngine.URL = "http://localhost:8082"
			conf.ScalingEngine.SyncInterval = 15 * time.Minute

			conf.Scheduler.URL = "http://localhost:8083"
			conf.Scheduler.SyncInterval = 15 * time.Minute

			conf.Lock.LockTTL = 15 * time.Second
			conf.Lock.LockRetryInterval = 10 * time.Second
			conf.Lock.ConsulClusterConfig = "http://127.0.0.1:8500"

			conf.AppSyncer.SyncInterval = 60 * time.Second
			conf.AppSyncer.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"

		})

		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("when all the configs are valid", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when InstanceMetrics db url is not set", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDB.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: instance_metrics_db.db.url is empty"))
			})
		})

		Context("when AppMetrics db url is not set", func() {

			BeforeEach(func() {
				conf.AppMetricsDB.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_metrics_db.db.url is empty"))
			})
		})

		Context("when ScalingEngine db url is not set", func() {

			BeforeEach(func() {
				conf.ScalingEngineDB.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine_db.db.url is empty"))
			})
		})

		Context("when AppSyncer db url is not set", func() {

			BeforeEach(func() {
				conf.AppSyncer.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: appSyncer.db.url is empty"))
			})
		})

		Context("when AppSyncer sync interval is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppSyncer.SyncInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: appSyncer.sync_interval is less than or equal to 0"))
			})
		})

		Context("when InstanceMetrics db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDB.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: instance_metrics_db.refresh_interval is less than or equal to 0"))
			})
		})

		Context("when AppMetrics db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDB.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_metrics_db.refresh_interval is less than or equal to 0"))
			})
		})

		Context("when ScalingEngine db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.ScalingEngineDB.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine_db.refresh_interval is less than or equal to 0"))
			})
		})

		Context("when InstanceMetrics db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDB.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: instance_metrics_db.cutoff_days is less than or equal to 0"))
			})
		})

		Context("when AppMetrics db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDB.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_metrics_db.cutoff_days is less than or equal to 0"))
			})
		})

		Context("when ScalingEngine db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.ScalingEngineDB.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine_db.cutoff_days is less than or equal to 0"))
			})
		})

		Context("when ScalingEngine url is not set", func() {

			BeforeEach(func() {
				conf.ScalingEngine.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine.scaling_engine_url is empty"))
			})
		})

		Context("when Scheduler url is not set", func() {

			BeforeEach(func() {
				conf.Scheduler.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scheduler.scheduler_url is empty"))
			})
		})

		Context("when ScalingEngine sync interval is set to 0", func() {

			BeforeEach(func() {
				conf.ScalingEngine.SyncInterval = 0
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine.sync_interval is less than or equal to 0"))
			})
		})

		Context("when Scheduler sync interval is set to 0", func() {

			BeforeEach(func() {
				conf.Scheduler.SyncInterval = 0
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scheduler.sync_interval is less than or equal to 0"))
			})
		})

		Context("when Lock ttl value in seconds is set to a negative value", func() {

			BeforeEach(func() {
				conf.Lock.LockTTL = -10
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: lock.lock_ttl is less than or equal to 0"))
			})
		})

		Context("when Lock retry interval value in seconds is set to a negative value", func() {

			BeforeEach(func() {
				conf.Lock.LockRetryInterval = -15
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: lock.lock_retry_interval is less than or equal to 0"))
			})
		})

		Context("when db lock is enabled but db url is empty", func() {

			BeforeEach(func() {
				conf.EnableDBLock = true
				conf.DBLock.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: db_lock.db.url is empty"))
			})
		})

		Context("when Consul Cluster Config is not set", func() {

			BeforeEach(func() {
				conf.Lock.ConsulClusterConfig = ""
			})

			It("should validate successfully", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})
})
