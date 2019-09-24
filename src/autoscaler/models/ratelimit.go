package models

import "time"

type RateLimitConfig struct {
	LimitPerMinute int           `yaml:"limit_per_minute"`
	ExpireDuration time.Duration `yaml:"expire_duration"`
}
