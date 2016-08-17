package cf

import (
	"fmt"
)

type CfConfig struct {
	Api       string `yaml:"api"`
	GrantType string `yaml:"grant_type"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	ClientId  string `yaml:"client_id"`
	Secret    string `yaml:"secret"`
}

func (conf *CfConfig) Validate() error {
	if conf.Api == "" {
		return fmt.Errorf("Configuration error: cf api is empty")
	}

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
