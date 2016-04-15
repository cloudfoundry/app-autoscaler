package config

import (
	"testing"
	"time"

	cfhelpers "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/services"

	"os"
)

const JAVA_APP = "../assets/app/HelloWorldJavaWeb.war"
const NODE_APP = "../assets/app/nodeApp"

type Config struct {
	services.Config
	ServiceName string `json:"service_name"`
	APIUrl      string `json:"api_url"`
}

var (
	DEFAULT_TIMEOUT      = 30 * time.Second
	CF_PUSH_TIMEOUT      = 2 * time.Minute
	LONG_CURL_TIMEOUT    = 2 * time.Minute
	CF_JAVA_TIMEOUT      = 10 * time.Minute
	DEFAULT_MEMORY_LIMIT = "700M"
)

func LoadConfig(t *testing.T) (Config, cfhelpers.Config) {
	configPath := os.Getenv("CONFIG")
	if configPath == "" {
		t.Fatalf("Must set $CONFIG to point to a config .json file")
	}

	var config Config
	err := services.LoadConfig(configPath, &config)
	if err != nil {
		t.Fatalf("Failed to load config, %s", err.Error())
	}
	err = services.ValidateConfig(&config.Config)
	if err != nil {
		t.Fatalf("Invalid config, %s", err.Error())
	}

	defaultConfig := cfhelpers.LoadConfig()

	if defaultConfig.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = defaultConfig.DefaultTimeout * time.Second
	}

	if defaultConfig.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = defaultConfig.CfPushTimeout * time.Second
	}

	if defaultConfig.LongCurlTimeout > 0 {
		LONG_CURL_TIMEOUT = defaultConfig.LongCurlTimeout * time.Second
	}

	return config, defaultConfig
}
