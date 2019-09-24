package ratelimiter

import (
	"time"

	"code.cloudfoundry.org/lager"
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

func NewRateLimiter(limitPerMinute int, expireDuration time.Duration, logger lager.Logger) *RateLimiter {
	return &RateLimiter{
		store: NewStore(limitPerMinute, expireDuration, logger),
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
