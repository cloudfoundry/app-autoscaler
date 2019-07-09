package cf

import (
	"fmt"
	"net/url"
	"strings"
)

type CFConfig struct {
	API               string `yaml:"api"`
	ClientID          string `yaml:"client_id"`
	Secret            string `yaml:"secret"`
	SkipSSLValidation bool   `yaml:"skip_ssl_validation"`
}

func (conf *CFConfig) Validate() error {
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
	if (scheme != "http") && (scheme != "https") {
		return fmt.Errorf("Configuration error: cf api scheme is invalid")
	}

	if strings.HasSuffix(apiURL.Path, "/") {
		apiURL.Path = strings.TrimSuffix(apiURL.Path, "/")
	}
	conf.API = apiURL.String()

	if conf.ClientID == "" {
		return fmt.Errorf("Configuration error: client_id is empty")
	}

	return nil
}
