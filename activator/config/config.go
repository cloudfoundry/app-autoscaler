package config

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	DefaultLoggingLevel     = "info"
	DefaultServerPort       = 8080
	DefaultHealthServerPort = 8081
	DefaultCFServerPort     = 8082
	DefaultReadinessTimeout = 30 * time.Second
	DefaultRouteServiceUPSI = "autoscaler-activator-rs"
)

// Config is the activator service configuration. The activator parks apps
// behind a route service while they are scaled to zero and wakes them on the
// first incoming request. See docs/design/scale-to-zero.md.
type Config struct {
	configutil.BaseConfig `yaml:",inline" json:",inline"`

	// CF is used to bind/unbind app routes to the activator route-service.
	CF cf.Config `yaml:"cf" json:"cf"`

	// RouteServiceUPSIName is the name of the user-provided service instance
	// (created at deploy time) whose route_service_url points at this activator.
	RouteServiceUPSIName string `yaml:"route_service_upsi_name" json:"route_service_upsi_name"`

	// ReadinessTimeout bounds how long a held request waits for the app to
	// become ready before returning 503 + Retry-After.
	ReadinessTimeout time.Duration `yaml:"readiness_timeout" json:"readiness_timeout"`

	// ScalingEngine is used to wake parked apps.
	ScalingEngine ScalingEngineConfig `yaml:"scaling_engine" json:"scaling_engine"`

	// RoutingAPI configures access to the CF routing-api event stream
	// (GET /routing/v1/events), the app-readiness signal.
	RoutingAPI RoutingAPIConfig `yaml:"routing_api" json:"routing_api"`
}

// ScalingEngineConfig points the activator at the scaling engine.
type ScalingEngineConfig struct {
	ScalingEngineURL string          `yaml:"scaling_engine_url" json:"scaling_engine_url"`
	TLSClientCerts   models.TLSCerts `yaml:"tls" json:"tls"`
}

// RoutingAPIConfig configures access to the CF routing-api. URL must be the
// base (e.g. https://api.<domain>) WITHOUT a /routing/v1 suffix — the routing-api
// client appends the /routing/v1/events path itself.
type RoutingAPIConfig struct {
	URL               string          `yaml:"url" json:"url"`
	SkipSSLValidation bool            `yaml:"skip_ssl_validation" json:"skip_ssl_validation"`
	UAACreds          models.UAACreds `yaml:"uaa" json:"uaa"`
}

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	return configutil.GenericLoadConfig(filepath, vcapReader, defaultConfig, configutil.VCAPConfigurableFunc[Config](LoadVcapConfig))
}

func LoadVcapConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	if vcapReader.IsRunningOnCF() {
		if err := configutil.ApplyCommonVCAPConfiguration(conf, vcapReader, "activator-config"); err != nil {
			return err
		}
		conf.ScalingEngine.TLSClientCerts = vcapReader.GetInstanceTLSCerts()
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
		RouteServiceUPSIName: DefaultRouteServiceUPSI,
		ReadinessTimeout:     DefaultReadinessTimeout,
	}
}

func (c *Config) Validate() error {
	return c.Health.Validate()
}
