package config

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"gopkg.in/yaml.v3"
)

// ErrInvalidPort is returned when the PORT environment variable is not a valid port number
var ErrInvalidPort = fmt.Errorf("Invalid port number in PORT environment variable")

const (
	DefaultMetronAddress        = "127.0.0.1:3458"
	DefaultCacheTTL             = 15 * time.Minute
	DefaultCacheCleanupInterval = 6 * time.Hour
	DefaultPolicyPollerInterval = 40 * time.Second
	DefaultMaxAmount            = 10
	DefaultValidDuration        = 1 * time.Second
)

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

var defaultServerConfig = helpers.ServerConfig{
	Port: 6110,
}

var defaultHealthConfig = helpers.HealthConfig{
	ServerConfig: helpers.ServerConfig{
		Port: 8081,
	},
}

var defaultLoggingConfig = helpers.LoggingConfig{
	Level: "info",
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type LoggregatorConfig struct {
	MetronAddress string          `yaml:"metron_address"`
	TLS           models.TLSCerts `yaml:"tls"`
}

type SyslogConfig struct {
	ServerAddress string          `yaml:"server_address"`
	Port          int             `yaml:"port"`
	TLS           models.TLSCerts `yaml:"tls"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Server:  defaultServerConfig,
		Logging: defaultLoggingConfig,
		LoggregatorConfig: LoggregatorConfig{
			MetronAddress: DefaultMetronAddress,
		},
		Health:               defaultHealthConfig,
		CacheTTL:             DefaultCacheTTL,
		CacheCleanupInterval: DefaultCacheCleanupInterval,
		PolicyPollerInterval: DefaultPolicyPollerInterval,
		RateLimit: models.RateLimitConfig{
			MaxAmount:     DefaultMaxAmount,
			ValidDuration: DefaultValidDuration,
		},
	}

	dec := yaml.NewDecoder(reader)
	dec.KnownFields(true)
	err := dec.Decode(conf)

	if os.Getenv("PORT") != "" {
		port := os.Getenv("PORT")
		portNumber, err := strconv.Atoi(port)
		if err != nil {
			return nil, ErrInvalidPort
		}
		conf.Server.Port = portNumber
	}

	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (c *Config) UsingSyslog() bool {
	return c.SyslogConfig.ServerAddress != "" && c.SyslogConfig.Port != 0
}

func (c *Config) Validate() error {
	if c.Db[db.PolicyDb].URL == "" {
		return fmt.Errorf("Configuration error: Policy DB url is empty")
	}
	if c.UsingSyslog() {
		if c.SyslogConfig.TLS.CACertFile == "" {
			return fmt.Errorf("Configuration error: SyslogServer Loggregator CACert is empty")
		}
		if c.SyslogConfig.TLS.CertFile == "" {
			return fmt.Errorf("Configuration error: SyslogServer ClientCert is empty")
		}
		if c.SyslogConfig.TLS.KeyFile == "" {
			return fmt.Errorf("Configuration error: SyslogServer ClientKey is empty")
		}
	} else {
		if c.LoggregatorConfig.TLS.CACertFile == "" {
			return fmt.Errorf("Configuration error: Loggregator CACert is empty")
		}
		if c.LoggregatorConfig.TLS.CertFile == "" {
			return fmt.Errorf("Configuration error: Loggregator ClientCert is empty")
		}
		if c.LoggregatorConfig.TLS.KeyFile == "" {
			return fmt.Errorf("Configuration error: Loggregator ClientKey is empty")
		}
	}

	if c.RateLimit.MaxAmount <= 0 {
		return fmt.Errorf("Configuration error: RateLimit.MaxAmount is equal or less than zero")
	}
	if c.RateLimit.ValidDuration <= 0*time.Nanosecond {
		return fmt.Errorf("Configuration error: RateLimit.ValidDuration is equal or less than zero nanosecond")
	}
	if c.CredHelperImpl == "" {
		return fmt.Errorf("Configuration error: CredHelperImpl is empty")
	}

	if err := c.Health.Validate(); err != nil {
		return err
	}

	return nil
}
