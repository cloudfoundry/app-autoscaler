package scalingengine

import (
	"hash/fnv"
	"sync"
)

type StripedLock struct {
	locks []*sync.Mutex
}

func NewStripedLock(capacity int) *StripedLock {
	if capacity <= 0 {
		panic("invalid striped lock capacity")
	}

	locks := make([]*sync.Mutex, capacity)
	for i := 0; i < capacity; i++ {
		locks[i] = &sync.Mutex{}
	}

	return &StripedLock{
		locks: locks,
	}
}

func (sl *StripedLock) GetLock(key string) *sync.Mutex {
	h := fnv.New32a()
	_, err := h.Write([]byte(key))
	if err != nil {
		return nil
	}
	return sl.locks[int(h.Sum32())%len(sl.locks)]
}
