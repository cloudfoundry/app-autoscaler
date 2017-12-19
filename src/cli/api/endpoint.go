package api

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/cli/cf/configuration/confighelpers"
)

type APIEndpoint struct {
	URL               string
	SkipSSLValidation bool
}

var configFileName = func() string {

	defaultCFConfigPath, _ := confighelpers.DefaultFilePath()
	targetsPath := filepath.Join(filepath.Dir(defaultCFConfigPath), "plugins", "autoscaler_config")
	os.Mkdir(targetsPath, 0700)

	return filepath.Join(targetsPath, "config.json")
}

func GetEndpoint() (*APIEndpoint, error) {

	configFilePath := configFileName()
	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	endpoint := &APIEndpoint{}
	err = json.Unmarshal(content, &endpoint)
	if err != nil || endpoint.URL == "" {
		ioutil.WriteFile(configFilePath, nil, 0600)
	}
	return endpoint, nil

}

func UnsetEndpoint() error {

	configFilePath := configFileName()
	err := ioutil.WriteFile(configFilePath, nil, 0600)
	if err != nil {
		return err
	}
	return nil
}

func SetEndpoint(url string, skipSSLValidation bool) (*APIEndpoint, error) {

	endpoint := &APIEndpoint{
		URL:               url,
		SkipSSLValidation: skipSSLValidation,
	}

	urlConfig, err := json.Marshal(endpoint)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(configFileName(), urlConfig, 0600)
	if err != nil {
		return nil, err
	}

	return endpoint, nil
}
