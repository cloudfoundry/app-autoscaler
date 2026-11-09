package config

import (
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/activator"
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
	DefaultReadinessTimeout = 120 * time.Second
)

// Config is the activator service configuration. The activator keeps a
// zero-instance app's route routable (by registering itself as an mTLS backend
// on the Gorouter NATS bus) and wakes the app on the first incoming request.
// See docs/design/scale-to-zero.md.
type Config struct {
	configutil.BaseConfig `yaml:",inline" json:",inline"`

	// CF is used to list an app's routes.
	CF cf.Config `yaml:"cf" json:"cf"`

	// Nats configures the connection used to register the activator as a route
	// backend on the Gorouter NATS bus (keeps parked routes routable).
	Nats activator.NatsConfig `yaml:"nats" json:"nats"`

	// ReadinessTimeout bounds how long a held request waits for the app to
	// become ready before returning 503 + Retry-After.
	ReadinessTimeout time.Duration `yaml:"readiness_timeout" json:"readiness_timeout"`

	// ScalingEngine is used to wake parked apps.
	ScalingEngine ScalingEngineConfig `yaml:"scaling_engine" json:"scaling_engine"`
}

// ScalingEngineConfig points the activator at the scaling engine.
type ScalingEngineConfig struct {
	ScalingEngineURL string          `yaml:"scaling_engine_url" json:"scaling_engine_url"`
	TLSClientCerts   models.TLSCerts `yaml:"tls" json:"tls"`
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

		// NATS client mTLS certs come from the bound nats-client service
		// (materialized to files), mirroring the syslog-client pattern.
		natsTLS, err := vcapReader.MaterializeTLSConfigFromService("nats-client")
		if err != nil {
			return err
		}
		conf.Nats.TLS = natsTLS
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
		ReadinessTimeout: DefaultReadinessTimeout,
	}
}

func (c *Config) Validate() error {
	return c.Health.Validate()
}
