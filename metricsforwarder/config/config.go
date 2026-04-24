package config

import (
	"errors"
	"time"

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
	Logging              helpers.LoggingConfig
	Server               helpers.ServerConfig
	LoggregatorConfig    LoggregatorConfig
	SyslogConfig         SyslogConfig
	Db                   map[string]db.DatabaseConfig
	CacheTTL             time.Duration
	CacheCleanupInterval time.Duration
	PolicyPollerInterval time.Duration
	Health               helpers.HealthConfig
	RateLimit            models.RateLimitConfig

	// CredentialHelperConfig configures how credentials for "Basic Authentication" are managed.
	// nil means no credential helper is configured (Basic Auth disabled).
	CredentialHelperConfig models.BasicAuthHandlingImplConfig
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
	if c.CredentialHelperConfig == nil {
		return errors.New("CredentialHelperConfig is not configured")
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
