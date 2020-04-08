package models

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type HealthConfig struct {
	Port                    int    `yaml:"port"`
	HealthCheckUsername     string `yaml:"username"`
	HealthCheckUsernameHash string `yaml:"username_hash"`
	HealthCheckPassword     string `yaml:"password"`
	HealthCheckPasswordHash string `yaml:"password_hash"`
}

func (c *HealthConfig) Validate(component string) error {

	if c.HealthCheckUsername != "" && c.HealthCheckUsernameHash != "" {
		return fmt.Errorf("Configuration error: both %s healthcheck username and %s healthcheck username_hash are set, please provide only one of them", component, component)
	}

	if c.HealthCheckPassword != "" && c.HealthCheckPasswordHash != "" {
		return fmt.Errorf("Configuration error: both %s healthcheck password and %s healthcheck password_hash are empty, please provide one of them", component, component)
	}

	if c.HealthCheckUsernameHash != "" {
		if _, err := bcrypt.Cost([]byte(c.HealthCheckUsernameHash)); err != nil {
			return fmt.Errorf("Configuration error: %s healthcheck username_hash is not a valid bcrypt hash", component)
		}
	}

	if c.HealthCheckPasswordHash != "" {
		if _, err := bcrypt.Cost([]byte(c.HealthCheckPasswordHash)); err != nil {
			return fmt.Errorf("Configuration error: %s healthcheck password_hash is not a valid bcrypt hash", component)
		}
	}

	if c.HealthCheckUsername == "" && c.HealthCheckPassword != "" {
		return fmt.Errorf("Configuration error: %s healthcheck username is empty", component)
	}

	if c.HealthCheckUsername != "" && c.HealthCheckPassword == "" {
		return fmt.Errorf("Configuration error: %s healthcheck password is empty", component)
	}

	return nil
}
