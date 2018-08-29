package cf

import (
	"fmt"
	"net/url"
	"strings"
)

type CFConfig struct {
	API               string `yaml:"api"`
	GrantType         string `yaml:"grant_type"`
	Username          string `yaml:"username"`
	Password          string `yaml:"password"`
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

	if conf.GrantType != GrantTypePassword && conf.GrantType != GrantTypeClientCredentials {
		return fmt.Errorf("Configuration error: unsupported grant_type [%s]", conf.GrantType)
	}

	if conf.GrantType == GrantTypePassword {
		if conf.Username == "" {
			return fmt.Errorf("Configuration error: username is empty")
		}
	}

	if conf.GrantType == GrantTypeClientCredentials {
		if conf.ClientID == "" {
			return fmt.Errorf("Configuration error: client_id is empty")
		}
	}
	return nil
}
