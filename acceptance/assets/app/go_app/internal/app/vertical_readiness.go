package app

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// VerticalReadinessHandler is an opt-in startup/readiness fixture. Its settings
// are captured once per container, so an env update affects replacements, not
// the old instances serving a rolling deployment. It never allocates excessive
// memory or changes the host/container resource limits.
func VerticalReadinessHandler(next http.Handler) http.Handler {
	if os.Getenv("VERTICAL_TEST_ENABLED") != "true" {
		return next
	}
	started := time.Now()
	delay, delayErr := strconv.Atoi(os.Getenv("VERTICAL_TEST_READINESS_DELAY_SECONDS"))
	minimum, minimumErr := strconv.Atoi(os.Getenv("VERTICAL_TEST_MIN_MEMORY_MB"))
	if os.Getenv("VERTICAL_TEST_MIN_MEMORY_MB") == "" {
		minimum, minimumErr = 0, nil
	}
	limit, limitErr := memoryLimitMiB(os.Getenv("MEMORY_LIMIT"))
	blocked := os.Getenv("VERTICAL_TEST_READINESS_BLOCK") == "true"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ready := !blocked && delayErr == nil && delay >= 0 && minimumErr == nil && minimum >= 0 &&
			time.Since(started) >= time.Duration(delay)*time.Second &&
			(minimum == 0 || (limitErr == nil && limit >= minimum))
		if !ready {
			http.Error(w, "vertical test application is not ready", http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/vertical/ready" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, `{"status":"ready"}`)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func memoryLimitMiB(value string) (int, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	multiplier := 1
	if strings.HasSuffix(value, "g") {
		multiplier = 1024
		value = strings.TrimSuffix(value, "g")
	} else {
		value = strings.TrimSuffix(value, "m")
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > 1048576 {
		return 0, fmt.Errorf("invalid MEMORY_LIMIT")
	}
	return n * multiplier, nil
}
