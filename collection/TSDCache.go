package collection

import (
	"fmt"
	"sync"
)

type TSD interface {
	GetTimestamp() int64
	HasLabels(map[string]string) bool
}

type TSDCache struct {
	lock     *sync.RWMutex
	data     []TSD
	capacity int
	cursor   int
	num      int
}

func NewTSDCache(capacity int) *TSDCache {
	if capacity <= 0 {
		panic("invalid TSDCache capacity")
	}
	return &TSDCache{
		lock:     &sync.RWMutex{},
		data:     make([]TSD, capacity),
		capacity: capacity,
		cursor:   0,
	}
}

func (c *TSDCache) binarySearch(t int64) int {
	if c.num == 0 {
		return 0
	}
	var l, r int
	if c.data[c.cursor] == nil {
		l = 0
		r = c.cursor - 1
	} else {
		l = c.cursor
		r = c.cursor - 1 + c.capacity
	}

	for {
		if l > r {
			return l
		}
		m := l + (r-l)/2
		if t <= c.data[m%c.capacity].GetTimestamp() {
			r = m - 1
		} else {
			l = m + 1
		}
	}
}

func (c *TSDCache) Put(d TSD) {
	c.lock.Lock()
	defer c.lock.Unlock()

	defer func() {
		c.num++
	}()

	if c.num == 0 || d.GetTimestamp() >= c.data[((c.cursor-1)+c.capacity)%c.capacity].GetTimestamp() {
		c.data[c.cursor] = d
		c.cursor = (c.cursor + 1) % c.capacity
		return
	}

	pos := c.binarySearch(d.GetTimestamp())
	if pos == c.cursor && c.data[c.cursor] != nil {
		return
	}

	end := c.cursor
	if c.data[end] != nil {
		end += c.capacity
	}
	for i := end; i > pos; i-- {
		c.data[i%c.capacity] = c.data[(i-1)%c.capacity]
	}
	c.data[pos%c.capacity] = d
	c.cursor = (c.cursor + 1) % c.capacity
}

func (c *TSDCache) String() string {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var head, tail int
	if c.data[c.cursor] == nil {
		head = 0
		tail = c.cursor - 1
	} else {
		head = c.cursor
		tail = c.cursor + c.capacity - 1
	}

	s := make([]TSD, tail-head+1)
	for i := 0; i <= tail-head; i++ {
		s[i] = c.data[(i+head)%c.capacity]
	}
	return fmt.Sprint(s)
}

// Query returns the time series with the timestamp in [start, end)
// If it can not guarantee all the data are in cache, it returns ([start, end), false)
func (c *TSDCache) Query(start, end int64, labels map[string]string) ([]TSD, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.num == 0 {
		return []TSD{}, false
	}

	result := []TSD{}
	from := c.binarySearch(start)
	to := c.binarySearch(end)
	for i := from; i < to; i++ {
		d := c.data[i%c.capacity]
		if d.HasLabels(labels) {
			result = append(result, d)
		}
	}

	if c.num < c.capacity {
		return result, c.data[0].GetTimestamp() <= start
	}
	return result, c.data[c.cursor].GetTimestamp() <= start
}
