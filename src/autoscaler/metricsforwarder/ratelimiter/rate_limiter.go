package ratelimiter

import (
	"fmt"
	"time"
)

type Stats []Stat
type Stat struct {
	Ip        string `json:"ip"`
	Available int    `json:"available"`
}

type Limiter interface {
	ExceedsLimit(string) bool
}

type RateLimiter struct {
	duration time.Duration
	store    Store
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		store: NewStore(limit),
	}
}

func (r *RateLimiter) ExceedsLimit(ip string) bool {
	if _, err := r.store.Increment(ip); err != nil {
		fmt.Printf("rate limit exceeded for %s\n", ip)
		return true
	}

	return false
}

func (r *RateLimiter) GetStats() Stats {
	s := Stats{}
	for k, v := range r.store.Stats() {
		s = append(s, Stat{
			Ip:        k,
			Available: v,
		})
	}
	return s
}
