package ratelimiter

import (
	"time"

	"code.cloudfoundry.org/lager/v3"
)

const (
	defaultBucketCapacity      = 20
	defaultExpireDuration      = 10 * time.Minute
	defaultExpireCheckInterval = 30 * time.Second
)

type Limiter interface {
	ExceedsLimit(string) bool
}

type RateLimiter struct {
	store Store
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
	if err := r.store.Increment(key); err != nil {
		return true
	}

	return false
}
