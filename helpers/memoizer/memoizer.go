package memoizer

import "sync"

func New[T comparable, R any](funcToMemoize func(T) (R, error)) *Memoizer[T, R] {
	return &Memoizer[T, R]{fn: funcToMemoize, cache: make(map[T]R)}
}

type Memoizer[T comparable, R any] struct {
	//The nolint comments are because struct check cant work out that they are used atm. 17/08/2022
	mu    sync.RWMutex       //nolint:structcheck
	cache map[T]R            //nolint:structcheck
	fn    func(T) (R, error) //nolint:structcheck
}

func (m *Memoizer[T, R]) Func(param T) (R, error) {
	result, found := m.getCache(param)
	if found {
		return result, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	//for the case where there are 2 or more waiting on the Lock at the same time.
	result, found = m.cache[param]
	if found {
		return result, nil
	}

	result, err := m.fn(param)
	if err != nil {
		return result, err
	}
	m.cache[param] = result
	return result, err
}

func (m *Memoizer[T, R]) getCache(param T) (R, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result, found := m.cache[param]
	return result, found
}
