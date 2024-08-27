package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"gopkg.in/yaml.v3"
)

// There are 3 type of errors that this package can return:
// - ErrReadYaml
// - ErrReadEnvironment
// - ErrReadVCAPEnvironment

var (
	ErrReadYaml                       = errors.New("failed to read config file")
	ErrReadJson                       = errors.New("failed to read vcap_services json")
	ErrMetricsforwarderConfigNotFound = errors.New("Configuration error: metricsforwarder config service not found")
)

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

func decodeYamlFile(filepath string, c *Config) error {
	r, err := os.Open(filepath)

	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to open config file '%s' : %s\n", filepath, err.Error())
		return err
	}

	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	err = dec.Decode(c)

	if err != nil {
		return fmt.Errorf("%w: %w", ErrReadYaml, err)
	}

	defer r.Close()
	return nil
}

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	var conf Config
	var err error

	conf = Config{
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

	if filepath != "" {
		err = decodeYamlFile(filepath, &conf)
		if err != nil {
			return nil, err
		}
	}

	if vcapReader.IsRunningOnCF() {
		conf.Server.Port = vcapReader.GetPort()

		data, err := vcapReader.GetServiceCredentialContent("config", "metricsforwarder")
		if err != nil {
			return &conf, fmt.Errorf("%w: %w", ErrMetricsforwarderConfigNotFound, err)
		}

		err = yaml.Unmarshal(data, &conf)
		if err != nil {
			return &conf, fmt.Errorf("%w: %w", ErrReadJson, err)
		}

		if conf.Db == nil {
			conf.Db = make(map[string]db.DatabaseConfig)
		}

		currentPolicyDb, ok := conf.Db[db.PolicyDb]
		if !ok {
			conf.Db[db.PolicyDb] = db.DatabaseConfig{}

		}

		currentPolicyDb.URL, err = vcapReader.MaterializeDBFromService(db.PolicyDb)
		if err != nil {
			return &conf, err
		}
		conf.Db[db.PolicyDb] = currentPolicyDb

		if conf.CredHelperImpl == "stored_procedure" {
			currentStoredProcedureDb, ok := conf.Db[db.StoredProcedureDb]
			if !ok {
				conf.Db[db.StoredProcedureDb] = db.DatabaseConfig{}
			}
			currentStoredProcedureDb.URL, err = vcapReader.MaterializeDBFromService(db.StoredProcedureDb)
			if err != nil {
				return &conf, err
			}
			conf.Db[db.StoredProcedureDb] = currentStoredProcedureDb
		}

		conf.SyslogConfig.TLS, err = vcapReader.MaterializeTLSConfigFromService("syslog-client")
		if err != nil {
			return &conf, err
		}
	}

	return &conf, nil
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
