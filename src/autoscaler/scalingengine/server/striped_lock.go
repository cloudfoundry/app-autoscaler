package server

import (
	"hash/crc32"
	"sync"
)

type StripedLock struct {
	capacity int
	sLock    *sync.Mutex
	locks    []*sync.Mutex
}

func NewStripedLock(capacity int) *StripedLock {
	if capacity <= 0 {
		panic("invalid striped lock capacity")
	}
	return &StripedLock{
		capacity: capacity,
		sLock:    &sync.Mutex{},
		locks:    make([]*sync.Mutex, capacity),
	}
}

func (sl *StripedLock) GetLock(key string) *sync.Mutex {
	idx := crc32.ChecksumIEEE([]byte(key)) % uint32(sl.capacity)
	sl.sLock.Lock()
	if sl.locks[idx] == nil {
		sl.locks[idx] = &sync.Mutex{}
	}
	sl.sLock.Unlock()
	return sl.locks[idx]
}
