package app

import (
	"bytes"
	"container/list"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/procfs"
)

const (
	_ = 1 << (10 * iota)
	Kibi
	Mebi
)

//counterfeiter:generate . MemoryGobbler
type MemoryGobbler interface {
	UseMemory(numBytes int64)
	Sleep(sleepTime time.Duration)
	IsRunning() bool
	StopTest()
}

type ListBasedMemoryGobbler struct {
	mu        sync.RWMutex
	used      *list.List
	isRunning bool
}

var _ MemoryGobbler = &ListBasedMemoryGobbler{}

func parsePositiveInt64(r *http.Request, name string) (int64, error) {
	val, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %s", name, err.Error())
	}
	if val < 1 {
		return 0, fmt.Errorf("%s must be > 0", name)
	}
	return val, nil
}

func MemoryTests(logger *slog.Logger, mux *http.ServeMux, memoryTest MemoryGobbler) {
	mux.HandleFunc("GET /memory/{memoryMiB}/{minutes}", func(w http.ResponseWriter, r *http.Request) {
		if memoryTest.IsRunning() {
			Errorf(logger, w, http.StatusConflict, "memory test is already running")
			return
		}
		memoryMiB, err := parsePositiveInt64(r, "memoryMiB")
		if err != nil {
			Errorf(logger, w, http.StatusBadRequest, "%s", err.Error())
			return
		}
		minutes, err := parsePositiveInt64(r, "minutes")
		if err != nil {
			Errorf(logger, w, http.StatusBadRequest, "%s", err.Error())
			return
		}
		duration := time.Duration(minutes) * time.Minute
		numBytes := memoryMiB * Mebi
		logger.Info("Starting memory test",
			"memoryMiB", memoryMiB,
			"minutes", minutes,
			"bytes", numBytes)
		go func() {
			logMemoryUsage(logger, "before memory test")
			memoryTest.UseMemory(numBytes)
			logMemoryUsage(logger, "after allocating memory")
			memoryTest.Sleep(duration)
			memoryTest.StopTest()
		}()
		if err := writeJSON(w, http.StatusOK, JSONResponse{
			"memoryMiB": memoryMiB,
			"minutes":   minutes,
		}); err != nil {
			logger.Error("Failed to write JSON response", slog.Any("error", err))
		}
	})

	mux.HandleFunc("GET /memory/close", func(w http.ResponseWriter, r *http.Request) {
		if !memoryTest.IsRunning() {
			Errorf(logger, w, http.StatusConflict, "memory test is not running")
			return
		}
		memoryTest.StopTest()
		logMemoryUsage(logger, "after freeing memory")
		if err := writeJSON(w, http.StatusOK, JSONResponse{"status": "close memory test"}); err != nil {
			logger.Error("Failed to write JSON response", slog.Any("error", err))
		}
	})
}

const chunkSize = 4 * Kibi

func (m *ListBasedMemoryGobbler) UseMemory(numBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isRunning = true
	m.used = list.New()
	used := int64(0)
	for used <= numBytes {
		m.used.PushBack(bytes.Repeat([]byte("X"), chunkSize)) // The bytes need to be non-zero to force memory allocation
		used += chunkSize
	}
}

func (m *ListBasedMemoryGobbler) Sleep(sleepTime time.Duration) {
	sleepTill := time.Now().Add(sleepTime)
	for m.IsRunning() && time.Now().Before(sleepTill) {
		time.Sleep(100 * time.Millisecond)
	}
}

func (m *ListBasedMemoryGobbler) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

func (m *ListBasedMemoryGobbler) StopTest() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isRunning = false
	m.used.Init()
	runtime.GC()
}

func logMemoryUsage(logger *slog.Logger, action string) {
	logAction := slog.String("action", action)
	memoryUsage, err := getTotalMemoryUsage()
	if err == nil {
		logger.Info("memory usage", "usage", memoryUsage/Mebi, logAction)
	} else {
		logger.Error("could not determine memory usage", slog.Any("error", err), logAction)
	}
}
func getTotalMemoryUsage() (uint64, error) {
	fs, err := procfs.NewFS("/proc")
	if err != nil {
		return 0, err
	}

	proc, err := fs.Proc(os.Getpid())
	if err != nil {
		return 0, err
	}

	stat, err := proc.NewStatus()
	if err != nil {
		return 0, err
	}

	result := stat.VmRSS + stat.VmSwap

	return result, nil
}
