package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

var (
	ErrReadYaml               = errors.New("failed to read config file")
	ErrReadJson               = errors.New("failed to read vcap_services json")
	ErrOperatorConfigNotFound = errors.New("operator config service not found")
)

const (
	DefaultLoggingLevel        string = "info"
	DefaultRefreshInterval            = 24 * time.Hour
	DefaultCutoffDuration             = 30 * 24 * time.Hour
	DefaultSyncInterval               = 24 * time.Hour
	DefaultDBLockRetryInterval        = 5 * time.Second
	DefaultDBLockTTL                  = 15 * time.Second
	DefaultHttpClientTimeout          = 5 * time.Second
)

type DbPrunerConfig struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	CutoffDuration  time.Duration `yaml:"cutoff_duration"`
}

type DBLockConfig struct {
	LockTTL           time.Duration `yaml:"ttl"`
	LockRetryInterval time.Duration `yaml:"retry_interval"`
}

var defaultDBLockConfig = DBLockConfig{
	LockTTL:           DefaultDBLockTTL,
	LockRetryInterval: DefaultDBLockRetryInterval,
}

type ScalingEngineConfig struct {
	URL            string          `yaml:"scaling_engine_url"`
	SyncInterval   time.Duration   `yaml:"sync_interval"`
	TLSClientCerts models.TLSCerts `yaml:"tls"`
}

type SchedulerConfig struct {
	URL            string          `yaml:"scheduler_url"`
	SyncInterval   time.Duration   `yaml:"sync_interval"`
	TLSClientCerts models.TLSCerts `yaml:"tls"`
}

type AppSyncerConfig struct {
	SyncInterval time.Duration `yaml:"sync_interval"`
}

var defaultHealthConfig = helpers.HealthConfig{
	ServerConfig: helpers.ServerConfig{
		Port: 8081,
	},
}

type Config struct {
	CF                cf.Config                    `yaml:"cf"`
	Db                map[string]db.DatabaseConfig `yaml:"db" json:"db"`
	Health            helpers.HealthConfig         `yaml:"health"`
	Logging           helpers.LoggingConfig        `yaml:"logging"`
	AppMetricsDb      DbPrunerConfig               `yaml:"app_metrics_db"`
	ScalingEngineDb   DbPrunerConfig               `yaml:"scaling_engine_db"`
	ScalingEngine     ScalingEngineConfig          `yaml:"scaling_engine"`
	Scheduler         SchedulerConfig              `yaml:"scheduler"`
	AppSyncer         AppSyncerConfig              `yaml:"app_syncer"`
	DBLock            DBLockConfig                 `yaml:"db_lock"`
	HttpClientTimeout time.Duration                `yaml:"http_client_timeout"`
}

func defaultConfig() Config {
	return Config{
		CF: cf.Config{
			ClientConfig: cf.ClientConfig{SkipSSLValidation: false},
		},
		Health:  defaultHealthConfig,
		Logging: helpers.LoggingConfig{Level: DefaultLoggingLevel},
		AppMetricsDb: DbPrunerConfig{
			RefreshInterval: DefaultRefreshInterval,
			CutoffDuration:  DefaultCutoffDuration,
		},
		Db: make(map[string]db.DatabaseConfig),
		ScalingEngineDb: DbPrunerConfig{
			RefreshInterval: DefaultRefreshInterval,
			CutoffDuration:  DefaultCutoffDuration,
		},
		ScalingEngine: ScalingEngineConfig{
			SyncInterval: DefaultSyncInterval,
		},
		Scheduler: SchedulerConfig{
			SyncInterval: DefaultSyncInterval,
		},
		AppSyncer: AppSyncerConfig{
			SyncInterval: DefaultSyncInterval,
		},
		DBLock:            defaultDBLockConfig,
		HttpClientTimeout: DefaultHttpClientTimeout,
	}
}

func LoadConfig(filepath string, vcapReader configutil.VCAPConfigurationReader) (*Config, error) {
	conf := defaultConfig()
	if err := loadYamlFile(filepath, &conf); err != nil {
		return nil, err
	}

	if err := loadVcapConfig(&conf, vcapReader); err != nil {
		return nil, err
	}

	return &conf, nil
}

func loadVcapConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	if !vcapReader.IsRunningOnCF() {
		return nil
	}

	tlsCerts := vcapReader.GetInstanceTLSCerts()

	// enable plain text logging. See src/autoscaler/helpers/logger.go
	conf.Logging.PlainTextSink = true

	conf.Health.ServerConfig.Port = vcapReader.GetPort()
	if err := loadOperatorConfig(conf, vcapReader); err != nil {
		return err
	}

	if err := vcapReader.ConfigureDatabases(&conf.Db, nil, ""); err != nil {
		return err
	}

	conf.Scheduler.TLSClientCerts = tlsCerts
	conf.ScalingEngine.TLSClientCerts = tlsCerts

	return nil
}

func loadOperatorConfig(conf *Config, vcapReader configutil.VCAPConfigurationReader) error {
	var raw string
	data, err := vcapReader.GetServiceCredentialContent("operator-config", "operator-config")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOperatorConfigNotFound, err)
	}
	// removes the first and last double quotes if they exist
	if json.Unmarshal(data, &raw) == nil {
		return yaml.Unmarshal([]byte(raw), conf)
	} else {
		return yaml.Unmarshal(data, conf)
	}
}

func loadYamlFile(filepath string, conf *Config) error {
	if filepath == "" {
		return nil
	}
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to open config file '%s': %s\n", filepath, err)
		return ErrReadYaml
	}
	defer file.Close()

	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	if err := dec.Decode(conf); err != nil {
		return fmt.Errorf("%w: %v", ErrReadYaml, err)
	}

	return nil
}

func (c *Config) validateDb() error {
	if c.Db[db.AppMetricsDb].URL == "" {
		return fmt.Errorf("Configuration error: app_metrics_db.db.url is empty")
	}

	if c.Db[db.ScalingEngineDb].URL == "" {
		return fmt.Errorf("Configuration error: scaling_engine_db.db.url is empty")
	}

	if c.Db[db.PolicyDb].URL == "" {
		return fmt.Errorf("Configuration error: app_syncer.db.url is empty")
	}

	if c.Db[db.LockDb].URL == "" {
		return fmt.Errorf("Configuration error: db_lock.db.url is empty")
	}

	return nil
}

func (c *Config) Validate() error {
	if err := c.validateDb(); err != nil {
		return err
	}

	if c.AppMetricsDb.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: app_metrics_db.refresh_interval is less than or equal to 0")
	}

	if c.AppMetricsDb.CutoffDuration <= 0 {
		return fmt.Errorf("Configuration error: app_metrics_db.cutoff_duration is less than or equal to 0")
	}

	if c.ScalingEngineDb.RefreshInterval <= 0 {
		return fmt.Errorf("Configuration error: scaling_engine_db.refresh_interval is less than or equal to 0")
	}
	if c.ScalingEngineDb.CutoffDuration <= 0 {
		return fmt.Errorf("Configuration error: scaling_engine_db.cutoff_duration is less than or equal to 0")
	}

	if c.ScalingEngine.URL == "" {
		return fmt.Errorf("Configuration error: scaling_engine.scaling_engine_url is empty")
	}
	if c.ScalingEngine.SyncInterval <= 0 {
		return fmt.Errorf("Configuration error: scaling_engine.sync_interval is less than or equal to 0")
	}
	if c.Scheduler.URL == "" {
		return fmt.Errorf("Configuration error: scheduler.scheduler_url is empty")
	}
	if c.Scheduler.SyncInterval <= 0 {
		return fmt.Errorf("Configuration error: scheduler.sync_interval is less than or equal to 0")
	}

	if c.AppSyncer.SyncInterval <= 0 {
		return fmt.Errorf("Configuration error: appSyncer.sync_interval is less than or equal to 0")
	}

	if c.HttpClientTimeout <= time.Duration(0) {
		return fmt.Errorf("Configuration error: http_client_timeout is less-equal than 0")
	}

	if err := c.Health.Validate(); err != nil {
		return err
	}

	return nil
}
