package config_test

import (
	"bytes"
	"time"

	"autoscaler/db"
	"autoscaler/pruner/config"

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
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
db_lock:
  ttl: 15s
  url: postgres://postgres:password@localhost/autoscaler?sslmode=disable
  retry_interval: 5s
enable_db_lock: false
`)
			})

			It("returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("debug"))

				Expect(conf.InstanceMetricsDb.Db).To(Equal(db.DatabaseConfig{
					Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.InstanceMetricsDb.RefreshInterval).To(Equal(12 * time.Hour))
				Expect(conf.InstanceMetricsDb.CutoffDays).To(Equal(20))

				Expect(conf.AppMetricsDb.Db).To(Equal(db.DatabaseConfig{
					Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.AppMetricsDb.RefreshInterval).To(Equal(10 * time.Hour))
				Expect(conf.AppMetricsDb.CutoffDays).To(Equal(15))

				Expect(conf.ScalingEngineDb.Db).To(Equal(db.DatabaseConfig{
					Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    10,
					MaxIdleConnections:    5,
					ConnectionMaxLifetime: 60 * time.Second,
				}))
				Expect(conf.ScalingEngineDb.RefreshInterval).To(Equal(36 * time.Hour))
				Expect(conf.ScalingEngineDb.CutoffDays).To(Equal(30))

				Expect(conf.Lock.LockTTL).To(Equal(15 * time.Second))
				Expect(conf.Lock.LockRetryInterval).To(Equal(10 * time.Second))
				Expect(conf.Lock.ConsulClusterConfig).To(Equal("http://127.0.0.1:8500"))

				Expect(conf.DBLock.LockTTL).To(Equal(15 * time.Second))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(5 * time.Second))
				Expect(conf.EnableDBLock).To(BeFalse())
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
`)
			})

			It("returns default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal(config.DefaultLoggingLevel))

				Expect(conf.InstanceMetricsDb.Db).To(Equal(db.DatabaseConfig{
					Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.InstanceMetricsDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.InstanceMetricsDb.CutoffDays).To(Equal(config.DefaultCutoffDays))
				Expect(conf.AppMetricsDb.Db).To(Equal(db.DatabaseConfig{
					Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.AppMetricsDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.AppMetricsDb.CutoffDays).To(Equal(config.DefaultCutoffDays))
				Expect(conf.ScalingEngineDb.Db).To(Equal(db.DatabaseConfig{
					Url:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
					MaxOpenConnections:    0,
					MaxIdleConnections:    0,
					ConnectionMaxLifetime: 0 * time.Second,
				}))
				Expect(conf.ScalingEngineDb.RefreshInterval).To(Equal(config.DefaultRefreshInterval))
				Expect(conf.ScalingEngineDb.CutoffDays).To(Equal(config.DefaultCutoffDays))

				Expect(conf.Lock.LockTTL).To(Equal(config.DefaultLockTTL))
				Expect(conf.Lock.LockRetryInterval).To(Equal(config.DefaultRetryInterval))

				Expect(conf.DBLock.LockTTL).To(Equal(config.DefaultDBLockTTL))
				Expect(conf.DBLock.LockRetryInterval).To(Equal(config.DefaultDBLockRetryInterval))

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
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
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
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
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
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
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
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
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
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
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
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
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
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
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
lock:
  lock_ttl: 15s
  lock_retry_interval: 10s
  consul_cluster_config: "http://127.0.0.1:8500"
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
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

			conf.InstanceMetricsDb.Db.Url = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.InstanceMetricsDb.RefreshInterval = 12 * time.Hour
			conf.InstanceMetricsDb.CutoffDays = 30

			conf.AppMetricsDb.Db.Url = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.AppMetricsDb.RefreshInterval = 10 * time.Hour
			conf.AppMetricsDb.CutoffDays = 15

			conf.ScalingEngineDb.Db.Url = "postgres://pqgotest:password@exampl.com/pqgotest"
			conf.ScalingEngineDb.RefreshInterval = 36 * time.Hour
			conf.ScalingEngineDb.CutoffDays = 20

			conf.Lock.LockTTL = 15 * time.Second
			conf.Lock.LockRetryInterval = 10 * time.Second
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

		Context("when InstanceMetrics db url is not set", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDb.Db.Url = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: InstanceMetrics DB url is empty")))
			})
		})

		Context("when AppMetrics db url is not set", func() {

			BeforeEach(func() {
				conf.AppMetricsDb.Db.Url = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: AppMetrics DB url is empty")))
			})
		})

		Context("when ScalingEngine db url is not set", func() {

			BeforeEach(func() {
				conf.ScalingEngineDb.Db.Url = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: ScalingEngine DB url is empty")))
			})
		})

		Context("when InstanceMetrics db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDb.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: InstanceMetrics DB refresh interval is negative")))
			})
		})

		Context("when AppMetrics db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDb.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: AppMetrics DB refresh interval is negative")))
			})
		})

		Context("when ScalingEngine db refresh interval in hours is set to a negative value", func() {

			BeforeEach(func() {
				conf.ScalingEngineDb.RefreshInterval = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: ScalingEngine DB refresh interval is negative")))
			})
		})

		Context("when InstanceMetrics db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.InstanceMetricsDb.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: InstanceMetrics DB cutoff days is negative")))
			})
		})

		Context("when AppMetrics db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.AppMetricsDb.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: AppMetrics DB cutoff days is negative")))
			})
		})

		Context("when ScalingEngine db cutoff days is set to a negative value", func() {

			BeforeEach(func() {
				conf.ScalingEngineDb.CutoffDays = -1
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: ScalingEngine DB cutoff days is negative")))
			})
		})

		Context("when Lock ttl value in seconds is set to a negative value", func() {

			BeforeEach(func() {
				conf.Lock.LockTTL = -10
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: lock ttl is less than or equal to 0")))
			})
		})

		Context("when Lock retry interval value in seconds is set to a negative value", func() {

			BeforeEach(func() {
				conf.Lock.LockRetryInterval = -15
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: lock retry interval is less than or equal to 0")))
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
