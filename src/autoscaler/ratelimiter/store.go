package ratelimiter

import (
	"errors"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/juju/ratelimit"
)

type Store interface {
	Increment(string) (int, error)
	Stats() map[string]int
}

type InMemoryStore struct {
	bucketCapacity      int
	fillInterval        time.Duration
	expireDuration      time.Duration
	expireCheckInterval time.Duration
	storage             map[string]*entry
	logger              lager.Logger
	sync.RWMutex
}

type entry struct {
	bucket    *ratelimit.Bucket
	expiredAt time.Time
}

func (e *entry) Expired() bool {
	return time.Now().After(e.expiredAt)
}

func NewStore(bucketCapacity int, fillInterval time.Duration, expireDuration time.Duration, expireCheckInterval time.Duration, logger lager.Logger) Store {
	store := &InMemoryStore{
		bucketCapacity:      bucketCapacity,
		fillInterval:        fillInterval,
		expireDuration:      expireDuration,
		expireCheckInterval: expireCheckInterval,
		storage:             make(map[string]*entry),
		logger:              logger,
	}
	store.expiryCycle()

	return store
}

func newEntry(fillInterval time.Duration, bucketCapacity int) *entry {
	return &entry{
		bucket: ratelimit.NewBucket(fillInterval, int64(bucketCapacity)),
	}
}

func (s *InMemoryStore) Increment(key string) (int, error) {
	v, ok := s.get(key)
	if !ok {
		v = newEntry(s.fillInterval, s.bucketCapacity)
	}
	v.expiredAt = time.Now().Add(s.expireDuration)
	if avail := v.bucket.Available(); avail == 0 {
		s.set(key, v)
		return int(avail), errors.New("empty bucket")
	}
	v.bucket.Take(1)
	s.set(key, v)
	return int(v.bucket.Available()), nil
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
		for _ = range ticker.C {
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

func (s *InMemoryStore) Available(key string) int {
	v, ok := s.get(key)
	if !ok {
		return 0
	}
	return int(v.bucket.Available())
}

func (s *InMemoryStore) Stats() map[string]int {
	m := make(map[string]int)
	s.Lock()
	for k, v := range s.storage {
		m[k] = int(v.bucket.Available())
	}
	s.Unlock()
	return m
}
