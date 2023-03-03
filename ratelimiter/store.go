package ratelimiter

import (
	"errors"
	"sync"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"golang.org/x/time/rate"
)

type Store interface {
	Increment(string) error
}

type InMemoryStore struct {
	bucketCapacity      int
	maxAmount           int
	validDuration       time.Duration
	expireDuration      time.Duration
	expireCheckInterval time.Duration
	storage             map[string]*entry
	logger              lager.Logger
	sync.RWMutex
}

type entry struct {
	limiter   *rate.Limiter
	expiredAt time.Time
	sync.RWMutex
}

func (e *entry) Expired() bool {
	e.RLock()
	defer e.RUnlock()
	return time.Now().After(e.expiredAt)
}

func (e *entry) SetExpire(expiredAt time.Time) {
	e.Lock()
	defer e.Unlock()
	e.expiredAt = expiredAt
}

func NewStore(bucketCapacity int, maxAmount int, validDuration time.Duration, expireDuration time.Duration, expireCheckInterval time.Duration, logger lager.Logger) Store {
	store := &InMemoryStore{
		bucketCapacity:      bucketCapacity,
		maxAmount:           maxAmount,
		validDuration:       validDuration,
		expireDuration:      expireDuration,
		expireCheckInterval: expireCheckInterval,
		storage:             make(map[string]*entry),
		logger:              logger,
	}
	store.expiryCycle()

	return store
}

func newEntry(validDuration time.Duration, bucketCapacity int, maxAmount int) *entry {
	limit := 1e9 * float64(maxAmount) / float64(validDuration)
	return &entry{
		limiter: rate.NewLimiter(rate.Limit(limit), bucketCapacity),
	}
}

func (s *InMemoryStore) Increment(key string) error {
	v, ok := s.get(key)
	if !ok {
		v = newEntry(s.validDuration, s.bucketCapacity, s.maxAmount)
	}
	v.SetExpire(time.Now().Add(s.expireDuration))
	if !v.limiter.Allow() {
		s.set(key, v)
		return errors.New("empty bucket")
	}
	s.set(key, v)
	return nil
}

func (s *InMemoryStore) get(key string) (*entry, bool) {
	s.RLock()
	defer s.RUnlock()
	v, ok := s.storage[key]
	return v, ok
}

func (s *InMemoryStore) set(key string, value *entry) {
	s.Lock()
	defer s.Unlock()
	s.storage[key] = value
}

func (s *InMemoryStore) expiryCycle() {
	ticker := time.NewTicker(s.expireCheckInterval)
	go func() {
		for range ticker.C {
			s.Lock()
			for k, v := range s.storage {
				if v.Expired() {
					s.logger.Info("removing-expired-key", lager.Data{"key": k})
					delete(s.storage, k)
				}
			}
			s.Unlock()
		}
	}()
}
