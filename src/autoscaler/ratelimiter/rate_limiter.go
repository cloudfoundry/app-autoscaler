package ratelimiter

import (
	"time"

	"code.cloudfoundry.org/lager"
)

const (
	defaultBucketCapacity      = 20
	defaultExpireDuration      = 10 * time.Minute
	defaultExpireCheckInterval = 30 * time.Second
)

type Stats []Stat
type Stat struct {
	Key       string `json:"key"`
	Available int    `json:"available"`
}

type Limiter interface {
	ExceedsLimit(string) bool
}

type RateLimiter struct {
	duration time.Duration
	store    Store
	logger   lager.Logger
}

func DefaultRateLimiter(maxAmount int, validDuration time.Duration, logger lager.Logger) *RateLimiter {
	return NewRateLimiter(defaultBucketCapacity, maxAmount, validDuration, defaultExpireDuration, defaultExpireCheckInterval, logger)
}

func NewRateLimiter(bucketCapacity int, maxAmount int, validDuration time.Duration, expireDuration time.Duration, expireCheckInterval time.Duration, logger lager.Logger) *RateLimiter {
	return &RateLimiter{
		store: NewStore(bucketCapacity, maxAmount, validDuration, expireDuration, expireCheckInterval, logger),
	}
}

func (r *RateLimiter) ExceedsLimit(key string) bool {
	if _, err := r.store.Increment(key); err != nil {
		return true
	}

	return false
}

func (r *RateLimiter) GetStats() Stats {
	s := Stats{}
	for k, v := range r.store.Stats() {
		s = append(s, Stat{
			Key:       k,
			Available: v,
		})
	}
	return s
}
