package config

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
)

type Config struct {
	configutil.BaseConfig `yaml:",inline" json:",inline"`
	CF                    cf.Config     `yaml:"cf" json:"cf"`
	PollInterval          time.Duration `yaml:"poll_interval" json:"poll_interval"`
	ReconcileInterval     time.Duration `yaml:"reconcile_interval" json:"reconcile_interval"`
	Window                time.Duration `yaml:"window" json:"window"`
}

func LoadConfig(path string, reader configutil.VCAPConfigurationReader) (*Config, error) {
	return configutil.GenericLoadConfig(path, reader, defaults, configutil.VCAPConfigurableFunc[Config](func(conf *Config, reader configutil.VCAPConfigurationReader) error {
		if reader.IsRunningOnCF() {
			return configutil.ApplyCommonVCAPConfiguration(conf, reader, "appstateobserver-config")
		}
		return nil
	}))
}

func defaults() Config {
	return Config{BaseConfig: configutil.BaseConfig{
		Logging: helpers.LoggingConfig{Level: "info"},
		Server:  helpers.ServerConfig{Port: 8080}, CFServer: helpers.ServerConfig{Port: 8082},
		Health: helpers.HealthConfig{ServerConfig: helpers.ServerConfig{Port: 8081}, ReadinessCheckEnabled: true},
		Db:     make(map[string]db.DatabaseConfig),
	}, PollInterval: 5 * time.Second, ReconcileInterval: 5 * time.Minute, Window: time.Minute}
}

func (c *Config) Validate() error {
	if err := c.CF.Validate(); err != nil {
		return err
	}
	if c.Db[db.PolicyDb].URL == "" || c.Db[db.ScalingEngineDb].URL == "" {
		return fmt.Errorf("observer requires policy_db and scalingengine_db")
	}
	if c.PollInterval <= 0 || c.ReconcileInterval <= 0 || c.Window < time.Second {
		return fmt.Errorf("observer intervals must be positive (window >= 1s)")
	}
	return c.Health.Validate()
}
