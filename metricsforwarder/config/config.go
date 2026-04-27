package config

import (
	"errors"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"
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

var _ startup.ConfigValidator = Config{}

// GetLogging returns the logging configuration
func (c *Config) GetLogging() *helpers.LoggingConfig {
	return &c.Logging
}
