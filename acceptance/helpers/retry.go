package helpers

import (
	"fmt"
	"log"
	"time"
)

const defaultRetryAttempt = 3
const defaultRetryAfter = 60

func Retry(attempts int, sleep int, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			log.Println(fmt.Sprintf("attempt %d/%d - retrying in %d seconds error ", i, attempts, sleep), err)
			time.Sleep(time.Duration(sleep) * time.Second)
			sleep *= 2
		}
		err = f()
		if err == nil {
			return
		}
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func TRetry[T any](attempts int, sleep int, f func() (T, error)) (result T, err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			log.Println(fmt.Sprintf("attempt %d/%d - retrying in %d seconds error ", i, attempts, sleep), err)
			time.Sleep(time.Duration(sleep) * time.Second)
			sleep *= 2
		}
		result, err = f()
		if err == nil {
			return result, nil
		}
	}
	return result, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
