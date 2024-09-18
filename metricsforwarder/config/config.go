package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"gopkg.in/yaml.v3"
)

var (
	ErrReadYaml                       = errors.New("failed to read config file")
	ErrReadJson                       = errors.New("failed to read vcap_services json")
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

	if err := loadYamlFile(filepath, &conf); err != nil {
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
		CacheCleanupInterval: DefaultCacheCleanupInterval,
		PolicyPollerInterval: DefaultPolicyPollerInterval,
		RateLimit: models.RateLimitConfig{
			MaxAmount:     DefaultMaxAmount,
			ValidDuration: DefaultValidDuration,
		},
	}
}

func loadYamlFile(filepath string, conf *Config) error {
	if filepath == "" {
		return nil
	}
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to open config file '%s': %s\n", filepath, err)
		return ErrReadYaml
	}
	defer file.Close()

	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	if err := dec.Decode(conf); err != nil {
		return fmt.Errorf("%w: %v", ErrReadYaml, err)
	}
	return nil
}

func loadVcapConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	if !vcapReader.IsRunningOnCF() {
		return nil
	}

	conf.Server.Port = vcapReader.GetPort()
	if err := loadMetricsforwarderConfig(conf, vcapReader); err != nil {
		return err
	}

	if conf.Db == nil {
		conf.Db = make(map[string]db.DatabaseConfig)
	}

	if err := configurePolicyDb(conf, vcapReader); err != nil {
		return err
	}

	if conf.CredHelperImpl == "stored_procedure" {
		if err := configureStoredProcedureDb(conf, vcapReader); err != nil {
			return err
		}
	}

	if err := configureSyslogTLS(conf, vcapReader); err != nil {
		return err
	}

	return nil
}

func loadMetricsforwarderConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	data, err := vcapReader.GetServiceCredentialContent("config", "metricsforwarder")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMetricsforwarderConfigNotFound, err)
	}
	return yaml.Unmarshal(data, conf)
}

func configurePolicyDb(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {

	currentPolicyDb, ok := conf.Db[db.PolicyDb]
	if !ok {
		conf.Db[db.PolicyDb] = db.DatabaseConfig{}
	}

	dbURL, err := vcapReader.MaterializeDBFromService(db.PolicyDb)
	currentPolicyDb.URL = dbURL
	if err != nil {
		return err
	}
	conf.Db[db.PolicyDb] = currentPolicyDb
	return nil
}

func configureStoredProcedureDb(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	currentStoredProcedureDb, exists := conf.Db[db.StoredProcedureDb]
	if !exists {
		conf.Db[db.StoredProcedureDb] = db.DatabaseConfig{}
	}

	dbURL, err := vcapReader.MaterializeDBFromService(db.StoredProcedureDb)

	currentStoredProcedureDb.URL = dbURL
	parsedUrl, err := url.Parse(currentStoredProcedureDb.URL)
	if err != nil {
		return err
	}

	if conf.StoredProcedureConfig != nil {
		if conf.StoredProcedureConfig.Username != "" {
			currentStoredProcedureDb.URL = strings.Replace(currentStoredProcedureDb.URL, parsedUrl.User.Username(), conf.StoredProcedureConfig.Username, 1)
		}
		if conf.StoredProcedureConfig.Password != "" {
			bindingPassword, _ := parsedUrl.User.Password()
			currentStoredProcedureDb.URL = strings.Replace(currentStoredProcedureDb.URL, bindingPassword, conf.StoredProcedureConfig.Password, 1)
		}
	}
	conf.Db[db.StoredProcedureDb] = currentStoredProcedureDb

	return nil
}

func configureSyslogTLS(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
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
		return errors.New("Policy DB url is empty")
	}
	return nil
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
