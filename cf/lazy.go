package cf

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type Lazy[T any] struct {
	mu    sync.Mutex        //nolint:structcheck
	value *T                //nolint:structcheck
	fn    func() (T, error) //nolint:structcheck
}

func NewLazy[T any](fn func() (T, error)) *Lazy[T] {
	return &Lazy[T]{fn: fn}
}
func (o *Lazy[T]) Get() (T, error) {
	value := (*T)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&o.value))))
	if value == nil {
		o.mu.Lock()
		defer o.mu.Unlock()
		value = o.value
		if value == nil {
			v, err := o.fn()
			if err != nil {
				return v, err
			}
			value = &v
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&o.value)), unsafe.Pointer(value))
		}
	}
	return *value, nil
}
