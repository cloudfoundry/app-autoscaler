package config

import (
	"errors"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

var (
	ErrMetricsforwarderConfigNotFound = errors.New("metricsforwarder config service not found")
)

const (
	DefaultMetronAddress        = "127.0.0.1:3458"
	DefaultCacheTTL             = 15 * time.Minute
	DefaultCacheCleanupInterval = 6 * time.Hour
	DefaultPolicyPollerInterval = 40 * time.Second
	DefaultMaxAmount            = 10
	DefaultValidDuration        = 1 * time.Second
)

type LoggregatorConfig struct {
	MetronAddress string          `yaml:"metron_address"`
	TLS           models.TLSCerts `yaml:"tls"`
}
type SyslogConfig struct {
	ServerAddress string          `yaml:"server_address"`
	Port          int             `yaml:"port"`
	TLS           models.TLSCerts `yaml:"tls"`
}

type Config struct {
	Logging               helpers.LoggingConfig         `yaml:"logging"`
	Server                helpers.ServerConfig          `yaml:"server"`
	LoggregatorConfig     LoggregatorConfig             `yaml:"loggregator"`
	SyslogConfig          SyslogConfig                  `yaml:"syslog"`
	Db                    map[string]db.DatabaseConfig  `yaml:"db"`
	CacheTTL              time.Duration                 `yaml:"cache_ttl"`
	CacheCleanupInterval  time.Duration                 `yaml:"cache_cleanup_interval"`
	PolicyPollerInterval  time.Duration                 `yaml:"policy_poller_interval"`
	Health                helpers.HealthConfig          `yaml:"health"`
	RateLimit             models.RateLimitConfig        `yaml:"rate_limit"`
	CredHelperImpl        string                        `yaml:"cred_helper_impl"`
	StoredProcedureConfig *models.StoredProcedureConfig `yaml:"stored_procedure_binding_credential_config"`
}

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	conf := defaultConfig()

	if err := helpers.LoadYamlFile(filepath, &conf); err != nil {
		return nil, err
	}

	if err := loadVcapConfig(&conf, vcapReader); err != nil {
		return nil, err
	}

	return &conf, nil
}

func defaultConfig() Config {
	return Config{
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

func loadVcapConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	if !vcapReader.IsRunningOnCF() {
		return nil
	}

	conf.Server.Port = vcapReader.GetPort()
	if err := configutil.LoadConfig(&conf, vcapReader, "metricsforwarder-config"); err != nil {
		return err
	}

	if err := vcapReader.ConfigureDatabases(&conf.Db, conf.StoredProcedureConfig, conf.CredHelperImpl); err != nil {
		return err
	}

	tls, err := vcapReader.MaterializeTLSConfigFromService("syslog-client")
	if err != nil {
		return err
	}
	conf.SyslogConfig.TLS = tls

	return nil
}

func (c *Config) Validate() error {
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

func (c *Config) validateDbConfig() error {
	if c.Db[db.PolicyDb].URL == "" {
		return errors.New("configuration error: Policy DB url is empty")
	}
	if c.Db[db.BindingDb].URL == "" {
		return errors.New("configuration error: Binding DB url is empty")
	}
	if c.UsingSyslog() {
		return c.validateSyslogConfig()
	}
	return c.validateLoggregatorConfig()
}
func (c *Config) validateSyslogOrLoggregator() error {
	if c.UsingSyslog() {
		return c.validateSyslogConfig()
	}
	return c.validateLoggregatorConfig()
}
func (c *Config) validateSyslogConfig() error {
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

func (c *Config) validateLoggregatorConfig() error {
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

func (c *Config) validateRateLimit() error {
	if c.RateLimit.MaxAmount <= 0 {
		return errors.New("RateLimit.MaxAmount is less than or equal to zero")
	}
	if c.RateLimit.ValidDuration <= 0 {
		return errors.New("RateLimit.ValidDuration is less than or equal to zero")
	}
	return nil
}

func (c *Config) validateCredHelperImpl() error {
	if c.CredHelperImpl == "" {
		return errors.New("CredHelperImpl is empty")
	}
	return nil
}

func (c *Config) UsingSyslog() bool {
	return c.SyslogConfig.ServerAddress != "" && c.SyslogConfig.Port != 0
}

// GetLogging returns the logging configuration
func (c *Config) GetLogging() *helpers.LoggingConfig {
	return &c.Logging
}
