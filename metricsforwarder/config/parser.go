package config

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type rawConfig struct {
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

func toConfig(raw rawConfig) (Config, error) {
	return Config{}, models.ErrUnimplemented
}

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	raw := defaultConfig()

	if err := helpers.LoadYamlFile(filepath, &raw); err != nil {
		return nil, err
	}

	if err := loadVcapConfig(&raw, vcapReader); err != nil {
		return nil, err
	}

	config, err := toConfig(raw)
	if err != nil {
		return nil, err
	}

	return &config, nil
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
		var impl models.BasicAuthHandlingImplConfig
		if conf.CredHelperImpl == "stored_procedure" && conf.StoredProcedureConfig != nil {
			impl = models.BasicAuthHandlingStoredProc{Config: *conf.StoredProcedureConfig}
		} else {
			impl = models.BasicAuthHandlingNative{}
		}
		basicAuthImplCfg = &impl
	}
	if err := vcapReader.ConfigureDatabases(&conf.Db, basicAuthImplCfg); err != nil {
		return err
	}

	tls, err := vcapReader.MaterializeTLSConfigFromService("syslog-client")
	if err != nil {
		return err
	}
	conf.SyslogConfig.TLS = tls

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
