package config

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("rawConfig validation", func() {
	var (
		conf *rawConfig
		err  error
	)

	BeforeEach(func() {
		conf = &rawConfig{}
		conf.Db = make(map[string]db.DatabaseConfig)
		conf.Db[db.BindingDb] = db.DatabaseConfig{
			URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 60 * time.Second,
		}
		conf.Db[db.PolicyDb] = db.DatabaseConfig{
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
		conf.BindingRequestSchemaPath = "../exampleconfig/policy.schema.json"

		conf.Scheduler.SchedulerURL = "https://localhost:8083"

		conf.ScalingEngine.ScalingEngineUrl = "https://localhost:8084"
		conf.EventGenerator.EventGeneratorUrl = "https://localhost:8085"
		conf.MetricsForwarder.MetricsForwarderUrl = "https://localhost:8088"

		conf.CF.API = "https://api.bosh-lite.com"
		conf.CF.ClientID = "client-id"
		conf.CF.Secret = "secret"

		conf.InfoFilePath = "../exampleconfig/info-file.json"

		conf.RateLimit.MaxAmount = 10
		conf.RateLimit.ValidDuration = 1 * time.Second

		conf.CredHelperImpl = "default"
	})

	JustBeforeEach(func() {
		err = conf.validate()
	})

	Context("when all the configs are valid", func() {
		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when bindingdb url is not set", func() {
		BeforeEach(func() {
			conf.Db[db.BindingDb] = db.DatabaseConfig{URL: ""}
		})
		It("should err", func() {
			Expect(err).To(MatchError(MatchRegexp("Configuration error: BindingDB URL is empty")))
		})
	})

	Context("when policydb url is not set", func() {
		BeforeEach(func() {
			conf.Db[db.PolicyDb] = db.DatabaseConfig{URL: ""}
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
			conf.BindingRequestSchemaPath = ""
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
})

var _ = Describe("toConfig credential helper and basic auth conversion", func() {
	var (
		conf   rawConfig
		result Config
		err    error
	)

	BeforeEach(func() {
		conf = rawConfig{}
		conf.Db = make(map[string]db.DatabaseConfig)
		conf.Db[db.BindingDb] = db.DatabaseConfig{
			URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 60 * time.Second,
		}
		conf.Db[db.PolicyDb] = db.DatabaseConfig{
			URL:                   "postgres://postgres:postgres@localhost/autoscaler?sslmode=disable",
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 60 * time.Second,
		}

		conf.BrokerCredentials = []BrokerCredentialsConfig{{
			BrokerUsernameHash: []byte("$2a$10$WNO1cPko4iDAT6MkhaDojeJMU8ZdNH6gt.SapsFOsC0OF4cQ9qQwu"),
			BrokerPasswordHash: []byte("$2a$10$evLviRLcIPKnWQqlBl3DJOvBZir9vJ4gdEeyoGgvnK/CGBnxIAFRu"),
		}}

		conf.CatalogSchemaPath = "../schemas/catalog.schema.json"
		conf.CatalogPath = "../exampleconfig/catalog-example.json"
		conf.BindingRequestSchemaPath = "../exampleconfig/policy.schema.json"
		conf.Scheduler.SchedulerURL = "https://localhost:8083"
		conf.ScalingEngine.ScalingEngineUrl = "https://localhost:8084"
		conf.EventGenerator.EventGeneratorUrl = "https://localhost:8085"
		conf.MetricsForwarder.MetricsForwarderUrl = "https://localhost:8088"
		conf.CF.API = "https://api.bosh-lite.com"
		conf.CF.ClientID = "client-id"
		conf.CF.Secret = "secret"
		conf.InfoFilePath = "../exampleconfig/info-file.json"
		conf.RateLimit.MaxAmount = 10
		conf.RateLimit.ValidDuration = 1 * time.Second

		// Default valid state
		conf.CredHelperImpl = "default"
		conf.BasicAuthForCustomMetrics = "on"
		conf.DefaultCustomMetricsCredentialType = "x509"
	})

	JustBeforeEach(func() {
		result, err = toConfig(conf)
	})

	// --- cred_helper_impl tests (same as metricsforwarder) ---

	When("cred_helper_impl is set to 'default'", func() {
		BeforeEach(func() {
			conf.CredHelperImpl = "default"
			conf.StoredProcedureConfig = nil
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should set BasicAuthHandlingImplConfig to BasicAuthHandlingNative", func() {
			Expect(result.CustomMetricsAuthConfig).NotTo(BeNil())
			Expect(result.CustomMetricsAuthConfig.BasicAuthHandlingImplConfig).To(
				BeAssignableToTypeOf(models.BasicAuthHandlingNative{}))
		})
	})

	When("cred_helper_impl is set to 'stored_procedure' with full config", func() {
		BeforeEach(func() {
			conf.CredHelperImpl = "stored_procedure"
			conf.StoredProcedureConfig = &models.StoredProcedureConfig{
				SchemaName:                             "test_schema",
				CreateBindingCredentialProcedureName:   "create_binding",
				DropBindingCredentialProcedureName:     "drop_binding",
				DropAllBindingCredentialProcedureName:  "drop_all_bindings",
				ValidateBindingCredentialProcedureName: "validate_binding",
			}
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should set BasicAuthHandlingImplConfig to BasicAuthHandlingStoredProc", func() {
			Expect(result.CustomMetricsAuthConfig).NotTo(BeNil())
			Expect(result.CustomMetricsAuthConfig.BasicAuthHandlingImplConfig).To(
				BeAssignableToTypeOf(models.BasicAuthHandlingStoredProc{}))
		})

		It("should carry the StoredProcedureConfig with all fields", func() {
			storedProc := result.CustomMetricsAuthConfig.BasicAuthHandlingImplConfig.(models.BasicAuthHandlingStoredProc)
			Expect(storedProc.Config.SchemaName).To(Equal("test_schema"))
			Expect(storedProc.Config.CreateBindingCredentialProcedureName).To(Equal("create_binding"))
			Expect(storedProc.Config.DropBindingCredentialProcedureName).To(Equal("drop_binding"))
			Expect(storedProc.Config.DropAllBindingCredentialProcedureName).To(Equal("drop_all_bindings"))
		})
	})

	When("cred_helper_impl is 'stored_procedure' but StoredProcedureConfig is nil", func() {
		BeforeEach(func() {
			conf.CredHelperImpl = "stored_procedure"
			conf.StoredProcedureConfig = nil
		})

		It("should return an error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("cred_helper_impl is set to an invalid value", func() {
		BeforeEach(func() {
			conf.CredHelperImpl = "some_unknown_impl"
		})

		It("should return an error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	// --- basic_auth_for_custom_metrics tests ---

	When("basic_auth_for_custom_metrics is set to an invalid value", func() {
		BeforeEach(func() {
			conf.BasicAuthForCustomMetrics = "invalid_value"
		})

		It("should return an error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("basic_auth_for_custom_metrics is 'off'", func() {
		BeforeEach(func() {
			conf.BasicAuthForCustomMetrics = "off"
			conf.CredHelperImpl = ""
			conf.StoredProcedureConfig = nil
			conf.DefaultCustomMetricsCredentialType = ""
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should set CustomMetricsAuthConfig to nil", func() {
			Expect(result.CustomMetricsAuthConfig).To(BeNil())
		})
	})

	When("basic_auth_for_custom_metrics is 'off' but cred_helper_impl is set", func() {
		BeforeEach(func() {
			conf.BasicAuthForCustomMetrics = "off"
			conf.CredHelperImpl = "default"
		})

		It("should return an error (contradictory config)", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("basic_auth_for_custom_metrics is 'off' but default_credential_type is set", func() {
		BeforeEach(func() {
			conf.BasicAuthForCustomMetrics = "off"
			conf.CredHelperImpl = ""
			conf.DefaultCustomMetricsCredentialType = "binding-secret"
		})

		It("should return an error (contradictory config)", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("basic_auth_for_custom_metrics is 'on'", func() {
		BeforeEach(func() {
			conf.BasicAuthForCustomMetrics = "on"
			conf.CredHelperImpl = "default"
			conf.DefaultCustomMetricsCredentialType = "binding-secret"
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should set CustomMetricsAuthConfig with BasicAuthHandlingOn", func() {
			Expect(result.CustomMetricsAuthConfig).NotTo(BeNil())
			Expect(result.CustomMetricsAuthConfig.BasicAuthHandling).To(Equal(BasicAuthHandlingOn))
		})

		It("should set the DefaultCustomMetricAuthType correctly", func() {
			Expect(result.CustomMetricsAuthConfig.DefaultCustomMetricAuthType).To(Equal(models.BindingSecret))
		})
	})

	When("basic_auth_for_custom_metrics is 'only_existing_bindings' with default_credential_type 'x509'", func() {
		BeforeEach(func() {
			conf.BasicAuthForCustomMetrics = "only_existing_bindings"
			conf.CredHelperImpl = "default"
			conf.DefaultCustomMetricsCredentialType = "x509"
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should set CustomMetricsAuthConfig with BasicAuthHandlingOnlyExistingBindings", func() {
			Expect(result.CustomMetricsAuthConfig).NotTo(BeNil())
			Expect(result.CustomMetricsAuthConfig.BasicAuthHandling).To(Equal(BasicAuthHandlingOnlyExistingBindings))
		})
	})

	When("basic_auth_for_custom_metrics is 'only_existing_bindings' with default_credential_type 'binding-secret'", func() {
		BeforeEach(func() {
			conf.BasicAuthForCustomMetrics = "only_existing_bindings"
			conf.CredHelperImpl = "default"
			conf.DefaultCustomMetricsCredentialType = "binding-secret"
		})

		It("should return an error (contradictory: new bindings would use binding-secret which is not allowed)", func() {
			Expect(err).To(HaveOccurred())
		})
	})
})
