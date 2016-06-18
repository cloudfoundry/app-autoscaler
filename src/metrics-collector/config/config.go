package config

import (
	"fmt"
	"github.com/cloudfoundry-incubator/candiedyaml"
	"io/ioutil"
	"strings"
)

const GRANT_TYPE_PASSWORD = "password"
const GRANT_TYPE_CLIENT_CREDENTIALS = "client_credentials"
const DEFAULT_LOGGING_LEVEL = "info"

type CfConfig struct {
	Api       string `yaml:"api"`
	GrantType string `yaml:"grant_type"`
	User      string `yaml:"user"`
	Pass      string `yaml:"pass`
	ClientId  string `yaml:"client_id"`
	Secret    string `yaml:"secret"`
}

var DefaultCfConfig = CfConfig{
	GrantType: GRANT_TYPE_PASSWORD,
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

var DefaultServerConfig = ServerConfig{
	Port: 8080,
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

var DefaultLoggingConfig = LoggingConfig{
	Level: DEFAULT_LOGGING_LEVEL,
}

type Config struct {
	Cf      CfConfig      `yaml:"cf"`
	Logging LoggingConfig `yaml:"logging"`
	Server  ServerConfig  `yaml:"server"`
}

func LoadConfigFromFile(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadConfigFromYaml(bytes)
}

func LoadConfigFromYaml(bytes []byte) (*Config, error) {
	conf := &Config{
		Cf:      DefaultCfConfig,
		Logging: DefaultLoggingConfig,
		Server:  DefaultServerConfig,
	}

	err := candiedyaml.Unmarshal(bytes, conf)
	if err != nil {
		return nil, err
	}

	conf.Cf.GrantType = strings.ToLower(conf.Cf.GrantType)
	conf.Logging.Level = strings.ToLower(conf.Logging.Level)

	return conf, nil
}

func (c *Config) Verify() error {
	if c.Cf.GrantType != GRANT_TYPE_PASSWORD && c.Cf.GrantType != GRANT_TYPE_CLIENT_CREDENTIALS {
		return fmt.Errorf("Error in configuration file: unsupported grant type [%s]", c.Cf.GrantType)
	}

	if c.Cf.GrantType == GRANT_TYPE_PASSWORD {
		if c.Cf.User == "" {
			return fmt.Errorf("Error in configuration file: user name is empty")
		}
	}

	if c.Cf.GrantType == GRANT_TYPE_CLIENT_CREDENTIALS {
		if c.Cf.ClientId == "" {
			return fmt.Errorf("Error in configuration file: client id is empty")
		}
	}
	return nil

}

func (c *Config) ToString() (string, error) {
	bytes, err := candiedyaml.Marshal(c)

	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", bytes), nil
}
