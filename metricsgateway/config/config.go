package config

import (
	"errors"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type SyslogConfig struct {
	ServerAddress string          `yaml:"server_address" json:"server_address"`
	Port          int             `yaml:"port" json:"port"`
	TLS           models.TLSCerts `yaml:"tls" json:"tls"`
}

type Config struct {
	configutil.BaseConfig `yaml:",inline"`
	SyslogConfig          SyslogConfig `yaml:"syslog" json:"syslog"`
	ValidOrgGuids         []string     `yaml:"valid_org_guids" json:"valid_org_guids"`
}

func defaultConfig() Config {
	return Config{
		BaseConfig: configutil.BaseConfig{
			Logging: helpers.LoggingConfig{Level: "info"},
			CFServer: helpers.ServerConfig{
				Port: 8080,
			},
			Health: helpers.HealthConfig{
				ServerConfig: helpers.ServerConfig{Port: 8081},
			},
			Db: make(map[string]db.DatabaseConfig),
		},
		SyslogConfig: SyslogConfig{
			ServerAddress: "log-cache.service.cf.internal",
			Port:          6067,
		},
	}
}

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	return configutil.GenericLoadConfig(filepath, vcapReader, defaultConfig, configutil.VCAPConfigurableFunc[Config](loadVcapConfig))
}

func loadVcapConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	if !vcapReader.IsRunningOnCF() {
		return nil
	}

	conf.Logging.PlainTextSink = true
	conf.CFServer.Port = vcapReader.GetPort()
	conf.Server.Port = 0

	if err := configutil.LoadConfig(conf, vcapReader, "metricsgateway-config"); err != nil {
		return err
	}

	// Configure CF instance identity certs for server TLS
	conf.CFServer.TLS = vcapReader.GetInstanceTLSCerts()

	tls, err := vcapReader.MaterializeTLSConfigFromService("syslog-client")
	if err != nil {
		return err
	}
	conf.SyslogConfig.TLS = tls

	if len(conf.ValidOrgGuids) == 0 {
		if orgGuid := vcapReader.GetOrgGuid(); orgGuid != "" {
			conf.ValidOrgGuids = []string{orgGuid}
		}
	}

	return nil
}

func (c *Config) Validate() error {
	if len(c.ValidOrgGuids) == 0 {
		return errors.New("configuration error: valid_org_guids must contain at least one org GUID")
	}
	if c.SyslogConfig.ServerAddress == "" {
		return errors.New("configuration error: syslog server_address is empty")
	}
	if c.SyslogConfig.Port == 0 {
		return errors.New("configuration error: syslog port is zero")
	}
	return c.Health.Validate()
}
