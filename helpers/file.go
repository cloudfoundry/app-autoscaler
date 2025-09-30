package helpers

import (
	"errors"
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

var ErrReadYaml = errors.New("failed to read config file")

func LoadYamlFile[T any](filepath string, conf *T) error {
	if filepath == "" {
		return nil
	}
	file, err := os.Open(filepath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to open config file '%s': %s\n", filepath, err)
		return ErrReadYaml
	}
	defer func() { _ = file.Close() }()

	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	if err := dec.Decode(conf); err != nil {
		return fmt.Errorf("%w: %v", ErrReadYaml, err)
	}
	return nil
}
