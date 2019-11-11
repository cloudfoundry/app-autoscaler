package models

import "time"

type RateLimitConfig struct {
	MaxAmount     int           `yaml:"max_amount"`
	ValidDuration time.Duration `yaml:"valid_duration"`
}
