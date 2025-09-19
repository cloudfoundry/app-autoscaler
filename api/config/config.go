package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"

	"golang.org/x/crypto/bcrypt"

	"github.com/xeipuuv/gojsonschema"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	DefaultLoggingLevel           = "info"
	DefaultMaxAmount              = 10
	DefaultValidDuration          = 1 * time.Second
	DefaultCPULowerThreshold      = 1
	DefaultCPUUpperThreshold      = 100
	DefaultCPUUtilLowerThreshold  = 1
	DefaultCPUUtilUpperThreshold  = 100
	DefaultDiskUtilLowerThreshold = 1
	DefaultDiskUtilUpperThreshold = 100
	DefaultDiskLowerThreshold     = 1
	DefaultDiskUpperThreshold     = 2 * 1024
)

var (
	ErrPublicApiServerConfigNotFound = errors.New("publicapiserver config service not found")
)

var defaultBrokerServerConfig = helpers.ServerConfig{
	Port: 8080,
}

var defaultPublicApiServerConfig = helpers.ServerConfig{
	Port: 8081,
}

var defaultLoggingConfig = helpers.LoggingConfig{
	Level: DefaultLoggingLevel,
}

type SchedulerConfig struct {
	SchedulerURL   string          `yaml:"scheduler_url" json:"scheduler_url"`
	TLSClientCerts models.TLSCerts `yaml:"tls" json:"tls"`
}

type ScalingEngineConfig struct {
	ScalingEngineUrl string          `yaml:"scaling_engine_url" json:"scaling_engine_url"`
	TLSClientCerts   models.TLSCerts `yaml:"tls" json:"tls"`
}

type EventGeneratorConfig struct {
	EventGeneratorUrl string          `yaml:"event_generator_url" json:"event_generator_url"`
	TLSClientCerts    models.TLSCerts `yaml:"tls" json:"tls"`
}
type MetricsForwarderConfig struct {
	MetricsForwarderUrl     string `yaml:"metrics_forwarder_url" json:"metrics_forwarder_url"`
	MetricsForwarderMtlsUrl string `yaml:"metrics_forwarder_mtls_url" json:"metrics_forwarder_mtls_url"`
}

type PlanDefinition struct {
	PlanCheckEnabled  bool `yaml:"planCheckEnabled" json:"planCheckEnabled"`
	SchedulesCount    int  `yaml:"schedules_count" json:"schedules_count"`
	ScalingRulesCount int  `yaml:"scaling_rules_count" json:"scaling_rules_count"`
	PlanUpdateable    bool `yaml:"plan_updateable" json:"plan_updateable"`
}

type BrokerCredentialsConfig struct {
	BrokerUsername     string `yaml:"broker_username" json:"broker_username"`
	BrokerUsernameHash []byte `yaml:"broker_username_hash" json:"broker_username_hash"`
	BrokerPassword     string `yaml:"broker_password" json:"broker_password"`
	BrokerPasswordHash []byte `yaml:"broker_password_hash" json:"broker_password_hash"`
}

type ScalingRulesConfig struct {
	CPU      LowerUpperThresholdConfig `yaml:"cpu" json:"cpu"`
	CPUUtil  LowerUpperThresholdConfig `yaml:"cpuutil" json:"cpuutil"`
	DiskUtil LowerUpperThresholdConfig `yaml:"diskutil" json:"diskutil"`
	Disk     LowerUpperThresholdConfig `yaml:"disk" json:"disk"`
}

type LowerUpperThresholdConfig struct {
	LowerThreshold int `yaml:"lower_threshold" json:"lower_threshold"`
	UpperThreshold int `yaml:"upper_threshold" json:"upper_threshold"`
}

