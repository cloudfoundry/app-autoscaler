package cf

import (
	"fmt"
	"net/url"
	"strings"
)

type ClientConfig struct {
	MaxRetries              int   `yaml:"max_retries" json:"max_retries,omitempty"`
	MaxRetryWaitMs          int64 `yaml:"max_retry_wait_ms" json:"max_retry_wait_ms"`
	IdleConnectionTimeoutMs int64 `yaml:"idle_connection_timeout_ms" json:"idle_connection_timeout_ms"`
	MaxIdleConnsPerHost     int   `yaml:"max_idle_conns_per_host_ms" json:"max_idle_conns_per_host_ms"`
	SkipSSLValidation       bool  `yaml:"skip_ssl_validation" json:"skip_ssl_validation"`
}

type Config struct {
	ClientConfig `yaml:",inline"`
	API          string `yaml:"api" json:"api"`
	ClientID     string `yaml:"client_id" json:"client_id"`
	Secret       string `yaml:"secret" json:"secret"`
	GrantType    string `yaml:"grant_type" json:"grant_type"`
	Username     string `yaml:"username" json:"username"`
	Password     string `yaml:"password" json:"password"`
}

func (conf *Config) IsPasswordGrant() bool {
	return conf.GrantType == GrantTypePassword
}

func (conf *Config) Validate() error {
	if err := conf.validateAPI(); err != nil {
		return err
	}

	if conf.IsPasswordGrant() {
		return conf.validatePasswordGrant()
	}
	return conf.validateClientCredentials()
}

func (conf *Config) validateAPI() error {
	if conf.API == "" {
		return fmt.Errorf("Configuration error: cf api is empty")
	}

	apiURL, err := url.Parse(conf.API)
	if err != nil {
		return fmt.Errorf("Configuration error: cf api is not a valid url")
	}

	if apiURL.Scheme == "" {
		return fmt.Errorf("Configuration error: cf api scheme is empty")
	}

	scheme := strings.ToLower(apiURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("Configuration error: cf api scheme is invalid")
	}

	apiURL.Path = strings.TrimSuffix(apiURL.Path, "/")
	conf.API = apiURL.String()

	return nil
}

func (conf *Config) validatePasswordGrant() error {
	if conf.Username == "" {
		return fmt.Errorf("Configuration error: username is empty for password grant")
	}
	if conf.Password == "" {
		return fmt.Errorf("Configuration error: password is empty for password grant")
	}
	if conf.ClientID == "" {
		conf.ClientID = "cf"
	}
	return nil
}

func (conf *Config) validateClientCredentials() error {
	if conf.GrantType == "" {
		conf.GrantType = GrantTypeClientCredentials
	}
	if conf.ClientID == "" {
		return fmt.Errorf("Configuration error: client_id is empty")
	}
	return nil
}
