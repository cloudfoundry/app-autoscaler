package cf_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	"github.com/stretchr/testify/assert"
)

func testFunc(counter *int, err error) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		*counter++
		time.Sleep(10 * time.Millisecond)
		return "someString", err
	}
}

func TestLazy_init(t *testing.T) {
	counter := 0
	lazy := cf.NewLazy(testFunc(&counter, nil))
	res, err := lazy.Get(context.Background())
	assert.Equal(t, "someString", res)
	assert.Nil(t, err)
	assert.Equal(t, counter, 1)
}

func TestLazy_hit(t *testing.T) {
	counter := 0
	lazy := cf.NewLazy(testFunc(&counter, nil))
	res, err := lazy.Get(context.Background())
	assert.Equal(t, "someString", res)
	assert.Nil(t, err)
	_, err = lazy.Get(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 1, counter)
}

func TestLazy_errorsDontGetCached(t *testing.T) {
	counter := 0
	lazy := cf.NewLazy(testFunc(&counter, errors.New("someErr")))
	_, err := lazy.Get(context.Background())
	assert.NotNil(t, err)
	_, err = lazy.Get(context.Background())
	assert.NotNil(t, err)
	assert.Equal(t, 2, counter)
}

func TestLazy_Thread_test(t *testing.T) {
	counter := 0
	numThreads := 100
	lazy := cf.NewLazy(testFunc(&counter, nil))
	mu := sync.RWMutex{}
	mu.Lock()
	wg := sync.WaitGroup{}
	wg.Add(numThreads)
	for i := 0; i < numThreads; i++ {
		go func() {
			mu.RLock()
			_, _ = lazy.Get(context.Background())
			wg.Done()
		}()
	}
	mu.Unlock()
	wg.Wait()
	assert.Equal(t, 1, counter)
}