type Config struct {
	Logging      helpers.LoggingConfig `yaml:"logging" json:"logging"`
	BrokerServer helpers.ServerConfig  `yaml:"broker_server" json:"broker_server"`
	Server       helpers.ServerConfig  `yaml:"public_api_server" json:"public_api_server"`

	CFServer helpers.ServerConfig `yaml:"cf_server" json:"cf_server"`

	Db                                 map[string]db.DatabaseConfig  `yaml:"db" json:"db,omitempty"`
	BrokerCredentials                  []BrokerCredentialsConfig     `yaml:"broker_credentials" json:"broker_credentials"`
	APIClientId                        string                        `yaml:"api_client_id" json:"api_client_id"`
	PlanCheck                          *PlanCheckConfig              `yaml:"plan_check" json:"plan_check"`
	CatalogPath                        string                        `yaml:"catalog_path" json:"catalog_path"`
	CatalogSchemaPath                  string                        `yaml:"catalog_schema_path" json:"catalog_schema_path"`
	DashboardRedirectURI               string                        `yaml:"dashboard_redirect_uri" json:"dashboard_redirect_uri"`
	PolicySchemaPath                   string                        `yaml:"policy_schema_path" json:"policy_schema_path"`
	Scheduler                          SchedulerConfig               `yaml:"scheduler" json:"scheduler"`
	ScalingEngine                      ScalingEngineConfig           `yaml:"scaling_engine" json:"scaling_engine"`
	EventGenerator                     EventGeneratorConfig          `yaml:"event_generator" json:"event_generator"`
	CF                                 cf.Config                     `yaml:"cf" json:"cf"`
	InfoFilePath                       string                        `yaml:"info_file_path" json:"info_file_path"`
	MetricsForwarder                   MetricsForwarderConfig        `yaml:"metrics_forwarder" json:"metrics_forwarder"`
	Health                             helpers.HealthConfig          `yaml:"health" json:"health"`
	RateLimit                          models.RateLimitConfig        `yaml:"rate_limit" json:"rate_limit,omitempty"`
	CredHelperImpl                     string                        `yaml:"cred_helper_impl" json:"cred_helper_impl"`
	StoredProcedureConfig              *models.StoredProcedureConfig `yaml:"stored_procedure_binding_credential_config" json:"stored_procedure_binding_credential_config"`
	ScalingRules                       ScalingRulesConfig            `yaml:"scaling_rules" json:"scaling_rules"`
	DefaultCustomMetricsCredentialType string                        `yaml:"default_credential_type" json:"default_credential_type"`
}

func (c *Config) SetLoggingLevel() {
	c.Logging.Level = strings.ToLower(c.Logging.Level)
}

// GetLogging returns the logging configuration
func (c *Config) GetLogging() *helpers.LoggingConfig {
	return &c.Logging
}

type PlanCheckConfig struct {
	PlanDefinitions map[string]PlanDefinition `yaml:"plan_definitions" json:"plan_definitions"`
}

func defaultConfig() Config {
	return Config{
		Logging:      defaultLoggingConfig,
		BrokerServer: defaultBrokerServerConfig,
		Server:       defaultPublicApiServerConfig,
		CF: cf.Config{
			ClientConfig: cf.ClientConfig{
				SkipSSLValidation: false,
			},
		},
		Db: make(map[string]db.DatabaseConfig),
		RateLimit: models.RateLimitConfig{
			MaxAmount:     DefaultMaxAmount,
			ValidDuration: DefaultValidDuration,
		},
		ScalingRules: ScalingRulesConfig{
			CPU: LowerUpperThresholdConfig{
				LowerThreshold: DefaultCPULowerThreshold,
				UpperThreshold: DefaultCPUUpperThreshold,
			},
			CPUUtil: LowerUpperThresholdConfig{
				LowerThreshold: DefaultCPUUtilLowerThreshold,
				UpperThreshold: DefaultCPUUtilUpperThreshold,
			},
			DiskUtil: LowerUpperThresholdConfig{
				LowerThreshold: DefaultDiskUtilLowerThreshold,
				UpperThreshold: DefaultDiskUtilUpperThreshold,
			},
			Disk: LowerUpperThresholdConfig{
				LowerThreshold: DefaultDiskLowerThreshold,
				UpperThreshold: DefaultDiskUpperThreshold,
			},
		},
	}
}

func LoadVcapConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	if !vcapReader.IsRunningOnCF() {
		return nil
	}

	tlsCert := vcapReader.GetInstanceTLSCerts()

	// enable plain text logging. See src/autoscaler/helpers/logger.go
	conf.Logging.PlainTextSink = true

	// Avoid port conflict: assign actual port to CF server, set BOSH server port to 0 (unused)
	conf.CFServer.Port = vcapReader.GetPort()
	conf.Server.Port = 0

	if err := configutil.LoadConfig(&conf, vcapReader, "apiserver-config"); err != nil {
		return err
	}

	if err := vcapReader.ConfigureDatabases(&conf.Db, conf.StoredProcedureConfig, conf.CredHelperImpl); err != nil {
		return err
	}

	if err := configureCatalog(conf, vcapReader); err != nil {
		return err
	}

	conf.ScalingEngine.TLSClientCerts = tlsCert
	conf.EventGenerator.TLSClientCerts = tlsCert
	conf.Scheduler.TLSClientCerts = tlsCert

	return nil
}

