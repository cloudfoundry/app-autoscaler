package config_test

import (
	"bytes"
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("Config", func() {

	var (
		conf        *Config
		err         error
		configBytes []byte
	)

	Describe("Load Config", func() {
		JustBeforeEach(func() {
			conf, err = LoadConfig(bytes.NewReader(configBytes))
		})

		Context("with invalid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
  broker_server:
	port: 8080,
public_api_server:
  port: 8081
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
cred_helper_path: path/to/helper/plugin
`)
			})

			It("returns an error", func() {
				Expect(err).To(MatchError(MatchRegexp("yaml: .*")))
			})
		})
		Context("with valid yaml", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/broker.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/broker.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
public_api_server:
  port: 8081
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/api.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/api.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
  metrics_forwarder_mtls_url: https://mtlssdsdds:8084

use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
cred_helper_impl: default
`)
			})

			It("It returns the config", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("debug"))
				Expect(conf.BrokerServer.Port).To(Equal(8080))
				Expect(conf.BrokerServer.TLS).To(Equal(
					models.TLSCerts{
						KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/broker.key",
						CACertFile: "/var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt",
						CertFile:   "/var/vcap/jobs/autoscaler/config/certs/broker.crt",
					},
				))
				Expect(conf.PublicApiServer.Port).To(Equal(8081))
				Expect(conf.PublicApiServer.TLS).To(Equal(
					models.TLSCerts{
						KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/api.key",
						CACertFile: "/var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt",
						CertFile:   "/var/vcap/jobs/autoscaler/config/certs/api.crt",
					},
				))
				Expect(conf.DB[db.BindingDb]).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.DB[db.PolicyDb]).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    10,
						MaxIdleConnections:    5,
						ConnectionMaxLifetime: 60 * time.Second,
					}))
				Expect(conf.BrokerCredentials[0].BrokerUsername).To(Equal("broker_username"))
				Expect(conf.BrokerCredentials[0].BrokerPassword).To(Equal("broker_password"))
				Expect(conf.CatalogPath).To(Equal("../exampleconfig/catalog-example.json"))
				Expect(conf.CatalogSchemaPath).To(Equal("../schemas/catalog.schema.json"))
				Expect(conf.PolicySchemaPath).To(Equal("../exampleconfig/policy.schema.json"))
				Expect(conf.Scheduler).To(Equal(
					SchedulerConfig{
						SchedulerURL: "https://localhost:8083",
						TLSClientCerts: models.TLSCerts{
							KeyFile:    "/var/vcap/jobs/autoscaler/config/certs/sc.key",
							CACertFile: "/var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt",
							CertFile:   "/var/vcap/jobs/autoscaler/config/certs/sc.crt",
						},
					},
				))
				Expect(conf.MetricsForwarder).To(Equal(
					MetricsForwarderConfig{
						MetricsForwarderUrl:     "https://localhost:8088",
						MetricsForwarderMtlsUrl: "https://mtlssdsdds:8084",
					},
				))
				Expect(conf.UseBuildInMode).To(BeFalse())
				Expect(conf.InfoFilePath).To(Equal("/var/vcap/jobs/autoscaer/config/info-file.json"))
				Expect(conf.CF).To(Equal(
					cf.CFConfig{
						API:               "https://api.example.com",
						ClientID:          "client-id",
						Secret:            "client-secret",
						SkipSSLValidation: false,
					},
				))
				Expect(conf.CredHelperImpl).To(Equal("default"))
			})
		})

		Context("with partial config", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
scaling_engine:
  scaling_engine_url: https://localhost:8083
metrics_collector:
  metrics_collector_url: https://localhost:8084
event_generator:
  event_generator_url: https://localhost:8083
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
`)
			})
			It("It returns the default values", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(conf.Logging.Level).To(Equal("info"))
				Expect(conf.BrokerServer.Port).To(Equal(8080))
				Expect(conf.PublicApiServer.Port).To(Equal(8081))
				Expect(conf.DB[db.BindingDb]).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.DB[db.PolicyDb]).To(Equal(
					db.DatabaseConfig{
						URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
						MaxOpenConnections:    0,
						MaxIdleConnections:    0,
						ConnectionMaxLifetime: 0 * time.Second,
					}))
				Expect(conf.UseBuildInMode).To(BeFalse())
			})
		})

		Context("when it gives a non integer broker_server port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: port
public_api_server:
  port: 8081
health:
  port: 8888
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer public_api_server port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: port
health:
  port: 8888
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer health server port", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: 8081
health:
  port: port
`)
			})

			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})

		Context("when it gives a non integer max_open_connections of bindingdb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: 8081
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer max_idle_connections of bindingdb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: 8081
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
  grant_type: client_credentials`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer connection_max_lifetime of bindingdb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: 8081
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: NOT-TIME-DURATION
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into time.Duration")))
			})
		})

		Context("when it gives a non integer max_open_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: 8081
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: NOT-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer max_idle_connections of bidingdb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8080
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2d
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer connection_max_lifetime of bidingdb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8080
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: NOT-TIME-DURATION
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into time.Duration")))
			})
		})

		Context("when it gives a non integer max_open_connections of policydb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
server:
  port: 8080
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: NON-INTEGER-VALUE
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer max_idle_connections of bindingdb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: 8081
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: NOT-INTEGER-VALUE
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
  `)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when it gives a non integer connection_max_lifetime of bindingdb", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: 8081
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: NOT-TIME-DURATION
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into time.Duration")))
			})
		})
		Context("when max_amount of rate_limit is not an integer", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: 8081
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
rate_limit:
  max_amount: NOT-INTEGER
  valid_duration: 1s
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into int")))
			})
		})
		Context("when valid_duration of rate_limit is not a time duration", func() {
			BeforeEach(func() {
				configBytes = []byte(`
broker_server:
  port: 8080
public_api_server:
  port: 8081
logging:
  level: debug
broker_credentials:
  - broker_username: broker_username
    broker_password: broker_password
  - broker_username: broker_username2
    broker_password: broker_password2
db:
  binding_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
  policy_db:
    url: postgres://postgres:postgres@localhost/autoscaler?sslmode=disable
    max_open_connections: 10
    max_idle_connections: 5
    connection_max_lifetime: 60s
catalog_schema_path: '../schemas/catalog.schema.json'
catalog_path: '../exampleconfig/catalog-example.json'
policy_schema_path: '../exampleconfig/policy.schema.json'
scheduler:
  scheduler_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/sc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/sc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
scaling_engine:
  scaling_engine_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/se.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/se.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_collector:
  metrics_collector_url: https://localhost:8084
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/mc.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/mc.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
event_generator:
  event_generator_url: https://localhost:8083
  tls:
    key_file: /var/vcap/jobs/autoscaler/config/certs/eg.key
    cert_file: /var/vcap/jobs/autoscaler/config/certs/eg.crt
    ca_file: /var/vcap/jobs/autoscaler/config/certs/autoscaler-ca.crt
metrics_forwarder:
  metrics_forwarder_url: https://localhost:8088
use_buildin_mode: false
info_file_path: /var/vcap/jobs/autoscaer/config/info-file.json
cf:
  api: https://api.example.com
  client_id: client-id
  secret: client-secret
  skip_ssl_validation: false
rate_limit:
  max_amount: 2
  valid_duration: NOT-TIME-DURATION
`)
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&yaml.TypeError{}))
				Expect(err).To(MatchError(MatchRegexp("cannot unmarshal.*into time.Duration")))
			})
		})

	})

	Describe("Validate", func() {
		BeforeEach(func() {
			conf = &Config{}
			conf.DB = make(map[string]db.DatabaseConfig)
			conf.DB[db.BindingDb] = db.DatabaseConfig{
				URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}
			conf.DB[db.PolicyDb] = db.DatabaseConfig{
				URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
				MaxOpenConnections:    10,
				MaxIdleConnections:    5,
				ConnectionMaxLifetime: 60 * time.Second,
			}

			brokerCred1 := BrokerCredentialsConfig{
				BrokerUsernameHash: []byte("$2a$10$WNO1cPko4iDAT6MkhaDojeJMU8ZdNH6gt.SapsFOsC0OF4cQ9qQwu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_username")'
				BrokerPasswordHash: []byte("$2a$10$evLviRLcIPKnWQqlBl3DJOvBZir9vJ4gdEeyoGgvnK/CGBnxIAFRu"), // ruby -r bcrypt -e 'puts BCrypt::Password.create("broker_password")'
			}
			var brokerCreds []BrokerCredentialsConfig
			brokerCreds = append(brokerCreds, brokerCred1)
			conf.BrokerCredentials = brokerCreds

			conf.CatalogSchemaPath = "../schemas/catalog.schema.json"
			conf.CatalogPath = "../exampleconfig/catalog-example.json"
			conf.PolicySchemaPath = "../exampleconfig/policy.schema.json"

			conf.Scheduler.SchedulerURL = "https://localhost:8083"

			conf.MetricsCollector.MetricsCollectorUrl = "https://localhost:8083"
			conf.ScalingEngine.ScalingEngineUrl = "https://localhost:8084"
			conf.EventGenerator.EventGeneratorUrl = "https://localhost:8085"
			conf.MetricsForwarder.MetricsForwarderUrl = "https://localhost:8088"

			conf.CF.API = "https://api.bosh-lite.com"
			conf.CF.ClientID = "client-id"
			conf.CF.Secret = "secret"

			conf.InfoFilePath = "../exampleconfig/info-file.json"
			conf.UseBuildInMode = false

			conf.RateLimit.MaxAmount = 10
			conf.RateLimit.ValidDuration = 1 * time.Second

			conf.CredHelperImpl = "path/to/plugin"
		})
		JustBeforeEach(func() {
			err = conf.Validate()
		})

		Context("When all the configs are valid", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when bindingdb url is not set", func() {
			BeforeEach(func() {
				conf.DB[db.BindingDb] = db.DatabaseConfig{URL: ""}
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: BindingDB URL is empty")))
			})
		})

		Context("when policydb url is not set", func() {
			BeforeEach(func() {
				conf.DB[db.PolicyDb] = db.DatabaseConfig{URL: ""}
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: PolicyDB URL is empty")))
			})
		})

		Context("when scheduler url is not set", func() {
			BeforeEach(func() {
				conf.Scheduler.SchedulerURL = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: scheduler.scheduler_url is empty")))
			})
		})

		Context("when neither the broker username nor its hash is set", func() {
			BeforeEach(func() {
				brokerCred1 := BrokerCredentialsConfig{
					BrokerPasswordHash: []byte(""),
					BrokerPassword:     "",
				}
				var brokerCreds []BrokerCredentialsConfig
				brokerCreds = append(brokerCreds, brokerCred1)
				conf.BrokerCredentials = brokerCreds
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: both broker_username and broker_username_hash are empty, please provide one of them")))
			})
		})

		Context("when both the broker username and its hash are set", func() {
			BeforeEach(func() {
				brokerCred1 := BrokerCredentialsConfig{
					BrokerUsername:     "broker_username",
					BrokerUsernameHash: []byte("$2a$10$WNO1cPko4iDAT6MkhaDojeJMU8ZdNH6gt.SapsFOsC0OF4cQ9qQwu"),
				}
				var brokerCreds []BrokerCredentialsConfig
				brokerCreds = append(brokerCreds, brokerCred1)
				conf.BrokerCredentials = brokerCreds
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: both broker_username and broker_username_hash are set, please provide only one of them")))
			})
		})

		Context("when just the broker username is set", func() {
			BeforeEach(func() {
				conf.BrokerCredentials[0].BrokerUsername = "broker_username"
				conf.BrokerCredentials[0].BrokerUsernameHash = []byte("")
			})
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the broker username hash is set to an invalid value", func() {
			BeforeEach(func() {
				conf.BrokerCredentials[0].BrokerUsernameHash = []byte("not a bcrypt hash")
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: broker_username_hash is not a valid bcrypt hash")))
			})
		})

		Context("when neither the broker password nor its hash is set", func() {
			BeforeEach(func() {
				conf.BrokerCredentials[0].BrokerPassword = ""
				conf.BrokerCredentials[0].BrokerPasswordHash = []byte("")
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: both broker_password and broker_password_hash are empty, please provide one of them")))
			})
		})

		Context("when both the broker password and its hash are set", func() {
			BeforeEach(func() {
				conf.BrokerCredentials[0].BrokerPassword = "broker_password"
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: both broker_password and broker_password_hash are set, please provide only one of them")))
			})
		})

		Context("when just the broker password is set", func() {
			BeforeEach(func() {
				conf.BrokerCredentials[0].BrokerPassword = "broker_password"
				conf.BrokerCredentials[0].BrokerPasswordHash = []byte("")
			})
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the broker password hash is set to an invalid value", func() {
			BeforeEach(func() {
				brokerCred1 := BrokerCredentialsConfig{
					BrokerUsername:     "broker_username",
					BrokerPasswordHash: []byte("not a bcrypt hash"),
				}
				var brokerCreds []BrokerCredentialsConfig
				brokerCreds = append(brokerCreds, brokerCred1)
				conf.BrokerCredentials = brokerCreds
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: broker_password_hash is not a valid bcrypt hash")))
			})
		})

		Context("when metricscollector url is not set", func() {
			BeforeEach(func() {
				conf.MetricsCollector.MetricsCollectorUrl = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: metrics_collector.metrics_collector_url is empty")))
			})
		})

		Context("when eventgenerator url is not set", func() {
			BeforeEach(func() {
				conf.EventGenerator.EventGeneratorUrl = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: event_generator.event_generator_url is empty")))
			})
		})

		Context("when scalingengine url is not set", func() {
			BeforeEach(func() {
				conf.ScalingEngine.ScalingEngineUrl = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: scaling_engine.scaling_engine_url is empty")))
			})
		})

		Context("when metricsforwarder url is not set", func() {
			BeforeEach(func() {
				conf.MetricsForwarder.MetricsForwarderUrl = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: metrics_forwarder.metrics_forwarder_url is empty")))
			})
		})

		Context("when catalog schema path is not set", func() {
			BeforeEach(func() {
				conf.CatalogSchemaPath = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: CatalogSchemaPath is empty")))
			})
		})

		Context("when catalog path is not set", func() {
			BeforeEach(func() {
				conf.CatalogPath = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: CatalogPath is empty")))
			})
		})

		Context("when policy schema path is not set", func() {
			BeforeEach(func() {
				conf.PolicySchemaPath = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: PolicySchemaPath is empty")))
			})
		})

		Context("when catalog is not valid json", func() {
			BeforeEach(func() {
				conf.CatalogPath = "../exampleconfig/catalog-invalid-json-example.json"
			})
			It("should err", func() {
				Expect(err).To(MatchError("invalid character '[' after object key"))
			})
		})

		Context("when catalog is missing required fields", func() {
			BeforeEach(func() {
				conf.CatalogPath = "../exampleconfig/catalog-missing-example.json"
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("{\"name is required\"}")))
			})
		})

		Context("when catalog has invalid type fields", func() {
			BeforeEach(func() {
				conf.CatalogPath = "../exampleconfig/catalog-invalid-example.json"
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("{\"Invalid type. Expected: boolean, given: integer\"}")))
			})
		})

		Context("when info_file_path is not set", func() {
			BeforeEach(func() {
				conf.InfoFilePath = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: InfoFilePath is empty")))
			})
		})

		Context("when cf.client_id is not set", func() {
			BeforeEach(func() {
				conf.CF.ClientID = ""
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: client_id is empty")))
			})
		})

		Context("when rate_limit.max_amount is <= zero", func() {
			BeforeEach(func() {
				conf.RateLimit.MaxAmount = 0
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: RateLimit.MaxAmount is equal or less than zero")))
			})
		})

		Context("when rate_limit.valid_duration is <= 0 ns", func() {
			BeforeEach(func() {
				conf.RateLimit.ValidDuration = 0 * time.Nanosecond
			})
			It("should err", func() {
				Expect(err).To(MatchError(MatchRegexp("Configuration error: RateLimit.ValidDuration is equal or less than zero nanosecond")))
			})
		})

		Describe("Using BuildIn Mode", func() {
			BeforeEach(func() {
				conf.UseBuildInMode = true
			})
			Context("when broker related configs are not set", func() {
				BeforeEach(func() {
					conf.DB[db.BindingDb] = db.DatabaseConfig{URL: ""}
					brokerCred1 := BrokerCredentialsConfig{
						BrokerUsername:     "",
						BrokerUsernameHash: nil,
						BrokerPasswordHash: nil,
						BrokerPassword:     "",
					}
					var brokerCreds []BrokerCredentialsConfig
					brokerCreds = append(brokerCreds, brokerCred1)
					conf.CatalogPath = ""
					conf.CatalogSchemaPath = ""
					conf.BrokerCredentials = brokerCreds
				})
				It("should not err", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})
})
