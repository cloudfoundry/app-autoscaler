package config

import (
	"fmt"
	"github.com/cloudfoundry-incubator/candiedyaml"
	"io/ioutil"
)

type CfConfig struct {
	Api       string `yaml:"api"`
	GrantType string `yaml:"grant_type"`
	User      string `yaml:"user"`
	Pass      string `yaml:"pass`
	ClientId  string `yaml:"client_id"`
	Secret    string `yaml:"secret"`
}

var defaultCfConfig = CfConfig{
	Api:       "https://api.bosh-lite.com",
	GrantType: "password",
	User:      "admin",
	Pass:      "admin",
	ClientId:  "admin",
	Secret:    "admin-secret",
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

var defaultServerConfig = ServerConfig{
	Port: 8080,
	User: "",
	Pass: "",
}

type LoggingConfig struct {
	Level       string `yaml:"level"`
	File        string `yaml:"file"`
	LogToStdout bool   `yaml:"log_to_stdout"`
}

var defaultLoggingConfig = LoggingConfig{
	Level:       "info",
	File:        "",
	LogToStdout: true,
}

type Config struct {
	Cf      CfConfig      `yaml:"cf"`
	Logging LoggingConfig `yaml:"logging"`
	Server  ServerConfig  `yaml:"server"`
}

func DefaultConfig() *Config {
	var c = Config{
		Cf:      defaultCfConfig,
		Logging: defaultLoggingConfig,
		Server:  defaultServerConfig,
	}

	return &c
}

func LoadConfigFromFile(path string) (c *Config, e error) {
	b, e := ioutil.ReadFile(path)
	if e == nil {
		return LoadConfigFromYaml(b)
	}
	return nil, e
}

func LoadConfigFromYaml(b []byte) (c *Config, e error) {
	c = DefaultConfig()
	e = candiedyaml.Unmarshal(b, c)
	return
}

func (c *Config) ToString() (s string, e error) {
	b, e := candiedyaml.Marshal(c)
	if e == nil {
		s = fmt.Sprintf("%s", b)
	}
	return
}