func configureCatalog(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	catalog, err := vcapReader.GetServiceCredentialContent("broker-catalog", "broker-catalog")
	if err != nil {
		return err
	}

	catalogPath, err := configutil.MaterializeContentInTmpFile("publicapi", "catalog.json", string(catalog))
	if err != nil {
		return err
	}

	conf.CatalogPath = catalogPath

	return err
}

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	return configutil.GenericLoadConfig(filepath, vcapReader, defaultConfig, configutil.VCAPConfigurableFunc[Config](LoadVcapConfig))
}

func FromJSON(data []byte) (*Config, error) {
	result := &Config{}
	err := json.Unmarshal(data, result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config from json: %s", err)
	}
	return result, nil
}

func (c *Config) Validate() error {
	err := c.CF.Validate()
	if err != nil {
		return err
	}

	if c.Db[db.PolicyDb].URL == "" {
		return fmt.Errorf("Configuration error: PolicyDB URL is empty")
	}
	if c.Scheduler.SchedulerURL == "" {
		return fmt.Errorf("Configuration error: scheduler.scheduler_url is empty")
	}
	if c.ScalingEngine.ScalingEngineUrl == "" {
		return fmt.Errorf("Configuration error: scaling_engine.scaling_engine_url is empty")
	}
	if c.EventGenerator.EventGeneratorUrl == "" {
		return fmt.Errorf("Configuration error: event_generator.event_generator_url is empty")
	}
	if c.MetricsForwarder.MetricsForwarderUrl == "" {
		return fmt.Errorf("Configuration error: metrics_forwarder.metrics_forwarder_url is empty")
	}
	if c.PolicySchemaPath == "" {
		return fmt.Errorf("Configuration error: PolicySchemaPath is empty")
	}
	if c.RateLimit.MaxAmount <= 0 {
		return fmt.Errorf("Configuration error: RateLimit.MaxAmount is equal or less than zero")
	}
	if c.RateLimit.ValidDuration <= 0*time.Nanosecond {
		return fmt.Errorf("Configuration error: RateLimit.ValidDuration is equal or less than zero nanosecond")
	}
	if err := c.Health.Validate(); err != nil {
		return err
	}

	if c.InfoFilePath == "" {
		return fmt.Errorf("Configuration error: InfoFilePath is empty")
	}

	if c.ScalingRules.CPU.LowerThreshold < 0 {
		return fmt.Errorf("Configuration error: ScalingRules.CPU.LowerThreshold is less than zero")
	}

	if c.ScalingRules.CPU.UpperThreshold < 0 {
		return fmt.Errorf("Configuration error: ScalingRules.CPU.UpperThreshold is less than zero")
	}

	if c.Db[db.BindingDb].URL == "" {
		return fmt.Errorf("Configuration error: BindingDB URL is empty")
	}

	for _, brokerCredential := range c.BrokerCredentials {
		if brokerCredential.BrokerUsername == "" && string(brokerCredential.BrokerUsernameHash) == "" {
			return fmt.Errorf("Configuration error: both broker_username and broker_username_hash are empty, please provide one of them")
		}
		if brokerCredential.BrokerUsername != "" && string(brokerCredential.BrokerUsernameHash) != "" {
			return fmt.Errorf("Configuration error: both broker_username and broker_username_hash are set, please provide only one of them")
		}
		if string(brokerCredential.BrokerUsernameHash) != "" {
			if _, err := bcrypt.Cost(brokerCredential.BrokerUsernameHash); err != nil {
				return fmt.Errorf("Configuration error: broker_username_hash is not a valid bcrypt hash")
			}
		}
		if brokerCredential.BrokerPassword == "" && string(brokerCredential.BrokerPasswordHash) == "" {
			return fmt.Errorf("Configuration error: both broker_password and broker_password_hash are empty, please provide one of them")
		}

		if brokerCredential.BrokerPassword != "" && string(brokerCredential.BrokerPasswordHash) != "" {
			return fmt.Errorf("Configuration error: both broker_password and broker_password_hash are set, please provide only one of them")
		}

		if string(brokerCredential.BrokerPasswordHash) != "" {
			if _, err := bcrypt.Cost(brokerCredential.BrokerPasswordHash); err != nil {
				return fmt.Errorf("Configuration error: broker_password_hash is not a valid bcrypt hash")
			}
		}
	}

	if c.CatalogSchemaPath == "" {
		return fmt.Errorf("Configuration error: CatalogSchemaPath is empty")
	}
	if c.CatalogPath == "" {
		return fmt.Errorf("Configuration error: CatalogPath is empty")
	}
	if c.CredHelperImpl == "" {
		return fmt.Errorf("Configuration error: CredHelperImpl is empty")
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
		return errors.New(errString)
	}

	return nil
}
