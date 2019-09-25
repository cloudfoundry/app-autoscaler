package models

import "time"

type RateLimitConfig struct {
	FillInterval time.Duration `yaml:"fill_interval"`
}
