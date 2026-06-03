package config

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type rawConfig struct {
	Logging              helpers.LoggingConfig        `yaml:"logging"`
	Server               helpers.ServerConfig         `yaml:"server"`
	LoggregatorConfig    LoggregatorConfig            `yaml:"loggregator"`
	SyslogConfig         SyslogConfig                 `yaml:"syslog"`
	MetricsGateway       MetricsGatewayConfig         `yaml:"metrics_gateway" json:"metrics_gateway"`
	InstanceTLSCerts     models.TLSCerts              `yaml:"-" json:"-"`
	Db                   map[string]db.DatabaseConfig `yaml:"db"`
	CacheTTL             time.Duration                `yaml:"cache_ttl"`
	CacheCleanupInterval time.Duration                `yaml:"cache_cleanup_interval"`
	PolicyPollerInterval time.Duration                `yaml:"policy_poller_interval"`
	Health               helpers.HealthConfig         `yaml:"health"`
	RateLimit            models.RateLimitConfig       `yaml:"rate_limit"`

	BasicAuthForCustomMetrics string                        `yaml:"basic_auth_for_custom_metrics" json:"basic_auth_for_custom_metrics"`
	CredHelperImpl            string                        `yaml:"cred_helper_impl"`
	StoredProcedureConfig     *models.StoredProcedureConfig `yaml:"stored_procedure_binding_credential_config"`
}

func toConfig(rawConfig rawConfig) (Config, error) {
	err := rawConfig.validate()
	if err != nil {
		return Config{}, fmt.Errorf("input-validation failed: %w", err)
	}

	result := Config{
		Logging:              rawConfig.Logging,
		Server:               rawConfig.Server,
		LoggregatorConfig:    rawConfig.LoggregatorConfig,
		SyslogConfig:         rawConfig.SyslogConfig,
		MetricsGateway:       rawConfig.MetricsGateway,
		InstanceTLSCerts:     rawConfig.InstanceTLSCerts,
		Db:                   rawConfig.Db,
		CacheTTL:             rawConfig.CacheTTL,
		CacheCleanupInterval: rawConfig.CacheCleanupInterval,
		PolicyPollerInterval: rawConfig.PolicyPollerInterval,
		Health:               rawConfig.Health,
		RateLimit:            rawConfig.RateLimit,
	}

	switch rawConfig.CredHelperImpl {
	case "stored_procedure":
		result.CredentialHelperConfig = models.BasicAuthHandlingStoredProc{
			Config: *rawConfig.StoredProcedureConfig,
		}
	case "default", "":
		result.CredentialHelperConfig = models.BasicAuthHandlingNative{}
	case "disabled":
		result.CredentialHelperConfig = nil
	}

	return result, nil
}

func (c *rawConfig) validate() error {
	if err := c.validateDbConfig(); err != nil {
		return err
	}
	if err := c.validateSyslogOrLoggregator(); err != nil {
		return err
	}
	if err := c.validateRateLimit(); err != nil {
		return err
	}
	if err := c.validateCredHelperImpl(); err != nil {
		return err
	}
	return c.Health.Validate()
}

func (c *rawConfig) validateDbConfig() error {
	if c.Db[db.PolicyDb].URL == "" {
		return errors.New("configuration error: Policy DB url is empty")
	}
	if c.Db[db.BindingDb].URL == "" {
		return errors.New("configuration error: Binding DB url is empty")
	}
	return nil
}
func (c *rawConfig) validateSyslogOrLoggregator() error {
	if c.usingGateway() {
		return nil
	}
	if c.usingSyslog() {
		return c.validateSyslogConfig()
	}
	return c.validateLoggregatorConfig()
}
func (c *rawConfig) validateSyslogConfig() error {
	if c.SyslogConfig.TLS.CACertFile == "" {
		return errors.New("SyslogServer Loggregator CACert is empty")
	}
	if c.SyslogConfig.TLS.CertFile == "" {
		return errors.New("SyslogServer ClientCert is empty")
	}
	if c.SyslogConfig.TLS.KeyFile == "" {
		return errors.New("SyslogServer ClientKey is empty")
	}
	return nil
}

func (c *rawConfig) validateLoggregatorConfig() error {
	if c.LoggregatorConfig.TLS.CACertFile == "" {
		return errors.New("Loggregator CACert is empty")
	}
	if c.LoggregatorConfig.TLS.CertFile == "" {
		return errors.New("Loggregator ClientCert is empty")
	}
	if c.LoggregatorConfig.TLS.KeyFile == "" {
		return errors.New("Loggregator ClientKey is empty")
	}
	return nil
}

func (c *rawConfig) validateRateLimit() error {
	if c.RateLimit.MaxAmount <= 0 {
		return errors.New("RateLimit.MaxAmount is less than or equal to zero")
	}
	if c.RateLimit.ValidDuration <= 0 {
		return errors.New("RateLimit.ValidDuration is less than or equal to zero")
	}
	return nil
}

