package ratelimiter

import (
	"errors"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/juju/ratelimit"
)

const expireInMin = 10 * time.Minute

type Store interface {
	Increment(string) (int, error)
	Stats() map[string]int
}

type InMemoryStore struct {
	limitPerMinute int
	expireDuration time.Duration
	storage        map[string]*entry
	logger         lager.Logger
	sync.RWMutex
}

type entry struct {
	bucket    *ratelimit.Bucket
	expiredAt time.Time
}

func (e *entry) Expired() bool {
	return time.Now().After(e.expiredAt)
}

func NewStore(limitPerMinute int, expireDuration time.Duration, logger lager.Logger) Store {
	store := &InMemoryStore{
		limitPerMinute: limitPerMinute,
		expireDuration: expireDuration,
		logger:         logger,
		storage:        make(map[string]*entry),
	}
	store.expiryCycle()

	return store
}

func newEntry(limit int) *entry {
	fillRatePerMin := (1000 * 60) / limit
	return &entry{
		bucket: ratelimit.NewBucket(time.Duration(fillRatePerMin)*time.Millisecond, int64(limit)),
	}
}

func (s *InMemoryStore) Increment(key string) (int, error) {
	v, ok := s.get(key)
	if !ok {
		v = newEntry(s.limitPerMinute)
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
	ticker := time.NewTicker(time.Second * 30)
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
