package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"

	"autoscaler/db"
	"autoscaler/helpers"
)

const (
	DefaultLoggingLevel = "info"
)

type ServerConfig struct {
	Port int `yaml:"port"`
}

var defaultServerConfig = ServerConfig{
	Port: 8080,
}

var defaultLoggingConfig = helpers.LoggingConfig{
	Level: DefaultLoggingLevel,
}

type DBConfig struct {
	BindingDB db.DatabaseConfig `yaml:"binding_db"`
	PolicyDB  db.DatabaseConfig `yaml:"policy_db"`
}

type Config struct {
	Logging              helpers.LoggingConfig `yaml:"logging"`
	Server               ServerConfig          `yaml:"server"`
	DB                   DBConfig              `yaml:"db"`
	BrokerUsername       string                `yaml:"broker_username"`
	BrokerPassword       string                `yaml:"broker_password"`
	CatalogPath          string                `yaml:"catalog_path"`
	CatalogSchemaPath    string                `yaml:"catalog_schema_path"`
	DashboardRedirectURI string                `yaml:"dashboard_redirect_uri"`
	PolicySchemaPath     string                `yaml:"policy_schema_path"`
}

func LoadConfig(reader io.Reader) (*Config, error) {
	conf := &Config{
		Logging: defaultLoggingConfig,
		Server:  defaultServerConfig,
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
	if c.DB.BindingDB.URL == "" {
		return fmt.Errorf("Configuration error: BindingDB URL is empty")
	}
	if c.DB.PolicyDB.URL == "" {
		return fmt.Errorf("Configuration error: PolicyDB URL is empty")
	}
	if c.BrokerUsername == "" {
		return fmt.Errorf("Configuration error: BrokerUsername is empty")
	}
	if c.BrokerPassword == "" {
		return fmt.Errorf("Configuration error: BrokerPassword is empty")
	}
	if c.CatalogSchemaPath == "" {
		return fmt.Errorf("Configuration error: CatalogSchemaPath is empty")
	}
	if c.CatalogPath == "" {
		return fmt.Errorf("Configuration error: CatalogPath is empty")
	}
	if c.PolicySchemaPath == "" {
		return fmt.Errorf("Configuration error: PolicySchemaPath is empty")
	}

	catalogSchemaLoader := gojsonschema.NewReferenceLoader("file://" + c.CatalogSchemaPath)
	catalogLoader := gojsonschema.NewReferenceLoader("file://" + c.CatalogPath)

	result, err := gojsonschema.Validate(catalogSchemaLoader, catalogLoader)
	if err != nil {
		return err
	}
	if !result.Valid() {
		errString := "{"
		for index, desc := range result.Errors() {
			if index == len(result.Errors())-1 {
				errString += fmt.Sprintf("\"%s\"", desc.Description())
			} else {
				errString += fmt.Sprintf("\"%s\",", desc.Description())
			}
		}
		errString += "}"
		return fmt.Errorf(errString)
	}

	return nil
}
