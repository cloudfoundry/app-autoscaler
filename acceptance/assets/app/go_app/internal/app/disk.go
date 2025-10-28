package app

import (
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func DiskTest(r *gin.RouterGroup, diskOccupier DiskOccupier) *gin.RouterGroup {
	r.GET("/:utilization/:minutes", func(c *gin.Context) {
		var utilisation int64
		var minutes int64
		var err error

		utilisation, err = strconv.ParseInt(c.Param("utilization"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, "invalid utilization: %s", err.Error())
			return
		}
		if minutes, err = strconv.ParseInt(c.Param("minutes"), 10, 64); err != nil {
			Error(c, http.StatusBadRequest, "invalid minutes: %s", err.Error())
			return
		}
		duration := time.Duration(minutes) * time.Minute
		spaceInMB := utilisation * 1000 * 1000
		if err = diskOccupier.Occupy(spaceInMB, duration); err != nil {
			Error(c, http.StatusInternalServerError, "error invoking occupation: %s", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"utilization": utilisation, "minutes": minutes})
	})

	r.GET("/close", func(c *gin.Context) {
		diskOccupier.Stop()
		c.String(http.StatusOK, "close disk test")
	})

	return r
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
