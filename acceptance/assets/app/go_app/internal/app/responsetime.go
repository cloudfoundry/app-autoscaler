package app

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

//counterfeiter:generate . TimeWaster
type TimeWaster interface {
	Sleep(sleepTime time.Duration)
}

type Sleeper struct{}

var _ TimeWaster = Sleeper{}

func ResponseTimeTests(logger *slog.Logger, mux *http.ServeMux, timeWaster TimeWaster) {
	mux.HandleFunc("GET /responsetime/slow/{delayInMS}", func(w http.ResponseWriter, r *http.Request) {
		var milliseconds int64
		var err error
		if milliseconds, err = strconv.ParseInt(r.PathValue("delayInMS"), 10, 64); err != nil {
			Errorf(logger, w, http.StatusBadRequest, "invalid milliseconds: %s", err.Error())
			return
		}
		duration := time.Duration(milliseconds) * time.Millisecond
		timeWaster.Sleep(duration)
		respondJSON(logger, w, JSONResponse{"duration": duration.String()})
	})

	mux.HandleFunc("GET /responsetime/fast", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(logger, w, JSONResponse{"fast": true})
	})
}

func (Sleeper) Sleep(sleepTime time.Duration) {
	time.Sleep(sleepTime)
}
