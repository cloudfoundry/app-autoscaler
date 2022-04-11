package config_test

import (
	"bytes"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/operator/config"

	. "github.com/onsi/ginkgo/v2"
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
  cutoff_duration: 10h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 24h
  cutoff_duration: 20h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
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
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
health:
  port: 9999
logging:
  level: "debug"
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
db_lock:
  ttl: 15s
  db:
    url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
  retry_interval: 5s
http_client_timeout: 10s
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.CF.API).To(Equal("https://api.example.com"))
				Expect(conf.CF.ClientID).To(Equal("client-id"))
				Expect(conf.CF.Secret).To(Equal("client-secret"))
				Expect(conf.CF.SkipSSLValidation).To(Equal(false))
				Expect(conf.Health.Port).To(Equal(9999))
				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.InstanceMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.InstanceMetricsDB.RefreshInterval).To(Equal(12 * time.Hour))
				Expect(conf.InstanceMetricsDB.CutoffDuration).To(Equal(20 * time.Hour))

				Expect(conf.AppMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.AppMetricsDB.RefreshInterval).To(Equal(10 * time.Hour))
				Expect(conf.AppMetricsDB.CutoffDuration).To(Equal(15 * time.Hour))

				Expect(conf.ScalingEngineDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.ScalingEngineDB.RefreshInterval).To(Equal(36 * time.Hour))
				Expect(conf.ScalingEngineDB.CutoffDuration).To(Equal(30 * time.Hour))

				Expect(conf.DBLock.LockTTL).To(Equal(15 * time.Second))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(5 * time.Second))
				Expect(conf.DBLock.DB.URL).To(Equal("postgres://postgres:password@localhost/autoscaler?sslmode=disable"))

				Expect(conf.AppSyncer.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.AppSyncer.SyncInterval).To(Equal(60 * time.Second))
				Expect(conf.HttpClientTimeout).To(Equal(10 * time.Second))
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
				Expect(conf.Health.Port).To(Equal(8081))
				Expect(conf.InstanceMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.InstanceMetricsDB.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.InstanceMetricsDB.CutoffDuration).To(Equal(config.DefaultCutoffDuration))
				Expect(conf.AppMetricsDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.AppMetricsDB.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.AppMetricsDB.CutoffDuration).To(Equal(config.DefaultCutoffDuration))
				Expect(conf.ScalingEngineDB.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.ScalingEngineDB.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.ScalingEngineDB.CutoffDuration).To(Equal(config.DefaultCutoffDuration))

				Expect(conf.ScalingEngine.SyncInterval).To(Equal(config.DefaultSyncInterval))
				Expect(conf.Scheduler.SyncInterval).To(Equal(config.DefaultSyncInterval))

				Expect(conf.DBLock.LockTTL).To(Equal(config.DefaultDBLockTTL))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(config.DefaultDBLockRetryInterval))

				Expect(conf.AppSyncer.SyncInterval).To(Equal(config.DefaultSyncInterval))
				Expect(conf.AppSyncer.DB).To(Equal(db.DatabaseConfig{
					URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))

				Expect(conf.HttpClientTimeout).To(Equal(5 * time.Second))

			})
		})
		Context("when cutoff_duration of instance_metrics_db is not a time.Duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 7k
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12k
  cutoff_duration: 15h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
logging:
  level: "debug"
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12k
  cutoff_duration: 15h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})
		Context("when cutoff_duration of app_metrics_db is not a time.Duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 15h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 7k
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when refresh_interval of app_metrics_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 15h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10k
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when cutoff_duration of scaling_engine_db is not a time.Duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 15h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 7k
scaling_engine:
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
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when refresh_interval of scaling_engine_db is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 15h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36k
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60k
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
health:
  port: 9999
instance_metrics_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 12h
cutoff_duration: 10h
app_metrics_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 24h
cutoff_duration: 20h
scaling_engine_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 36h
cutoff_duration: 30h
scaling_engine:
  scaling_engine_url: http://localhost:8082
  sync_interval: 60kddd
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
health:
  port: 9999
instance_metrics_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 12h
cutoff_duration: 10h
app_metrics_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 24h
cutoff_duration: 20h
scaling_engine_db:
db:
  url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
  max_open_connections: 10
  max_idle_connections: 5
  connection_max_lifetime: 60s
refresh_interval: 36h
cutoff_duration: 30h
scaling_engine:
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
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal .* into time.Duration")))
			})
		})

		Context("when http_client_timeout of http is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
logging:
  level: "debug"
health:
  port: 9999
instance_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 12h
  cutoff_duration: 20h
app_metrics_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  refresh_interval: 10h
  cutoff_duration: 15h
scaling_engine_db:
  db:
    url: "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  refresh_interval: 36h
  cutoff_duration: 30h
scaling_engine:
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
http_client_timeout: 10k
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
			conf.InstanceMetricsDB.CutoffDuration = 30 * time.Hour

			conf.AppMetricsDB.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.AppMetricsDB.RefreshInterval = 10 * time.Hour
			conf.AppMetricsDB.CutoffDuration = 15 * time.Hour

			conf.ScalingEngineDB.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.ScalingEngineDB.RefreshInterval = 36 * time.Hour
			conf.ScalingEngineDB.CutoffDuration = 20 * time.Hour

			conf.ScalingEngine.URL = "http://localhost:8082"
			conf.ScalingEngine.SyncInterval = 15 * time.Minute

			conf.Scheduler.URL = "http://localhost:8083"
			conf.Scheduler.SyncInterval = 15 * time.Minute

			conf.AppSyncer.SyncInterval = 60 * time.Second
			conf.AppSyncer.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.DBLock.DB.URL = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.HttpClientTimeout = 10 * time.Second
			conf.Health.Port = 8081

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

		Context("when InstanceMetrics db cutoff duration is set to a negative value", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDB.CutoffDuration = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: instance_metrics_db.cutoff_duration is less than or equal to 0"))
			})
		})

		Context("when AppMetrics db cutoff duration is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDB.CutoffDuration = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: app_metrics_db.cutoff_duration is less than or equal to 0"))
			})
		})

		Context("when ScalingEngine db cutoff duration is set to a negative value", func() {

			BeforeEach(func() {
				conf.ScalingEngineDB.CutoffDuration = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: scaling_engine_db.cutoff_duration is less than or equal to 0"))
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

		Context("when db lockdb url is empty", func() {

			BeforeEach(func() {
				conf.DBLock.DB.URL = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError("Configuration error: db_lock.db.url is empty"))
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
