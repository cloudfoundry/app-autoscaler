package config

import (
	"autoscaler/db"
	"autoscaler/helpers"
	"autoscaler/models"
	"fmt"
	"strings"

	"time"

	yaml "gopkg.in/yaml.v2"
)

const (
	DefaultShardID                          = "CF_AUTOSCALER"
	DefaultLoggingLevel                     = "info"
	DefaultAppRefreshInterval time.Duration = 60 * time.Second
	DefaultHandshakeTimeout   time.Duration = 500 * time.Millisecond
	DefaultKeepAliveInterval  time.Duration = 5 * time.Second
	DefaultNozzleCount                      = 3
	DefaultEnvelopChanSize                  = 500
	DefaultEmitterBufferSize                = 500
)

type AppManagerConfig struct {
	AppRefreshInterval time.Duration     `yaml:"app_refresh_interval"`
	PolicyDB           db.DatabaseConfig `yaml:"policy_db"`
}

type NozzleConfig struct {
	TLS     *models.TLSCerts `yaml:"tls"`
	RLPAddr string           `yaml:"rlp_addr"`
	ShardID string           `yaml:"shard_id"`
}

type EmitterConfig struct {
	TLS               *models.TLSCerts `yaml:"tls"`
	BufferSize        int              `yaml:"buffer_size"`
	KeepAliveInterval time.Duration    `yaml:"keep_alive_interval"`
	HandshakeTimeout  time.Duration    `yaml:"handshake_timeout"`
}

type Config struct {
	Logging           helpers.LoggingConfig `yaml:"logging"`
	EnvelopChanSize   int                   `yaml:"envelop_chan_size"`
	NozzleCount       int                   `yaml:"nozzle_count"`
	MetricServerAddrs []string              `yaml:"metric_server_addrs"`
	AppManager        AppManagerConfig      `yaml:"app_manager"`
	Emitter           EmitterConfig         `yaml:"emitter"`
	Nozzle            NozzleConfig          `yaml:"nozzle"`
	Health            models.HealthConfig   `yaml:"health"`
}

func LoadConfig(bytes []byte) (*Config, error) {
	conf := &Config{
		Logging: helpers.LoggingConfig{
			Level: DefaultLoggingLevel,
		},
		EnvelopChanSize: DefaultEnvelopChanSize,
		NozzleCount:     DefaultNozzleCount,
		Emitter: EmitterConfig{
			BufferSize:        DefaultEmitterBufferSize,
			KeepAliveInterval: DefaultKeepAliveInterval,
			HandshakeTimeout:  DefaultHandshakeTimeout,
		},
		Nozzle: NozzleConfig{
			ShardID: DefaultShardID,
		},
		AppManager: AppManagerConfig{
			AppRefreshInterval: DefaultAppRefreshInterval,
		},
	}

	err := yaml.Unmarshal(bytes, conf)
	if err != nil {
		return nil, err
	}

	conf.Logging.Level = strings.ToLower(conf.Logging.Level)
	return conf, nil
}

func (c *Config) Validate() error {
	if c.NozzleCount <= 0 {
		return fmt.Errorf("Configuration error: nozzle_count is less-equal than 0")
	}
	if c.EnvelopChanSize <= 0 {
		return fmt.Errorf("Configuration error: envelope_chan_size is less-equal than 0")
	}

	if len(c.MetricServerAddrs) <= 0 {
		return fmt.Errorf("Configuration error: metrics_server_addrs is empty")
	}

	if c.AppManager.PolicyDB.URL == "" {
		return fmt.Errorf("Configuration error: app_manager.policy_db.url is empty")
	}
	if c.AppManager.PolicyDB.MaxOpenConnections <= 0 {
		return fmt.Errorf("Configuration error: app_manager.policy_db.max_open_connections is less-equal than 0")
	}
	if c.AppManager.PolicyDB.MaxIdleConnections <= 0 {
		return fmt.Errorf("Configuration error: app_manager.policy_db.max_idle_connections is less-equal than 0")
	}
	if c.AppManager.PolicyDB.ConnectionMaxLifetime == time.Duration(0) {
		return fmt.Errorf("Configuration error: app_manager.policy_db.connection_max_lifetime is 0")
	}
	if c.AppManager.AppRefreshInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: app_manager.app_refresh_interval is 0")
	}
	if c.Emitter.BufferSize <= 0 {
		return fmt.Errorf("Configuration error: emitter.buffer_size is less-equal than 0")
	}
	if c.Emitter.HandshakeTimeout == time.Duration(0) {
		return fmt.Errorf("Configuration error: emitter.handshake_timeout is 0")
	}
	if c.Emitter.KeepAliveInterval == time.Duration(0) {
		return fmt.Errorf("Configuration error: emitter.keep_alive_interval is 0")
	}
	if c.Emitter.TLS.CertFile == "" {
		return fmt.Errorf("Configuration error: emitter.tls.cert_file is empty")
	}
	if c.Emitter.TLS.KeyFile == "" {
		return fmt.Errorf("Configuration error: emitter.tls.key_file is empty")
	}
	if c.Emitter.TLS.CACertFile == "" {
		return fmt.Errorf("Configuration error: emitter.tls.ca_file is empty")
	}
	if c.Nozzle.RLPAddr == "" {
		return fmt.Errorf("Configuration error: nozzle.rlp_addr is empty")
	}
	if c.Nozzle.ShardID == "" {
		return fmt.Errorf("Configuration error: nozzle.shard_id is empty")
	}
	if c.Nozzle.TLS.CertFile == "" {
		return fmt.Errorf("Configuration error: nozzle.tls.cert_file is empty")
	}
	if c.Nozzle.TLS.KeyFile == "" {
		return fmt.Errorf("Configuration error: nozzle.tls.key_file is empty")
	}
	if c.Nozzle.TLS.CACertFile == "" {
		return fmt.Errorf("Configuration error: nozzle.tls.ca_file is empty")
	}

	return nil
}
