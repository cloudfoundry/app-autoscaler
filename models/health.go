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
	ReadinessCheckEnabled   bool   `yaml:"readiness_enabled"`
}

var ConfigurationErr = fmt.Errorf("configuration error")

func (c *HealthConfig) Validate() error {
	if c.HealthCheckUsername != "" && c.HealthCheckUsernameHash != "" {
		return fmt.Errorf("%w: both healthcheck username and healthcheck username_hash are set, please provide only one of them", ConfigurationErr)
	}

	if c.HealthCheckPassword != "" && c.HealthCheckPasswordHash != "" {
		return fmt.Errorf("%w: both healthcheck password and healthcheck password_hash are provided, please provide only one of them", ConfigurationErr)
	}

	if c.HealthCheckUsernameHash != "" {
		if _, err := bcrypt.Cost([]byte(c.HealthCheckUsernameHash)); err != nil {
			return fmt.Errorf("%w: healthcheck username_hash is not a valid bcrypt hash", ConfigurationErr)
		}
	}

	if c.HealthCheckPasswordHash != "" {
		if _, err := bcrypt.Cost([]byte(c.HealthCheckPasswordHash)); err != nil {
			return fmt.Errorf("%w: healthcheck password_hash is not a valid bcrypt hash", ConfigurationErr)
		}
	}

	if c.HealthCheckUsername == "" && c.HealthCheckPassword != "" {
		return fmt.Errorf("%w: healthcheck username is empty", ConfigurationErr)
	}

	if c.HealthCheckUsername != "" && c.HealthCheckPassword == "" {
		return fmt.Errorf("%w: healthcheck password is empty", ConfigurationErr)
	}

	return nil
}
