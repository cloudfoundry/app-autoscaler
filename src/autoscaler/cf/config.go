package cf

import (
	"fmt"
	"net/url"
	"strings"
)

type CfConfig struct {
	Api       string `yaml:"api"`
	GrantType string `yaml:"grant_type"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	ClientId  string `yaml:"client_id"`
	Secret    string `yaml:"secret"`
	SkipSSLValidation bool `yaml:"skip_ssl_validation"`
}

func (conf *CfConfig) Validate() error {
	if conf.Api == "" {
		return fmt.Errorf("Configuration error: cf api is empty")
	}

	apiUrl, err := url.Parse(conf.Api)
	if err != nil {
		return fmt.Errorf("Configuration error: cf api is not a valid url")
	}

	if apiUrl.Scheme == "" {
		return fmt.Errorf("Configuration error: cf api scheme is empty")
	}

	scheme := strings.ToLower(apiUrl.Scheme)
	if (scheme != "http") && (scheme != "https") {
		return fmt.Errorf("Configuration error: cf api scheme is invalid")
	}

	if strings.HasSuffix(apiUrl.Path, "/") {
		apiUrl.Path = strings.TrimSuffix(apiUrl.Path, "/")
	}
	conf.Api = apiUrl.String()

	if conf.GrantType != GrantTypePassword && conf.GrantType != GrantTypeClientCredentials {
		return fmt.Errorf("Configuration error: unsupported grant type [%s]", conf.GrantType)
	}

	if conf.GrantType == GrantTypePassword {
		if conf.Username == "" {
			return fmt.Errorf("Configuration error: user name is empty")
		}
	}

	if conf.GrantType == GrantTypeClientCredentials {
		if conf.ClientId == "" {
			return fmt.Errorf("Configuration error: client id is empty")
		}
	}
	return nil
}
