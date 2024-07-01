package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cloudfoundry-community/go-cfenv"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"github.com/kelseyhightower/envconfig"

	"gopkg.in/yaml.v3"
)

// There are 3 type of errors that this package can return:
// - ReadYamlError
// - ReadFileError
// - ReadEnvironmentError

var ReadYamlError = errors.New("failed to read config file")
var ReadFileError = errors.New("failed to open config file")
var ReadEnvironmentError = errors.New("failed to read environment variables")
var ReadVCAPEnvironmentError = errors.New("failed to read VCAP environment variables")

const (
	DefaultMetronAddress        = "127.0.0.1:3458"
	DefaultCacheTTL             = 15 * time.Minute
	DefaultCacheCleanupInterval = 6 * time.Hour
	DefaultPolicyPollerInterval = 40 * time.Second
	DefaultMaxAmount            = 10
	DefaultValidDuration        = 1 * time.Second
)

type Config struct {
	Logging               helpers.LoggingConfig         `yaml:"logging"`
	Server                helpers.ServerConfig          `yaml:"server"`
	LoggregatorConfig     LoggregatorConfig             `yaml:"loggregator"`
	SyslogConfig          SyslogConfig                  `yaml:"syslog"`
	Db                    map[string]db.DatabaseConfig  `yaml:"db"`
	PolicyDB              db.DatabaseConfig             `yaml:"policy_db"`
	CacheTTL              time.Duration                 `yaml:"cache_ttl"`
	CacheCleanupInterval  time.Duration                 `yaml:"cache_cleanup_interval"`
	PolicyPollerInterval  time.Duration                 `yaml:"policy_poller_interval"`
	Health                helpers.HealthConfig          `yaml:"health"`
	RateLimit             models.RateLimitConfig        `yaml:"rate_limit"`
	CredHelperImpl        string                        `yaml:"cred_helper_impl"`
	StoredProcedureConfig *models.StoredProcedureConfig `yaml:"stored_procedure_binding_credential_config"`
}

var defaultServerConfig = helpers.ServerConfig{
	Port: 6110,
}

var defaultHealthConfig = helpers.HealthConfig{
	ServerConfig: helpers.ServerConfig{
		Port: 8081,
	},
}

var defaultLoggingConfig = helpers.LoggingConfig{
	Level: "info",
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type LoggregatorConfig struct {
	MetronAddress string          `yaml:"metron_address"`
	TLS           models.TLSCerts `yaml:"tls"`
}

type SyslogConfig struct {
	ServerAddress string          `yaml:"server_address"`
	Port          int             `yaml:"port"`
	TLS           models.TLSCerts `yaml:"tls"`
}

func LoadConfig(filepath string) (*Config, error) {
	var conf Config
	var err error

	conf = Config{
		Server:  defaultServerConfig,
		Logging: defaultLoggingConfig,
		LoggregatorConfig: LoggregatorConfig{
			MetronAddress: DefaultMetronAddress,
		},
		Health:               defaultHealthConfig,
		CacheTTL:             DefaultCacheTTL,
		CacheCleanupInterval: DefaultCacheCleanupInterval,
		PolicyPollerInterval: DefaultPolicyPollerInterval,
		RateLimit: models.RateLimitConfig{
			MaxAmount:     DefaultMaxAmount,
			ValidDuration: DefaultValidDuration,
		},
	}

	if filepath == "" {
		fmt.Fprintln(os.Stdout, "missing config file, using environment variables")
	} else {
		r, err := os.Open(filepath)

		if err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "failed to open config file '%s' : %s\n", filepath, err.Error())
			return nil, err
		}

		dec := yaml.NewDecoder(r)
		dec.KnownFields(true)
		err = dec.Decode(&conf)

		if err != nil {
			return nil, fmt.Errorf("%w: %w", ReadYamlError, err)
		}

		defer r.Close()
	}

	err = envconfig.Process("", &conf)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ReadEnvironmentError, err)
	}

	err = loadVCAPEnvs(&conf)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ReadVCAPEnvironmentError, err)
	}

	return &conf, nil
}

func (c *Config) UsingSyslog() bool {
	return c.SyslogConfig.ServerAddress != "" && c.SyslogConfig.Port != 0
}

func (c *Config) Validate() error {
	if c.Db[db.PolicyDb].URL == "" {
		return fmt.Errorf("Configuration error: Policy DB url is empty")
	}
	if c.UsingSyslog() {
		if c.SyslogConfig.TLS.CACertFile == "" {
			return fmt.Errorf("Configuration error: SyslogServer Loggregator CACert is empty")
		}
		if c.SyslogConfig.TLS.CertFile == "" {
			return fmt.Errorf("Configuration error: SyslogServer ClientCert is empty")
		}
		if c.SyslogConfig.TLS.KeyFile == "" {
			return fmt.Errorf("Configuration error: SyslogServer ClientKey is empty")
		}
	} else {
		if c.LoggregatorConfig.TLS.CACertFile == "" {
			return fmt.Errorf("Configuration error: Loggregator CACert is empty")
		}
		if c.LoggregatorConfig.TLS.CertFile == "" {
			return fmt.Errorf("Configuration error: Loggregator ClientCert is empty")
		}
		if c.LoggregatorConfig.TLS.KeyFile == "" {
			return fmt.Errorf("Configuration error: Loggregator ClientKey is empty")
		}
	}

	if c.RateLimit.MaxAmount <= 0 {
		return fmt.Errorf("Configuration error: RateLimit.MaxAmount is equal or less than zero")
	}
	if c.RateLimit.ValidDuration <= 0*time.Nanosecond {
		return fmt.Errorf("Configuration error: RateLimit.ValidDuration is equal or less than zero nanosecond")
	}
	if c.CredHelperImpl == "" {
		return fmt.Errorf("Configuration error: CredHelperImpl is empty")
	}

	if err := c.Health.Validate(); err != nil {
		return err
	}

	return nil
}

func loadVCAPEnvs(c *Config) error {
	if os.Getenv("VCAP_APPLICATION") == "" || os.Getenv("VCAP_SERVICES") == "" {
		return nil
	}

	// panic here
	appEnv, err := cfenv.Current()
	if err != nil {
		return err
	}

	dbServices, err := appEnv.Services.WithTag("relational")
	if err != nil {
		return fmt.Errorf("failed to get db service with relational tag")
	}

	//	if len(dbServices) != 1 {
	//		return nil,
	//	}

	dbService := dbServices[0]

	dbURI, ok := dbService.CredentialString("uri")
	if !ok {
		return fmt.Errorf("failed to get uri from db service")
	}

	c.Db[db.PolicyDb] = db.DatabaseConfig{
		URL: dbURI,
	}

	//dbURL, err := url.Parse(dbURI)
	//if err != nil {
	//	return nil, err
	//}

	//parameters, err := url.ParseQuery(dbURL.RawQuery)
	//if err != nil {
	//	return nil, err
	//}

	//err = materializeConnectionParameter(dbService, parameters, "client_cert", "sslcert")
	//if err != nil {
	//	return nil, err
	//}

	//err = materializeConnectionParameter(dbService, parameters, "client_key", "sslkey")
	//if err != nil {
	//	return nil, err
	//}

	//err = materializeConnectionParameter(dbService, parameters, "server_ca", "sslrootcert")
	//if err != nil {
	//	return nil, err
	//}

	//dbURL.RawQuery = parameters.Encode()

	return nil
}
