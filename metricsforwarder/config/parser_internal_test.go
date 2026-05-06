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
		conf.Server.Port = 8081
		conf.Logging.Level = "debug"
		conf.LoggregatorConfig.MetronAddress = "127.0.0.1:3458"
		conf.LoggregatorConfig.TLS.CACertFile = "../testcerts/ca.crt"
		conf.LoggregatorConfig.TLS.CertFile = "../testcerts/client.crt"
		conf.LoggregatorConfig.TLS.KeyFile = "../testcerts/client.crt"
		conf.Db = make(map[string]db.DatabaseConfig)
		conf.Db[db.PolicyDb] = db.DatabaseConfig{
			URL:                   "postgres://pqgotest:password@localhost/pqgotest",
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 60 * time.Second,
		}
		conf.Db[db.BindingDb] = db.DatabaseConfig{
			URL:                   "postgres://pqgotest:password@localhost/pqgotest",
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 60 * time.Second,
		}
		conf.RateLimit.MaxAmount = 10
		conf.RateLimit.ValidDuration = 1 * time.Second

		conf.CredHelperImpl = "default"
	})

	JustBeforeEach(func() {
		err = conf.validate()
	})

	When("all the configs are valid", func() {
		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("syslog is available", func() {
		BeforeEach(func() {
			conf.SyslogConfig = SyslogConfig{
				ServerAddress: "localhost",
				Port:          514,
				TLS: models.TLSCerts{
					CACertFile: "../testcerts/ca.crt",
					CertFile:   "../testcerts/client.crt",
					KeyFile:    "../testcerts/client.crt",
				},
			}
			conf.LoggregatorConfig = LoggregatorConfig{}
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		When("SyslogServer CACert is not set", func() {
			BeforeEach(func() {
				conf.SyslogConfig.TLS.CACertFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("SyslogServer Loggregator CACert is empty")))
			})
		})

		When("SyslogServer CertFile is not set", func() {
			BeforeEach(func() {
				conf.SyslogConfig.TLS.KeyFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("SyslogServer ClientKey is empty")))
			})
		})

		When("SyslogServer ClientCert is not set", func() {
			BeforeEach(func() {
				conf.SyslogConfig.TLS.CertFile = ""
			})

			It("should error", func() {
				Expect(err).To(MatchError(MatchRegexp("SyslogServer ClientCert is empty")))
			})
		})
	})

	When("policy db url is not set", func() {
		BeforeEach(func() {
			conf.Db[db.PolicyDb] = db.DatabaseConfig{URL: ""}
		})

		It("should error", func() {
			Expect(err).To(MatchError(MatchRegexp("configuration error: Policy DB url is empty")))
		})
	})

	When("binding db url is not set", func() {
		BeforeEach(func() {
			conf.Db[db.BindingDb] = db.DatabaseConfig{URL: ""}
		})

		It("should error", func() {
			Expect(err).To(MatchError(MatchRegexp("configuration error: Binding DB url is empty")))
		})
	})

	When("Loggregator CACert is not set", func() {
		BeforeEach(func() {
			conf.LoggregatorConfig.TLS.CACertFile = ""
		})

		It("should error", func() {
			Expect(err).To(MatchError(MatchRegexp("Loggregator CACert is empty")))
		})
	})

	When("Loggregator ClientCert is not set", func() {
		BeforeEach(func() {
			conf.LoggregatorConfig.TLS.CertFile = ""
		})

		It("should error", func() {
			Expect(err).To(MatchError(MatchRegexp("Loggregator ClientCert is empty")))
		})
	})

	When("Loggregator ClientKey is not set", func() {
		BeforeEach(func() {
			conf.LoggregatorConfig.TLS.KeyFile = ""
		})

		It("should error", func() {
			Expect(err).To(MatchError(MatchRegexp("Loggregator ClientKey is empty")))
		})
	})

	When("rate_limit.max_amount is <= zero", func() {
		BeforeEach(func() {
			conf.RateLimit.MaxAmount = 0
		})

		It("should err", func() {
			Expect(err).To(MatchError(MatchRegexp("RateLimit.MaxAmount is less than or equal to zero")))
		})
	})

	When("rate_limit.valid_duration is <= 0 ns", func() {
		BeforeEach(func() {
			conf.RateLimit.ValidDuration = 0 * time.Nanosecond
		})

		It("should err", func() {
			Expect(err).To(MatchError(MatchRegexp("RateLimit.ValidDuration is less than or equal to zero")))
		})
	})
})

var _ = Describe("toConfig credential helper conversion", func() {
	var (
		conf   rawConfig
		result Config
		err    error
	)

	BeforeEach(func() {
		conf = rawConfig{}
		conf.Server.Port = 8081
		conf.Logging.Level = "debug"
		conf.LoggregatorConfig.MetronAddress = "127.0.0.1:3458"
		conf.LoggregatorConfig.TLS.CACertFile = "../testcerts/ca.crt"
		conf.LoggregatorConfig.TLS.CertFile = "../testcerts/client.crt"
		conf.LoggregatorConfig.TLS.KeyFile = "../testcerts/client.crt"
		conf.Db = make(map[string]db.DatabaseConfig)
		conf.Db[db.PolicyDb] = db.DatabaseConfig{
			URL:                   "postgres://pqgotest:password@localhost/pqgotest",
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 60 * time.Second,
		}
		conf.Db[db.BindingDb] = db.DatabaseConfig{
			URL:                   "postgres://pqgotest:password@localhost/pqgotest",
			MaxOpenConnections:    10,
			MaxIdleConnections:    5,
			ConnectionMaxLifetime: 60 * time.Second,
		}
		conf.RateLimit.MaxAmount = 10
		conf.RateLimit.ValidDuration = 1 * time.Second
	})
	JustBeforeEach(func() {
		result, err = toConfig(conf)
	})

	When("cred_helper_impl is set to 'default'", func() {
		BeforeEach(func() {
			conf.CredHelperImpl = "default"
			conf.StoredProcedureConfig = nil
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should set CredentialHelperConfig to BasicAuthHandlingNative", func() {
			Expect(result.CredentialHelperConfig).To(BeAssignableToTypeOf(models.BasicAuthHandlingNative{}))
		})
	})
	When("cred_helper_impl is empty (not set)", func() {
		BeforeEach(func() {
			conf.CredHelperImpl = ""
			conf.StoredProcedureConfig = nil
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should set CredentialHelperConfig to BasicAuthHandlingNative", func() {
			Expect(result.CredentialHelperConfig).To(BeAssignableToTypeOf(models.BasicAuthHandlingNative{}))
		})
	})
	When("cred_helper_impl is set to 'stored_procedure'", func() {
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

		It("should set CredentialHelperConfig to BasicAuthHandlingStoredProc", func() {
			Expect(result.CredentialHelperConfig).To(BeAssignableToTypeOf(models.BasicAuthHandlingStoredProc{}))
		})

		It("should carry the StoredProcedureConfig with all fields", func() {
			storedProc := result.CredentialHelperConfig.(models.BasicAuthHandlingStoredProc)
			Expect(storedProc.Config.SchemaName).To(Equal("test_schema"))
			Expect(storedProc.Config.CreateBindingCredentialProcedureName).To(Equal("create_binding"))
			Expect(storedProc.Config.DropBindingCredentialProcedureName).To(Equal("drop_binding"))
			Expect(storedProc.Config.DropAllBindingCredentialProcedureName).To(Equal("drop_all_bindings"))
		})
	})
	When("cred_helper_impl is set to 'stored_procedure' but StoredProcedureConfig is nil", func() {
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
})
