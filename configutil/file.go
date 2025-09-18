package configutil

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
)

func MaterializeContentInFile(folderName, fileName, content string) (string, error) {
	dirPath := fmt.Sprintf("/tmp/%s", folderName)

	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return "", err
	}

	filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return "", err
	}

	return filePath, nil
}

func MaterializeContentInTmpFile(folderName, fileName, content string) (string, error) {
	dirPath, err := os.MkdirTemp("", folderName)
	if err != nil {
		return "", err
	}

	filePath := fmt.Sprintf("%s/%s", dirPath, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return "", err
	}

	return filePath, nil
}

type Configurable interface {
	SetLoggingLevel()
}

type VCAPConfigurable[T any] interface {
	LoadVCAPConfig(conf *T, vcapReader VCAPConfigurationReader) error
}

// VCAPConfigurableFunc allows using a function as VCAPConfigurable
type VCAPConfigurableFunc[T any] func(*T, VCAPConfigurationReader) error

func (f VCAPConfigurableFunc[T]) LoadVCAPConfig(conf *T, vcapReader VCAPConfigurationReader) error {
	return f(conf, vcapReader)
}

func GenericLoadConfig[T any](
	filepath string,
	vcapReader VCAPConfigurationReader,
	defaultConfig func() T,
	vcapLoader VCAPConfigurable[T],
) (*T, error) {
	conf := defaultConfig()

	if err := helpers.LoadYamlFile(filepath, &conf); err != nil {
		return nil, err
	}

	if err := vcapLoader.LoadVCAPConfig(&conf, vcapReader); err != nil {
		return nil, err
	}

	if configurable, ok := any(&conf).(Configurable); ok {
		configurable.SetLoggingLevel()
	}

	return &conf, nil
}
