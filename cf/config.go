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
	PerPage      int    `yaml:"per_page" json:"per_page"`
}

func (conf *Config) Validate() error {
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

	if conf.ClientID == "" {
		return fmt.Errorf("Configuration error: client_id is empty")
	}

	return nil
}