func (c *rawConfig) validateCredHelperImpl() error {
	switch c.CredHelperImpl {
	case "stored_procedure":
		if c.StoredProcedureConfig == nil {
			msg := "CredHelperImpl is set to stored_procedure but there is no StoredProcedureConfig"
			return errors.New(msg)
		}
		if c.StoredProcedureConfig.CreateBindingCredentialProcedureName == "" {
			msg := "CredHelperImpl is set to stored_procedure but the StoredProcedureConfig does not contain the CreateBindingCredentialProcedureName"
			return errors.New(msg)
		}
		if c.StoredProcedureConfig.DropAllBindingCredentialProcedureName == "" {
			msg := "CredHelperImpl is set to stored_procedure but the StoredProcedureConfig does not contain the DeleteBindingCredentialProcedureName"
			return errors.New(msg)
		}
		if c.StoredProcedureConfig.DropBindingCredentialProcedureName == "" {
			msg := "CredHelperImpl is set to stored_procedure but the StoredProcedureConfig does not contain the DropBindingCredentialProcedureName"
			return errors.New(msg)
		}
		if c.StoredProcedureConfig.SchemaName == "" {
			msg := "CredHelperImpl is set to stored_procedure but the StoredProcedureConfig does not contain the SchemaName"
			return errors.New(msg)
		}
		if c.StoredProcedureConfig.ValidateBindingCredentialProcedureName == "" {
			msg := "CredHelperImpl is set to stored_procedure but the StoredProcedureConfig does not contain the ValidateBindingCredentialProcedureName"
			return errors.New(msg)
		}
	case "", "default":
		if c.StoredProcedureConfig != nil {
			msg := "CredHelperImpl is set to the default but there is a StoredProcedureConfig"
			return errors.New(msg)
		}
	case "disabled":
	// Nothing to validate if the credential helper is disabled.
	default:
		msg := fmt.Sprintf("CredHelperImpl is set to %s which is not a supported value", c.CredHelperImpl)
		return errors.New(msg)
	}

	return nil
}

func (c *rawConfig) usingSyslog() bool {
	return c.SyslogConfig.ServerAddress != "" && c.SyslogConfig.Port != 0
}

func (c *rawConfig) usingGateway() bool {
	return c.MetricsGateway.URL != ""
}

// ================================================================================
// Legacy parsing-machinery
// ================================================================================
//
// 🏚️ This is legacy spaghetti-code from the migration from "Bosh" to the "Cloud Controller". Afer removing the support for "Bosh", revisiting makes sense. Ideally, a signature like `func FromCFEnv(env map[string]string) (Config, error)` would be best.

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	rawConfig, err := configutil.GenericLoadConfig(filepath, vcapReader, defaultConfig, configutil.VCAPConfigurableFunc[rawConfig](loadVcapConfig))
	if err != nil {
		return nil, err
	}

	config, err := toConfig(*rawConfig)
	return &config, err
}

func loadVcapConfig(conf *rawConfig, vcapReader configutil.VCAPConfigurationReader) error {
	if !vcapReader.IsRunningOnCF() {
		return nil
	}

	conf.Server.Port = vcapReader.GetPort()
	if err := configutil.LoadConfig(&conf, vcapReader, "metricsforwarder-config"); err != nil {
		return err
	}

	var basicAuthImplCfg *models.BasicAuthHandlingImplConfig
	if conf.CredHelperImpl != "" {
		var impl models.BasicAuthHandlingImplConfig = nil
		if conf.CredHelperImpl == "stored_procedure" && conf.StoredProcedureConfig != nil {
			impl = models.BasicAuthHandlingStoredProc{Config: *conf.StoredProcedureConfig}
			basicAuthImplCfg = &impl
		} else if slices.Contains([]string{"", "default"}, conf.CredHelperImpl) {
			impl = models.BasicAuthHandlingNative{}
			basicAuthImplCfg = &impl
		} else {
			basicAuthImplCfg = nil
		}
	}
	if err := vcapReader.ConfigureDatabases(&conf.Db, basicAuthImplCfg); err != nil {
		return err
	}

	if conf.usingGateway() {
		conf.InstanceTLSCerts = vcapReader.GetInstanceTLSCerts()
	} else {
		tls, err := vcapReader.MaterializeTLSConfigFromService("syslog-client")
		if err != nil {
			return err
		}
		conf.SyslogConfig.TLS = tls
	}

	return nil
}

func defaultConfig() rawConfig {
	return rawConfig{
		Server:  helpers.ServerConfig{Port: 6110},
		Logging: helpers.LoggingConfig{Level: "info"},
		LoggregatorConfig: LoggregatorConfig{
			MetronAddress: DefaultMetronAddress,
		},
		Health:               helpers.HealthConfig{ServerConfig: helpers.ServerConfig{Port: 8081}},
		CacheTTL:             DefaultCacheTTL,
		Db:                   make(map[string]db.DatabaseConfig),
		CacheCleanupInterval: DefaultCacheCleanupInterval,
		PolicyPollerInterval: DefaultPolicyPollerInterval,
		RateLimit: models.RateLimitConfig{
			MaxAmount:     DefaultMaxAmount,
			ValidDuration: DefaultValidDuration,
		},
	}
}
