package app

import (
	"crypto/rand"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

func DiskTest(logger *slog.Logger, mux *http.ServeMux, diskOccupier DiskOccupier) {
	mux.HandleFunc("GET /disk/{utilization}/{minutes}", func(w http.ResponseWriter, r *http.Request) {
		utilisation, err := parsePositiveInt64(r, "utilization")
		if err != nil {
			respondWithErrorf(logger, w, http.StatusBadRequest, "%s", err.Error())
			return
		}
		minutes, err := parsePositiveInt64(r, "minutes")
		if err != nil {
			respondWithErrorf(logger, w, http.StatusBadRequest, "%s", err.Error())
			return
		}
		duration := time.Duration(minutes) * time.Minute
		spaceInMB := utilisation * 1000 * 1000
		if err = diskOccupier.Occupy(spaceInMB, duration); err != nil {
			respondWithErrorf(logger, w, http.StatusInternalServerError, "error invoking occupation: %s", err.Error())
			return
		}
		respondJSON(logger, w, JSONResponse{"utilization": utilisation, "minutes": minutes})
	})

	mux.HandleFunc("GET /disk/close", func(w http.ResponseWriter, r *http.Request) {
		diskOccupier.Stop()
		respondJSON(logger, w, JSONResponse{"message": "close disk test"})
	})
}

//counterfeiter:generate . DiskOccupier
type DiskOccupier interface {
	Occupy(space int64, duration time.Duration) error
	Stop()
}

type defaultDiskOccupier struct {
	mu        sync.RWMutex
	isRunning bool
	filePath  string
}

func NewDefaultDiskOccupier(filePath string) *defaultDiskOccupier {
	return &defaultDiskOccupier{
		filePath: filePath,
	}
}

func (d *defaultDiskOccupier) Occupy(space int64, duration time.Duration) error {
	if err := d.checkAlreadyRunning(); err != nil {
		return err
	}

	if err := d.occupy(space); err != nil {
		return err
	}

	d.stopAfter(duration)

	return nil
}

func (d *defaultDiskOccupier) checkAlreadyRunning() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.isRunning {
		return errors.New("disk space is already being occupied")
	}

	return nil
}

func (d *defaultDiskOccupier) occupy(space int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	file, err := os.Create(d.filePath)
	if err != nil {
		return err
	}
	if _, err := io.CopyN(file, io.LimitReader(rand.Reader, space), space); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	d.isRunning = true

	return nil
}

func (d *defaultDiskOccupier) stopAfter(duration time.Duration) {
	go func() {
		time.Sleep(duration)
		d.Stop()
	}()
}

func (d *defaultDiskOccupier) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := os.Remove(d.filePath); err == nil {
		d.isRunning = false
	}
}
