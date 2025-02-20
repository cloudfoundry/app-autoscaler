package models

import "time"

type RateLimitConfig struct {
	MaxAmount     int           `yaml:"max_amount" json:"max_amount,omitempty"`
	ValidDuration time.Duration `yaml:"valid_duration" json:"valid_duration,omitempty"`
}
