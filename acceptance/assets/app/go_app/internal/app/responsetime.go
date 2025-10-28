package app

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
)

//counterfeiter:generate . TimeWaster
type TimeWaster interface {
	Sleep(sleepTime time.Duration)
}

type Sleeper struct{}

var _ TimeWaster = Sleeper{}

func ResponseTimeTests(logger logr.Logger, r *gin.RouterGroup, timeWaster TimeWaster) *gin.RouterGroup {
	r.GET("/slow/:delayInMS", func(c *gin.Context) {
		var milliseconds int64
		var err error
		if milliseconds, err = strconv.ParseInt(c.Param("delayInMS"), 10, 64); err != nil {
			Error(c, http.StatusBadRequest, "invalid milliseconds: %s", err.Error())
			return
		}
		duration := time.Duration(milliseconds) * time.Millisecond
		timeWaster.Sleep(duration)
		c.JSON(http.StatusOK, gin.H{"duration": duration.String()})
	})

	r.GET("/fast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"fast": true})
	})
	return r
}

func (Sleeper) Sleep(sleepTime time.Duration) {
	time.Sleep(sleepTime)
}
