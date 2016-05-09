package config

import (
	"testing"
	"time"

	cfhelpers "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

const JAVA_APP = "../assets/app/HelloWorldJavaWeb.war"
const NODE_APP = "../assets/app/nodeApp"

type Config struct {
	cfhelpers.Config
	ServiceName string `json:"service_name"`
	APIUrl      string `json:"api_url"`
}

var (
	DEFAULT_TIMEOUT      = 60 * time.Second
	CF_PUSH_TIMEOUT      = 2 * time.Minute
	LONG_CURL_TIMEOUT    = 2 * time.Minute
	CF_JAVA_TIMEOUT      = 10 * time.Minute
	DEFAULT_MEMORY_LIMIT = "700M"
)

func LoadConfig(t *testing.T) Config {
	var config Config
	err := cfhelpers.Load(cfhelpers.ConfigPath(), &config)
	if err != nil {
		t.Fatalf("Failed to load config, %s", err.Error())
	}

	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}

	if config.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = config.CfPushTimeout * time.Second
	}

	if config.LongCurlTimeout > 0 {
		LONG_CURL_TIMEOUT = config.LongCurlTimeout * time.Second
	}

	return config
}
