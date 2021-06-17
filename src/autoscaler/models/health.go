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

func (c *HealthConfig) Validate() error {

	if c.HealthCheckUsername != "" && c.HealthCheckUsernameHash != "" {
		return fmt.Errorf("Configuration error: both healthcheck username and healthcheck username_hash are set, please provide only one of them")
	}

	if c.HealthCheckPassword != "" && c.HealthCheckPasswordHash != "" {
		return fmt.Errorf("Configuration error: both healthcheck password and healthcheck password_hash are empty, please provide one of them")
	}

	if c.HealthCheckUsernameHash != "" {
		if _, err := bcrypt.Cost([]byte(c.HealthCheckUsernameHash)); err != nil {
			return fmt.Errorf("Configuration error: healthcheck username_hash is not a valid bcrypt hash")
		}
	}

	if c.HealthCheckPasswordHash != "" {
		if _, err := bcrypt.Cost([]byte(c.HealthCheckPasswordHash)); err != nil {
			return fmt.Errorf("Configuration error: healthcheck password_hash is not a valid bcrypt hash")
		}
	}

	if c.HealthCheckUsername == "" && c.HealthCheckPassword != "" {
		return fmt.Errorf("Configuration error: healthcheck username is empty")
	}

	if c.HealthCheckUsername != "" && c.HealthCheckPassword == "" {
		return fmt.Errorf("Configuration error: healthcheck password is empty")
	}

	return nil
}
