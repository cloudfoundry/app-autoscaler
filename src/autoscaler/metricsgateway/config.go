package metricsgateway

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	DefaultLoggingLevel                      = "info"
	DefaultAppRefreshInterval  time.Duration = 60 * time.Second
	DefaultWSReconnectInterval time.Duration = 30 * time.Second
	DefaultNozzleCount                       = 3
	DefaultEventChannelSize                  = 500
)

type DispatcherConfig struct {
	AppRefreshInterval time.Duration     `yaml:"app_refresh_interval"`
	PolicyDB           db.DatabaseConfig `yaml:"policy_db"`
}

type NozzleConfig struct {
	TLS     models.TLSCerts `yaml:"tls"`
	RLPAddr string          `yaml:"rlp_addr`
}

type EmitterConfig struct {
	TLS               models.TLSCerts `yaml:"tls"`
	ReconnectInterval time.Duration   `yaml:"reconnect_interval"`
}

type Config struct {
	Logging            helpers.LoggingConfig `yaml:"logging"`
	NozzleCount        int                   `yaml:"Nozzle_count"`
	EventChannelSize   int                   `yaml:"event_channel_size"`
	MetricsServerAddrs []string              `yaml:"metrics_server_addrs"`
	Dispatcher         DispatcherConfig      `yaml:"dispatcher"`
	Emitter            EmitterConfig         `yaml:"emitter"`
	Nozzle             NozzleConfig          `yaml:"nozzle"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Logging: helpers.LoggingConfig{
			Level: DefaultLoggingLevel,
		},
		NozzleCount:      DefaultNozzleCount,
		EventChannelSize: DefaultEventChannelSize,
		Dispatcher: DispatcherConfig{
			AppRefreshInterval: DefaultAppRefreshInterval,
		},
		Emitter: EmitterConfig{
			ReconnectInterval: DefaultWSReconnectInterval,
		},
	}

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, conf)
	if err != nil {
		return nil, err
	}

	conf.Logging.Level = strings.ToLower(conf.Logging.Level)

	return conf, nil
}

func (c *Config) Validate() error {
	if c.NozzleCount <= 0 {
		return fmt.Errorf("Configuration error: nozzle_count <= 0")
	}
	if c.EventChannelSize <= 0 {
		return fmt.Errorf("Configuration error: event_channel_size <= 0")
	}

	if len(c.MetricsServerAddrs) <= 0 {
		return fmt.Errorf("Configuration error: metrics_server_addrs is empty")
	}

	if c.Dispatcher.PolicyDB.URL == "" {
		return fmt.Errorf("Configuration error: dispatcher.policy_db.url is empty")
	}
	if c.Dispatcher.AppRefreshInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: dispatcher.app_refresh_interval is 0")
	}

	if c.Nozzle.RLPAddr == "" {
		return fmt.Errorf("COnfiguration error: nozzle.rlp_addr is empty")
	}

	if c.Emitter.ReconnectInterval == time.Duration(0) {
		return fmt.Errorf("Configuratoin error: emitter.reconnect_interval is 0")
	}
	return nil
}
