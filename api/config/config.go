package config

import (
	"errors"
	"strings"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/startup"

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
	Logging      helpers.LoggingConfig
	BrokerServer helpers.ServerConfig
	Server       helpers.ServerConfig

	CFServer helpers.ServerConfig

	Db                       map[string]db.DatabaseConfig
	BrokerCredentials        []BrokerCredentialsConfig
	APIClientId              string
	PlanCheck                *PlanCheckConfig
	CatalogPath              string
	CatalogSchemaPath        string
	DashboardRedirectURI     string
	BindingRequestSchemaPath string
	Scheduler                SchedulerConfig
	ScalingEngine            ScalingEngineConfig
	EventGenerator           EventGeneratorConfig
	CF                       cf.Config
	InfoFilePath             string
	MetricsForwarder         MetricsForwarderConfig
	Health                   helpers.HealthConfig
	RateLimit                models.RateLimitConfig
	ScalingRules             ScalingRulesConfig

	CustomMetricsAuthConfig *CustomMetricsAuthConfig
}

var _ startup.ConfigValidator = Config{}

type CustomMetricsAuthConfig struct {
	// Configures if "Basic Authentication" is generally available or only available for already
	// bound apps.
	BasicAuthHandling BasicAuthHandling

	// Configures the authentication method that is used by default for sending custom metrics.
	DefaultCustomMetricAuthType models.CustomMetricsBindingAuthScheme

	// Configures how "Basic Authentication" is done.
	BasicAuthHandlingImplConfig models.BasicAuthHandlingImplConfig
}

type BasicAuthHandling int

const (
	BasicAuthHandlingOn BasicAuthHandling = iota
	BasicAuthHandlingOnlyExistingBindings

	// // The follwing gets encoded by having or not having a `CustomMetricsAuthConfig` at all.
	// BasicAuthHandlingOff
)

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
