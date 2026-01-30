package app

import (
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

//counterfeiter:generate . CPUWaster
type CPUWaster interface {
	UseCPU(utilisation int64, duration time.Duration)
	IsRunning() bool
	StopTest()
}

type ConcurrentBusyLoopCPUWaster struct {
	mu        sync.RWMutex
	isRunning bool
}

var _ CPUWaster = &ConcurrentBusyLoopCPUWaster{}

func CPUTests(logger *slog.Logger, mux *http.ServeMux, cpuTest CPUWaster) {
	mux.HandleFunc("GET /cpu/{utilization}/{minutes}", func(w http.ResponseWriter, r *http.Request) {
		if cpuTest.IsRunning() {
			Errorf(logger, w, http.StatusConflict, "CPU test is already running")
			return
		}
		var utilization int64
		var minutes int64
		var err error
		utilization, err = strconv.ParseInt(r.PathValue("utilization"), 10, 64)
		if err != nil {
			Errorf(logger, w, http.StatusBadRequest, "invalid utilization: %s", err.Error())
			return
		}
		if minutes, err = strconv.ParseInt(r.PathValue("minutes"), 10, 64); err != nil {
			Errorf(logger, w, http.StatusBadRequest, "invalid minutes: %s", err.Error())
			return
		}
		duration := time.Duration(minutes) * time.Minute
		go func() {
			cpuTest.UseCPU(utilization, duration)
		}()
		if err := writeJSON(w, http.StatusOK, JSONResponse{"utilization": utilization, "minutes": minutes}); err != nil {
			logger.Error("Failed to write JSON response", slog.Any("error", err))
		}
	})

	mux.HandleFunc("GET /cpu/close", func(w http.ResponseWriter, r *http.Request) {
		if !cpuTest.IsRunning() {
			Errorf(logger, w, http.StatusConflict, "CPU test is not running")
			return
		}
		cpuTest.StopTest()
		if err := writeJSON(w, http.StatusOK, JSONResponse{"status": "close cpu test"}); err != nil {
			logger.Error("Failed to write JSON response", slog.Any("error", err))
		}
	})
}

func (m *ConcurrentBusyLoopCPUWaster) UseCPU(utilisation int64, duration time.Duration) {
	m.startTest()

	for utilisation > 0 {
		// to allow to use more than one CPU, we distribute the utilisation to multiple goroutines
		perGoRoutineUtilisation := min(utilisation, 100)
		utilisation = utilisation - perGoRoutineUtilisation

		// the core cpu wasting goroutine
		go func(util int64) {
			// to achieve a desired utilisation, we run a busy loop for a certain percentage of time and then wait for the remainder
			// concretely, we split a second into two parts: one busy loop and one sleep
			// we repeat this "second" until the test is stopped
			run := time.Duration(util) * time.Second / 100
			sleep := time.Duration(100-util) * time.Second / 100
			runtime.LockOSThread()
			for m.IsRunning() {
				begin := time.Now()
				for time.Since(begin) < run {
					// burn cpu time
				}
				time.Sleep(sleep)
			}
			runtime.UnlockOSThread()
		}(perGoRoutineUtilisation)
	}

	// how long
	go func() {
		time.Sleep(duration)
		m.StopTest()
	}()
}

func (m *ConcurrentBusyLoopCPUWaster) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

func (m *ConcurrentBusyLoopCPUWaster) StopTest() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isRunning = false
}

func (m *ConcurrentBusyLoopCPUWaster) startTest() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isRunning = true
}
