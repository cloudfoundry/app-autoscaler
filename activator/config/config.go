package config

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
)

const (
	DefaultLoggingLevel     = "info"
	DefaultServerPort       = 8080
	DefaultHealthServerPort = 8081
	DefaultCFServerPort     = 8082
)

// Config is the activator service configuration. The activator parks apps
// behind a route service while they are scaled to zero and wakes them on the
// first incoming request. See docs/design/scale-to-zero.md.
type Config struct {
	configutil.BaseConfig `yaml:",inline" json:",inline"`

	// RoutingAPI holds the credentials/endpoint the activator uses to subscribe
	// to route-registration events (readiness signal for scale-from-zero).
	// Left minimal for the PoC scaffold; wired up as the loops are implemented.
	RoutingAPI RoutingAPIConfig `yaml:"routing_api" json:"routing_api"`
}

// RoutingAPIConfig configures access to the CF routing-api event stream
// (GET /routing/v1/events), used as the app-readiness signal.
type RoutingAPIConfig struct {
	URL string `yaml:"url" json:"url"`
}

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	return configutil.GenericLoadConfig(filepath, vcapReader, defaultConfig, configutil.VCAPConfigurableFunc[Config](LoadVcapConfig))
}

func LoadVcapConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	if vcapReader.IsRunningOnCF() {
		if err := configutil.ApplyCommonVCAPConfiguration(conf, vcapReader, "activator-config"); err != nil {
			return err
		}
	}
	return nil
}

func defaultConfig() Config {
	return Config{
		BaseConfig: configutil.BaseConfig{
			Logging: helpers.LoggingConfig{
				Level: DefaultLoggingLevel,
			},
			Server: helpers.ServerConfig{
				Port: DefaultServerPort,
			},
			CFServer: helpers.ServerConfig{
				Port: DefaultCFServerPort,
			},
			Health: helpers.HealthConfig{
				ServerConfig: helpers.ServerConfig{
					Port: DefaultHealthServerPort,
				},
			},
			Db: make(map[string]db.DatabaseConfig),
		},
	}
}

func (c *Config) Validate() error {
	return c.Health.Validate()
}
