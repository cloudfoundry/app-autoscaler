package healthendpoint

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

type (
	Pinger interface {
		Ping() error
	}
	ReadinessCheck struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}
	readinessResponse struct {
		OverallStatus string           `json:"overall_status"`
		Checks        []ReadinessCheck `json:"checks"`
	}
	Checker func() ReadinessCheck
)

const (
	statusUp   = "UP"
	statusDown = "DOWN"
)

type cacheEntry struct {
	response     *http.Response
	responseTime time.Time
	body         []byte
	sync.RWMutex
}

func (c *cacheEntry) send(w http.ResponseWriter) {
	for k, v := range c.response.Header {
		w.Header().Set(k, strings.Join(v, ","))
	}
	w.WriteHeader(c.response.StatusCode)
	_, _ = w.Write(c.body)
}

func (c *cacheEntry) isNotStale(currentTime time.Time) bool {
	return currentTime.Before(c.responseTime.Add(30 * time.Second))
}

type handler func(w http.ResponseWriter, r *http.Request, vars map[string]string)

func cachedResponse(timefunc func() time.Time, next handler) handler {
	cached := cacheEntry{}
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		currentTime := timefunc()
		cached.RLock()
		if cached.isNotStale(currentTime) {
			defer cached.RUnlock()
			cached.send(w)
			return
		}
		cached.RUnlock()

		cached.Lock()
		defer cached.Unlock()
		if cached.isNotStale(currentTime) {
			cached.send(w)
			return
		}
		recorder := httptest.NewRecorder()
		next(recorder, r, vars)
		cached.save(currentTime, recorder)
		cached.send(w)
	}
}

func (c *cacheEntry) save(currentTime time.Time, recorder *httptest.ResponseRecorder) {
	c.responseTime = currentTime
	c.response = recorder.Result()
	c.body = recorder.Body.Bytes()
}

func readiness(checkers []Checker, timefunc func() time.Time) handler {
	return cachedResponse(timefunc, readinessHandler(checkers))
}
func readinessHandler(checkers []Checker) handler {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		checks := make([]ReadinessCheck, 0, 8)
		overallStatus := statusUp
		for _, checker := range checkers {
			check := checker()
			checks = append(checks, check)
			if check.Status == statusDown {
				overallStatus = statusDown
			}
		}
		response, err := json.Marshal(readinessResponse{OverallStatus: overallStatus, Checks: checks})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"Internal error"}`))
		}
		_, _ = w.Write(response)
	}
}

func DbChecker(dbName string, pinger Pinger) Checker {
	if pinger != nil {
		return func() ReadinessCheck {
			status := statusUp
			err := pinger.Ping()
			if err != nil {
				status = statusDown
			}
			return ReadinessCheck{Name: dbName, Type: "database", Status: status}
		}
	} else {
		return func() ReadinessCheck {
			return ReadinessCheck{Name: dbName, Type: "database", Status: statusUp}
		}
	}
}
