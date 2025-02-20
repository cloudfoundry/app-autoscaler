package helpers

import (
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"golang.org/x/crypto/bcrypt"
)

type HealthConfig struct {
	ServerConfig          ServerConfig     `yaml:"server_config" json:"server_config"`
	BasicAuth             models.BasicAuth `yaml:"basic_auth" json:"basic_auth"`
	ReadinessCheckEnabled bool             `yaml:"readiness_enabled" json:"readiness_enabled"`
}

var ErrConfiguration = fmt.Errorf("configuration error")

func (c *HealthConfig) Validate() error {
	if c.BasicAuth.Username != "" && c.BasicAuth.UsernameHash != "" {
		return fmt.Errorf("%w: both healthcheck username and healthcheck username_hash are set, please provide only one of them", ErrConfiguration)
	}

	if c.BasicAuth.Password != "" && c.BasicAuth.PasswordHash != "" {
		return fmt.Errorf("%w: both healthcheck password and healthcheck password_hash are provided, please provide only one of them", ErrConfiguration)
	}

	if c.BasicAuth.UsernameHash != "" {
		if _, err := bcrypt.Cost([]byte(c.BasicAuth.UsernameHash)); err != nil {
			return fmt.Errorf("%w: healthcheck username_hash is not a valid bcrypt hash", ErrConfiguration)
		}
	}

	if c.BasicAuth.PasswordHash != "" {
		if _, err := bcrypt.Cost([]byte(c.BasicAuth.PasswordHash)); err != nil {
			return fmt.Errorf("%w: healthcheck password_hash is not a valid bcrypt hash", ErrConfiguration)
		}
	}

	if c.BasicAuth.Username == "" && c.BasicAuth.Password != "" {
		return fmt.Errorf("%w: healthcheck username is empty", ErrConfiguration)
	}

	if c.BasicAuth.Username != "" && c.BasicAuth.Password == "" {
		return fmt.Errorf("%w: healthcheck password is empty", ErrConfiguration)
	}

	return nil
}
