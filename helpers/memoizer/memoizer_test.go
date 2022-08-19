package memoizer_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/memoizer"

	"github.com/stretchr/testify/assert"
)

func testFunc(counter *int) func(string) (string, error) {
	return func(name string) (string, error) {
		*counter++
		if len(name) < 3 {
			return "", errors.New("failed")
		}
		time.Sleep(10 * time.Millisecond)
		return name[:3], nil
	}
}

func TestMemoiser_miss(t *testing.T) {
	counter := 0
	memo := memoizer.New(testFunc(&counter))
	res, err := memo.Func("string")
	assert.Equal(t, "str", res)
	assert.Nil(t, err)
	assert.Equal(t, counter, 1)
}

func TestMemoiser_hit(t *testing.T) {
	counter := 0
	memo := memoizer.New(testFunc(&counter))
	res, err := memo.Func("string")
	assert.Equal(t, "str", res)
	assert.Nil(t, err)
	res, err = memo.Func("string")
	assert.Equal(t, "str", res)
	assert.Nil(t, err)
	assert.Equal(t, 1, counter)
}

func TestMemoiser_errorsDontGetCached(t *testing.T) {
	counter := 0
	memo := memoizer.New(testFunc(&counter))
	_, err := memo.Func("st")
	assert.NotNil(t, err)
	_, err = memo.Func("st")
	assert.NotNil(t, err)
	res, err := memo.Func("string")
	assert.Nil(t, err)
	assert.Equal(t, "str", res)
	_, _ = memo.Func("string")
	assert.Equal(t, 3, counter)
}

func TestMemoiser_Thread_test(t *testing.T) {
	counter := 0
	numThreads := 100
	memo := memoizer.New(testFunc(&counter))
	mu := sync.RWMutex{}
	mu.Lock()
	wg := sync.WaitGroup{}
	wg.Add(numThreads)
	for i := 0; i < numThreads; i++ {
		go func() {
			mu.RLock()
			_, _ = memo.Func("string")
			wg.Done()
		}()
	}
	mu.Unlock()
	wg.Wait()
	assert.Equal(t, 1, counter)
}
